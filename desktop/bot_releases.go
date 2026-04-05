package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	nyaaRSSURL           = "https://nyaa.si/?page=rss&q=pt-br&c=1_2&f=0"
	releaseCheckInterval = 30 * time.Minute
	maxStoredReleases    = 200
)

type nyaaRSSFeed struct {
	Channel struct {
		Items []nyaaRSSItem `xml:"item"`
	} `xml:"channel"`
}

type nyaaRSSItem struct {
	Title   string `xml:"title"`
	Link    string `xml:"link"`
	PubDate string `xml:"pubDate"`
	GUID    string `xml:"guid"`
	Size    string `xml:"size"`
	Seeders int    `xml:"seeders"`
}

type releaseStore struct {
	Releases  []NyaaRelease   `json:"releases"`
	SeenGUIDs map[string]bool `json:"seenGuids"`
}

type releasesBot struct {
	ctx        context.Context
	cancel     context.CancelFunc
	httpClient *http.Client

	mu        sync.RWMutex
	releases  []NyaaRelease
	newCount  int
	lastCheck time.Time

	appCtx   context.Context                    // Wails context for events
	notifyFn func(title string, body string)     // optional notification callback
}

func newReleasesBot(appCtx context.Context) *releasesBot {
	ctx, cancel := context.WithCancel(context.Background())
	return &releasesBot{
		ctx:        ctx,
		cancel:     cancel,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		appCtx:     appCtx,
	}
}

func (b *releasesBot) start() {
	// Initial fetch
	go b.fetchReleases()

	// Periodic check
	go func() {
		ticker := time.NewTicker(releaseCheckInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				b.fetchReleases()
			case <-b.ctx.Done():
				return
			}
		}
	}()
}

func (b *releasesBot) stop() {
	b.cancel()
}

func (b *releasesBot) fetchReleases() {
	ctx, cancel := context.WithTimeout(b.ctx, 20*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, nyaaRSSURL, nil)
	if err != nil {
		log.Printf("[bot:releases] request error: %v", err)
		return
	}
	req.Header.Set("User-Agent", "GoAnime/1.0")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		log.Printf("[bot:releases] fetch error: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[bot:releases] RSS returned %d", resp.StatusCode)
		return
	}

	var feed nyaaRSSFeed
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		log.Printf("[bot:releases] parse error: %v", err)
		return
	}

	// Load seen GUIDs
	store := b.loadStore()

	var newReleases []NyaaRelease
	for _, item := range feed.Channel.Items {
		if store.SeenGUIDs[item.GUID] {
			continue
		}

		pubDate, _ := time.Parse(time.RFC1123Z, item.PubDate)
		if pubDate.IsZero() {
			pubDate, _ = time.Parse(time.RFC1123, item.PubDate)
		}

		release := NyaaRelease{
			Title:   strings.TrimSpace(item.Title),
			Link:    strings.TrimSpace(item.Link),
			Size:    strings.TrimSpace(item.Size),
			Date:    pubDate,
			Seeders: item.Seeders,
			IsNew:   true,
		}

		store.SeenGUIDs[item.GUID] = true
		newReleases = append(newReleases, release)
	}

	b.mu.Lock()
	// Mark old releases as not new
	for i := range b.releases {
		b.releases[i].IsNew = false
	}
	// Prepend new ones
	b.releases = append(newReleases, b.releases...)
	if len(b.releases) > maxStoredReleases {
		b.releases = b.releases[:maxStoredReleases]
	}
	b.newCount = len(newReleases)
	b.lastCheck = time.Now()

	// Save store
	store.Releases = b.releases
	b.mu.Unlock()

	b.saveStore(store)

	// Emit event to frontend
	if len(newReleases) > 0 && b.appCtx != nil {
		wailsruntime.EventsEmit(b.appCtx, "bot:newReleases", map[string]interface{}{
			"count":  len(newReleases),
			"titles": releaseTitles(newReleases, 5),
		})

		// Send desktop notification
		if b.notifyFn != nil {
			title := fmt.Sprintf("%d novos lançamentos PT-BR", len(newReleases))
			body := strings.Join(releaseTitles(newReleases, 3), "\n")
			b.notifyFn(title, body)
		}
	}

	log.Printf("[bot:releases] checked: %d new, %d total", len(newReleases), len(b.releases))
}

func releaseTitles(releases []NyaaRelease, max int) []string {
	titles := make([]string, 0, max)
	for i, r := range releases {
		if i >= max {
			break
		}
		titles = append(titles, r.Title)
	}
	return titles
}

func (b *releasesBot) getReleases() []NyaaRelease {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]NyaaRelease, len(b.releases))
	copy(out, b.releases)
	return out
}

func (b *releasesBot) getNewCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.newCount
}

func (b *releasesBot) clearNewCount() {
	b.mu.Lock()
	b.newCount = 0
	for i := range b.releases {
		b.releases[i].IsNew = false
	}
	b.mu.Unlock()
}

func releasesStorePath() string {
	if runtime.GOOS == "windows" {
		if d := os.Getenv("LOCALAPPDATA"); d != "" {
			return filepath.Join(d, "GoAnime", "bot_releases.json")
		}
	}
	if h := os.Getenv("HOME"); h != "" {
		return filepath.Join(h, ".local", "goanime", "bot_releases.json")
	}
	return ""
}

func (b *releasesBot) loadStore() releaseStore {
	store := releaseStore{SeenGUIDs: make(map[string]bool)}
	p := releasesStorePath()
	if p == "" {
		return store
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return store
	}
	_ = json.Unmarshal(data, &store)
	if store.SeenGUIDs == nil {
		store.SeenGUIDs = make(map[string]bool)
	}

	b.mu.Lock()
	b.releases = store.Releases
	// Sort by date descending
	sort.Slice(b.releases, func(i, j int) bool {
		return b.releases[i].Date.After(b.releases[j].Date)
	})
	b.mu.Unlock()

	return store
}

func (b *releasesBot) saveStore(store releaseStore) {
	p := releasesStorePath()
	if p == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(p, data, 0o644)
}
