package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alvarorichard/Goanime/internal/ai"
	api "github.com/alvarorichard/Goanime/internal/api"
	"github.com/alvarorichard/Goanime/internal/models"
	"github.com/alvarorichard/Goanime/internal/player"
	"github.com/alvarorichard/Goanime/internal/scraper"
	"github.com/alvarorichard/Goanime/internal/tracking"
	"github.com/alvarorichard/Goanime/internal/util"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	maxSearchItems  = 100
	maxEpisodeItems = 400
)

type App struct {
	ctx context.Context

	manager    *scraper.ScraperManager
	httpClient *http.Client
	tracker    *tracking.LocalTracker

	mu            sync.RWMutex
	proxyBaseURL  string
	proxyServer   *http.Server
	proxyListener net.Listener

	catalogMu      sync.RWMutex
	catalogCache   []CatalogSection
	catalogExpiry  time.Time
	genreCacheMu   sync.RWMutex
	genreCache     map[string][]CatalogSection
	genreCacheTime map[string]time.Time

	movieCatalogMu     sync.RWMutex
	movieCatalogCache  []CatalogSection
	movieCatalogExpiry time.Time

	calendarMu     sync.RWMutex
	calendarCache  []CalendarDay
	calendarExpiry time.Time

	settings   AppSettings
	settingsMu sync.RWMutex

	watchedMarks map[string][]int
	watchedMu    sync.RWMutex

	// Custom lists
	customLists   customListsData
	customListsMu sync.RWMutex

	// Play queue (in-memory only)
	playQueue   []QueueEntry
	playQueueMu sync.RWMutex

	// Anime notes
	notes   map[string]AnimeNote
	notesMu sync.RWMutex

	// Anime library (AniList-based identity)
	library *animeLibrary

	// Bot system
	aiClient  *ai.Client
	relBot    *releasesBot
	recBot    *recommenderBot
	curBot    *curatorBot
	botCancel context.CancelFunc
}

func NewApp() *App {
	return &App{
		manager:    scraper.NewScraperManager(),
		httpClient: util.GetFastClient(),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if err := a.startProxyServer(); err != nil {
		log.Printf("proxy bootstrap failed: %v", err)
	}
	if p := desktopTrackerPath(); p != "" {
		if t := tracking.NewLocalTracker(p); t != nil {
			a.tracker = t
		}
	}

	// Load user settings, watched marks, custom lists, notes, and anime library
	a.loadSettings()
	a.loadWatchedMarks()
	a.loadCustomLists()
	a.loadNotes()
	a.library = newAnimeLibrary()
	a.library.loadLibrary()

	// Initialize bot system
	a.aiClient = ai.NewClient()
	a.relBot = newReleasesBot(ctx)
	a.relBot.notifyFn = func(title, body string) {
		if a.GetSettings().NotificationsEnabled {
			a.SendDesktopNotification(title, body)
		}
	}
	a.recBot = newRecommenderBot(a.aiClient, a.GetWatchProgress)
	a.curBot = newCuratorBot(a.aiClient, a.relBot)

	// Start background bots
	botCtx, botCancel := context.WithCancel(context.Background())
	a.botCancel = botCancel
	a.relBot.start()

	// Refresh AI bots in background (non-blocking)
	go func() {
		// Small delay to let the app start
		time.Sleep(5 * time.Second)
		a.recBot.refresh(botCtx)
		a.curBot.refresh(botCtx)
	}()
}

func desktopTrackerPath() string {
	if runtime.GOOS == "windows" {
		if d := os.Getenv("LOCALAPPDATA"); d != "" {
			return filepath.Join(d, "GoAnime", "tracking", "progress.db")
		}
	}
	if h := os.Getenv("HOME"); h != "" {
		return filepath.Join(h, ".local", "goanime", "tracking", "progress.db")
	}
	return ""
}

func (a *App) GetWatchProgress() []WatchProgressEntry {
	if a.tracker == nil {
		return []WatchProgressEntry{}
	}
	items, err := a.tracker.GetAllAnime()
	if err != nil {
		return []WatchProgressEntry{}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].LastUpdated.After(items[j].LastUpdated)
	})
	entries := make([]WatchProgressEntry, 0, len(items))
	for _, item := range items {
		title := strings.TrimSpace(item.Title)
		if title == "" || strings.EqualFold(title, "No title") {
			continue
		}
		entries = append(entries, WatchProgressEntry{
			AllanimeID:      item.AllanimeID,
			Title:           title,
			EpisodeNumber:   item.EpisodeNumber,
			PlaybackTime:    item.PlaybackTime,
			Duration:        item.Duration,
			ProgressPercent: progressPercent(item.PlaybackTime, item.Duration),
			MediaType:       item.MediaType,
			LastUpdated:     item.LastUpdated.Format(time.RFC3339),
		})
	}
	return entries
}

func (a *App) UpdateWatchProgress(req UpdateWatchProgressRequest) error {
	if a.tracker == nil {
		return fmt.Errorf("tracker not available")
	}

	allanimeID := strings.TrimSpace(req.AllanimeID)
	if allanimeID == "" {
		return fmt.Errorf("progress id is required")
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		return fmt.Errorf("title is required")
	}

	if req.Duration <= 0 {
		return fmt.Errorf("duration must be greater than 0")
	}

	episodeNumber := req.EpisodeNumber
	if episodeNumber < 1 {
		episodeNumber = 1
	}

	playbackTime := req.PlaybackTime
	if playbackTime < 0 {
		playbackTime = 0
	}
	if playbackTime > req.Duration {
		playbackTime = req.Duration
	}

	return a.tracker.UpdateProgress(tracking.Anime{
		AllanimeID:    allanimeID,
		Title:         title,
		EpisodeNumber: episodeNumber,
		PlaybackTime:  playbackTime,
		Duration:      req.Duration,
		MediaType:     strings.TrimSpace(req.MediaType),
		LastUpdated:   time.Now(),
	})
}

func (a *App) DownloadEpisode(req DownloadEpisodeRequest) (*DownloadEpisodeResponse, error) {
	stream, err := a.GetStream(StreamRequest{
		Media:         req.Media,
		EpisodeURL:    req.EpisodeURL,
		EpisodeNumber: req.EpisodeNumber,
		Mode:          req.Mode,
		Quality:       req.Quality,
	})
	if err != nil {
		return nil, err
	}

	streamURL := strings.TrimSpace(stream.StreamURL)
	if streamURL == "" {
		streamURL = strings.TrimSpace(stream.ProxyURL)
	}
	if streamURL == "" {
		return nil, fmt.Errorf("stream URL is empty")
	}

	episodeNum := parseEpisodeNum(req.EpisodeNumber)
	if episodeNum < 1 {
		episodeNum = 1
	}

	downloadURL := strings.TrimSpace(req.Media.URL)
	downloadSource := strings.TrimSpace(req.Media.Source)
	if strings.TrimSpace(stream.ResolvedURL) != "" {
		downloadURL = strings.TrimSpace(stream.ResolvedURL)
	}
	if strings.TrimSpace(stream.ResolvedSource) != "" {
		downloadSource = strings.TrimSpace(stream.ResolvedSource)
	}

	filePath, err := player.DownloadEpisodeForDesktop(streamURL, downloadURL, req.Media.Name, episodeNum)
	if err != nil {
		recordSourceFailure(req.Media.GroupKey, "download", downloadSource, err)
		return nil, err
	}
	clearSourceFailure(req.Media.GroupKey, "download", downloadSource)

	return &DownloadEpisodeResponse{
		FilePath: filePath,
		Message:  "Download concluido",
	}, nil
}

func (a *App) shutdown(_ context.Context) {
	// Stop bots
	if a.botCancel != nil {
		a.botCancel()
	}
	if a.relBot != nil {
		a.relBot.stop()
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.proxyServer != nil {
		_ = a.proxyServer.Close()
		a.proxyServer = nil
	}
	if a.proxyListener != nil {
		_ = a.proxyListener.Close()
		a.proxyListener = nil
	}
	a.proxyBaseURL = ""
}

func (a *App) GetProxyBaseURL() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.proxyBaseURL
}

func (a *App) GetSearchHistory() []HistoryEntry {
	names := util.LoadSearchHistory()
	entries := make([]HistoryEntry, len(names))
	for i, n := range names {
		entries[i] = HistoryEntry{Name: n}
	}
	return entries
}

// GetBotStatus returns the current status of all bots.
func (a *App) GetBotStatus() BotStatus {
	status := BotStatus{}

	if a.aiClient != nil {
		status.AIOnline = a.aiClient.IsAvailable()
		if status.AIOnline {
			status.AIModel = a.aiClient.ModelName()
		}
	}

	if a.relBot != nil {
		releases := a.relBot.getReleases()
		status.ReleasesCount = len(releases)
		status.NewReleases = a.relBot.getNewCount()
		a.relBot.mu.RLock()
		if !a.relBot.lastCheck.IsZero() {
			status.LastCheck = a.relBot.lastCheck.Format(time.RFC3339)
		}
		a.relBot.mu.RUnlock()
	}

	if a.recBot != nil {
		recs := a.recBot.getRecs()
		status.RecsAvailable = a.recBot.isAvailable()
		status.RecsCount = len(recs)
	}

	if a.curBot != nil {
		status.CuratedCount = len(a.curBot.getCurated())
	}

	return status
}

// GetNyaaReleases returns the latest PT-BR releases from Nyaa.
func (a *App) GetNyaaReleases() []NyaaRelease {
	if a.relBot == nil {
		return []NyaaRelease{}
	}
	return a.relBot.getReleases()
}

// ClearNewReleases resets the new releases counter.
func (a *App) ClearNewReleases() {
	if a.relBot != nil {
		a.relBot.clearNewCount()
	}
}

// GetAIRecommendations returns personalized anime recommendations.
func (a *App) GetAIRecommendations() []AIRecommendation {
	if a.recBot == nil {
		return []AIRecommendation{}
	}
	return a.recBot.getRecs()
}

// RefreshRecommendations forces a refresh of AI recommendations.
func (a *App) RefreshRecommendations() []AIRecommendation {
	if a.recBot == nil || a.aiClient == nil {
		return []AIRecommendation{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	a.recBot.refresh(ctx)
	return a.recBot.getRecs()
}

// GetCuratedReleases returns AI-curated PT-BR releases.
func (a *App) GetCuratedReleases() []CuratedRelease {
	if a.curBot == nil {
		return []CuratedRelease{}
	}
	return a.curBot.getCurated()
}

// RefreshCuratedReleases forces a refresh of curated releases.
func (a *App) RefreshCuratedReleases() []CuratedRelease {
	if a.curBot == nil || a.aiClient == nil {
		return []CuratedRelease{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	a.curBot.refresh(ctx)
	return a.curBot.getCurated()
}

func (a *App) SearchMedia(query string, source string, mediaType string) ([]MediaResult, error) {
	query = strings.TrimSpace(query)
	if len(query) < 2 {
		return nil, fmt.Errorf("query must have at least 2 characters")
	}

	sourceValue := strings.ToLower(strings.TrimSpace(source))
	mediaTypeValue := strings.ToLower(strings.TrimSpace(mediaType))

	sourceType, sourceErr := parseSource(sourceValue)
	if sourceErr != "" {
		return nil, fmt.Errorf("%s", sourceErr)
	}

	// ── Fast direct search (like the terminal version) ──
	// Search scrapers directly with the user's query — no AniList/Jikan
	// pre-resolution. This makes search nearly instant.
	results, err := a.manager.SearchAnime(query, sourceType)
	if err != nil {
		return nil, err
	}

	// Filter out English-only sources when no specific source was chosen
	if sourceType == nil {
		filtered := results[:0]
		for _, r := range results {
			src := strings.ToLower(strings.TrimSpace(r.Source))
			if src != "flixhq" && src != "allanime" {
				filtered = append(filtered, r)
			}
		}
		results = filtered
	}

	filtered := filterByType(results, mediaTypeValue)

	// Deduplicate by source+URL
	seen := make(map[string]bool, len(filtered))
	out := make([]MediaResult, 0, min(len(filtered), maxSearchItems))
	for _, item := range filtered {
		if item == nil {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(item.Source)) + "|" + strings.TrimSpace(item.URL)
		if key == "|" || seen[key] {
			continue
		}
		seen[key] = true
		if len(out) >= maxSearchItems {
			break
		}
		out = append(out, MediaResult{
			Name:      cleanDisplayName(item.Name),
			URL:       item.URL,
			ImageURL:  item.ImageURL,
			Source:    item.Source,
			MediaType: normalizeMediaType(item),
			Year:      item.Year,
			Score:     item.Rating,
		})
	}

	return out, nil
}

func (a *App) emitPartialSearchResults(results []MediaResult) {
	if len(results) == 0 {
		return
	}
	wailsruntime.EventsEmit(a.ctx, "search:partial", results)
}

func fetchAniListCovers(client *http.Client, query string) map[string]string {
	type anilistSearchResp struct {
		Data struct {
			Page struct {
				Media []struct {
					Title struct {
						Romaji  string `json:"romaji"`
						English string `json:"english"`
					} `json:"title"`
					CoverImage struct {
						Large string `json:"large"`
					} `json:"coverImage"`
				} `json:"media"`
			} `json:"Page"`
		} `json:"data"`
	}

	const searchQuery = `
query ($search: String) {
  Page(perPage: 10) {
    media(search: $search, type: ANIME, isAdult: false) {
      title { romaji english }
      coverImage { large }
    }
  }
}`

	body, err := json.Marshal(map[string]interface{}{
		"query":     searchQuery,
		"variables": map[string]interface{}{"search": query},
	})
	if err != nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, anilistEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	util.GetAniListLimiter().Wait()
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var page anilistSearchResp
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil
	}

	covers := make(map[string]string)
	for _, m := range page.Data.Page.Media {
		img := m.CoverImage.Large
		if img == "" {
			continue
		}
		if t := strings.ToLower(m.Title.English); t != "" {
			covers[t] = img
		}
		if t := strings.ToLower(m.Title.Romaji); t != "" {
			covers[t] = img
		}
	}
	return covers
}

type jikanMeta struct {
	ImageURL    string
	Score       float64
	Description string
	Genres      []string
}

func fetchJikanMeta(client *http.Client, query string) map[string]jikanMeta {
	type jikanResp struct {
		Data []struct {
			Title        string `json:"title"`
			TitleEnglish string `json:"title_english"`
			Images       struct {
				Jpg struct {
					LargeImageURL string `json:"large_image_url"`
					ImageURL      string `json:"image_url"`
				} `json:"jpg"`
			} `json:"images"`
			Score    float64 `json:"score"`
			Synopsis string  `json:"synopsis"`
			Genres   []struct {
				Name string `json:"name"`
			} `json:"genres"`
		} `json:"data"`
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	reqURL := "https://api.jikan.moe/v4/anime?q=" + url.QueryEscape(query) + "&limit=10&sfw=true"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Accept", "application/json")

	util.GetJikanLimiter().Wait()
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var result jikanResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil
	}

	meta := make(map[string]jikanMeta)
	for _, item := range result.Data {
		img := item.Images.Jpg.LargeImageURL
		if img == "" {
			img = item.Images.Jpg.ImageURL
		}
		genres := make([]string, 0, len(item.Genres))
		for _, g := range item.Genres {
			genres = append(genres, g.Name)
		}
		m := jikanMeta{
			ImageURL:    img,
			Score:       item.Score,
			Description: item.Synopsis,
			Genres:      genres,
		}
		if t := strings.ToLower(strings.TrimSpace(item.Title)); t != "" {
			meta[t] = m
		}
		if t := strings.ToLower(strings.TrimSpace(item.TitleEnglish)); t != "" {
			meta[t] = m
		}
	}
	return meta
}

func applyJikanMeta(results []MediaResult, meta map[string]jikanMeta) {
	for i := range results {
		nameLower := strings.ToLower(results[i].Name)
		keyLen := len(nameLower)
		if keyLen > 20 {
			keyLen = 20
		}
		for title, m := range meta {
			titleLen := len(title)
			if titleLen > 20 {
				titleLen = 20
			}
			if strings.HasPrefix(nameLower, title[:titleLen]) || strings.HasPrefix(title, nameLower[:keyLen]) {
				if results[i].ImageURL == "" && m.ImageURL != "" {
					results[i].ImageURL = m.ImageURL
				}
				if results[i].Score == 0 && m.Score > 0 {
					results[i].Score = m.Score
				}
				if results[i].Description == "" && m.Description != "" {
					results[i].Description = m.Description
				}
				if len(results[i].Genres) == 0 && len(m.Genres) > 0 {
					results[i].Genres = m.Genres
				}
				break
			}
		}
	}
}

var jikanRelationPTBR = map[string]string{
	"Sequel":              "Continuação",
	"Prequel":             "Préquel",
	"Alternative version": "Versão alternativa",
	"Alternative setting": "Cenário alternativo",
	"Side story":          "História paralela",
	"Parent story":        "História principal",
	"Summary":             "Resumo",
	"Spin-off":            "Spin-off",
	"Full story":          "História completa",
	"Other":               "Relacionado",
}

var jikanRelationPriority = []string{
	"Prequel", "Sequel", "Parent story", "Full story",
	"Side story", "Alternative version", "Alternative setting",
	"Spin-off", "Summary", "Other",
}

func jikanSearchMALID(client *http.Client, query string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	reqURL := "https://api.jikan.moe/v4/anime?q=" + url.QueryEscape(query) + "&limit=3&sfw=true"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0
	}
	req.Header.Set("Accept", "application/json")

	util.GetJikanLimiter().Wait()
	resp, err := client.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	var result struct {
		Data []struct {
			MalID int    `json:"mal_id"`
			Title string `json:"title"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || len(result.Data) == 0 {
		return 0
	}

	queryLower := strings.ToLower(strings.TrimSpace(query))
	for _, item := range result.Data {
		if strings.ToLower(item.Title) == queryLower {
			return item.MalID
		}
	}
	return result.Data[0].MalID
}

func jikanFetchRelations(client *http.Client, malID int) []RelatedAnime {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	reqURL := fmt.Sprintf("https://api.jikan.moe/v4/anime/%d/relations", malID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Accept", "application/json")

	util.GetJikanLimiter().Wait()
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var result struct {
		Data []struct {
			Relation string `json:"relation"`
			Entry    []struct {
				MalID int    `json:"mal_id"`
				Type  string `json:"type"`
				Name  string `json:"name"`
			} `json:"entry"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil
	}

	seen := make(map[int]bool)
	relByPriority := make(map[string][]RelatedAnime)
	for _, r := range result.Data {
		label, ok := jikanRelationPTBR[r.Relation]
		if !ok {
			continue
		}
		for _, e := range r.Entry {
			if e.Type != "anime" || seen[e.MalID] {
				continue
			}
			seen[e.MalID] = true
			relByPriority[r.Relation] = append(relByPriority[r.Relation], RelatedAnime{
				MalID:    e.MalID,
				Name:     e.Name,
				Relation: label,
			})
		}
	}

	var relations []RelatedAnime
	for _, relType := range jikanRelationPriority {
		relations = append(relations, relByPriority[relType]...)
	}
	if len(relations) > 8 {
		relations = relations[:8]
	}
	return relations
}

func jikanFetchAnimeImage(client *http.Client, malID int) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	reqURL := fmt.Sprintf("https://api.jikan.moe/v4/anime/%d", malID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Accept", "application/json")

	util.GetJikanLimiter().Wait()
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Images struct {
				Jpg struct {
					LargeImageURL string `json:"large_image_url"`
					ImageURL      string `json:"image_url"`
				} `json:"jpg"`
			} `json:"images"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ""
	}
	if img := result.Data.Images.Jpg.LargeImageURL; img != "" {
		return img
	}
	return result.Data.Images.Jpg.ImageURL
}

func (a *App) GetRelatedAnime(title string) []RelatedAnime {
	cleaned := strings.TrimSpace(title)
	// Remove language tags like the frontend does
	for _, tag := range []string{"[English]", "[Portuguese]", "[Português]"} {
		cleaned = strings.ReplaceAll(cleaned, tag, "")
	}
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return []RelatedAnime{}
	}

	malID := jikanSearchMALID(a.httpClient, cleaned)
	if malID == 0 {
		return []RelatedAnime{}
	}

	// Small delay to respect Jikan rate limit after search request
	time.Sleep(350 * time.Millisecond)

	relations := jikanFetchRelations(a.httpClient, malID)
	if len(relations) == 0 {
		return []RelatedAnime{}
	}

	// Fetch images for all relations in parallel with staggered start
	type imageResult struct {
		index int
		url   string
	}
	ch := make(chan imageResult, len(relations))
	var imgWg sync.WaitGroup
	for i, rel := range relations {
		i, rel := i, rel
		imgWg.Add(1)
		go func() {
			defer imgWg.Done()
			time.Sleep(time.Duration(i) * 400 * time.Millisecond)
			ch <- imageResult{i, jikanFetchAnimeImage(a.httpClient, rel.MalID)}
		}()
	}

	// Close channel once all goroutines complete
	go func() {
		imgWg.Wait()
		close(ch)
	}()

	deadline := time.After(8 * time.Second)
	received := 0
	for received < len(relations) {
		select {
		case r, ok := <-ch:
			if !ok {
				received = len(relations)
				break
			}
			relations[r.index].ImageURL = r.url
			received++
		case <-deadline:
			received = len(relations)
		}
	}

	return relations
}

// GetSkipTimes returns opening and ending skip timestamps for an episode.
func (a *App) GetSkipTimes(malID int, episodeNum int) (*SkipTimesResult, error) {
	if malID <= 0 || episodeNum <= 0 {
		return &SkipTimesResult{Found: false}, nil
	}

	ep := &models.Episode{}
	err := api.GetAndParseAniSkipData(malID, episodeNum, ep)
	if err != nil {
		// Not found is not a hard error for the frontend
		return &SkipTimesResult{Found: false}, nil
	}

	return &SkipTimesResult{
		OpStart: float64(ep.SkipTimes.Op.Start),
		OpEnd:   float64(ep.SkipTimes.Op.End),
		EdStart: float64(ep.SkipTimes.Ed.Start),
		EdEnd:   float64(ep.SkipTimes.Ed.End),
		Found:   ep.SkipTimes.Op.End > 0 || ep.SkipTimes.Ed.End > 0,
	}, nil
}

func (a *App) emitSearchCoverUpdates(ctx context.Context, query string, source string, mediaType string, results []MediaResult) {
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	type anilistDone struct{ covers map[string]string }
	type jikanDone struct{ meta map[string]jikanMeta }

	anilistCh := make(chan anilistDone, 1)
	jikanCh := make(chan jikanDone, 1)

	go func() { anilistCh <- anilistDone{fetchAniListCovers(a.httpClient, query)} }()
	go func() { jikanCh <- jikanDone{fetchJikanMeta(a.httpClient, query)} }()

	select {
	case <-ctx.Done():
		return
	case al := <-anilistCh:
		if len(al.covers) > 0 {
			applyAniListCovers(results, al.covers)
		}
	}

	select {
	case <-ctx.Done():
		return
	case jk := <-jikanCh:
		if len(jk.meta) > 0 {
			applyJikanMeta(results, jk.meta)
		}
	}

	updated := make([]MediaResult, 0, len(results))
	for _, item := range results {
		if strings.TrimSpace(item.ImageURL) != "" || item.Score > 0 || item.Description != "" || len(item.Genres) > 0 {
			updated = append(updated, item)
		}
	}
	if len(updated) == 0 {
		return
	}

	wailsruntime.EventsEmit(a.ctx, "search:covers", SearchCoversEvent{
		Query:     query,
		Source:    source,
		MediaType: mediaType,
		Results:   updated,
	})
}

func applyAniListCovers(results []MediaResult, covers map[string]string) {
	for i := range results {
		if results[i].ImageURL != "" {
			continue
		}
		nameLower := strings.ToLower(results[i].Name)
		keyLen := len(nameLower)
		if keyLen > 15 {
			keyLen = 15
		}
		for title, img := range covers {
			if strings.HasPrefix(nameLower, title) || strings.HasPrefix(title, nameLower[:keyLen]) {
				results[i].ImageURL = img
				break
			}
		}
	}
}

func (a *App) GetEpisodes(req MediaRequest) (*EpisodesResponse, error) {
	return a.tryGetEpisodes(req)
}

func (a *App) GetStream(req StreamRequest) (*StreamResponse, error) {
	return a.tryGetStream(req)
}

func (a *App) startProxyServer() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/proxy", a.handleProxy)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}

	a.mu.Lock()
	a.proxyListener = listener
	a.proxyBaseURL = "http://" + listener.Addr().String()
	a.proxyServer = &http.Server{Handler: mux}
	a.mu.Unlock()

	go func() {
		if serveErr := a.proxyServer.Serve(listener); serveErr != nil && serveErr != http.ErrServerClosed {
			log.Printf("proxy server stopped with error: %v", serveErr)
		}
	}()

	return nil
}

func (a *App) handleProxy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	targetRaw := strings.TrimSpace(r.URL.Query().Get("u"))
	if targetRaw == "" {
		http.Error(w, "missing target", http.StatusBadRequest)
		return
	}

	targetURL, err := url.Parse(targetRaw)
	if err != nil || targetURL.Host == "" {
		http.Error(w, "invalid target", http.StatusBadRequest)
		return
	}
	if targetURL.Scheme != "http" && targetURL.Scheme != "https" {
		http.Error(w, "unsupported scheme", http.StatusBadRequest)
		return
	}

	upstreamReq, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, targetURL.String(), nil)
	upstreamReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0 Safari/537.36")
	upstreamReq.Header.Set("Accept", "*/*")
	upstreamReq.Header.Set("Referer", targetURL.Scheme+"://"+targetURL.Host+"/")
	if rangeHeader := r.Header.Get("Range"); rangeHeader != "" {
		upstreamReq.Header.Set("Range", rangeHeader)
	}

	resp, reqErr := a.httpClient.Do(upstreamReq)
	if reqErr != nil {
		http.Error(w, reqErr.Error(), http.StatusBadGateway)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	contentType := resp.Header.Get("Content-Type")
	isPlaylist := strings.Contains(strings.ToLower(contentType), "mpegurl") || strings.HasSuffix(strings.ToLower(targetURL.Path), ".m3u8")
	if isPlaylist {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			http.Error(w, readErr.Error(), http.StatusBadGateway)
			return
		}
		rewritten := a.rewriteM3U8(string(body), targetURL)
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.WriteHeader(resp.StatusCode)
		_, _ = io.WriteString(w, rewritten)
		return
	}

	copyProxyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (a *App) rewriteM3U8(content string, baseURL *url.URL) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			if strings.Contains(trimmed, "URI=\"") {
				lines[i] = a.rewriteM3U8TagURI(line, baseURL)
			}
			continue
		}

		resolved := resolveReference(baseURL, trimmed)
		lines[i] = a.toProxyURL(resolved)
	}
	return strings.Join(lines, "\n")
}

func (a *App) rewriteM3U8TagURI(line string, baseURL *url.URL) string {
	start := strings.Index(line, "URI=\"")
	if start < 0 {
		return line
	}
	valueStart := start + len("URI=\"")
	valueEnd := strings.Index(line[valueStart:], "\"")
	if valueEnd < 0 {
		return line
	}
	valueEnd = valueStart + valueEnd

	rawURI := line[valueStart:valueEnd]
	resolved := resolveReference(baseURL, rawURI)
	proxied := a.toProxyURL(resolved)

	return line[:valueStart] + proxied + line[valueEnd:]
}

func (a *App) toProxyURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	a.mu.RLock()
	base := a.proxyBaseURL
	a.mu.RUnlock()
	if base == "" {
		return raw
	}
	return base + "/proxy?u=" + url.QueryEscape(raw)
}
func resolveReference(baseURL *url.URL, raw string) string {
	if raw == "" {
		return raw
	}
	candidate, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if candidate.IsAbs() {
		return candidate.String()
	}
	return baseURL.ResolveReference(candidate).String()
}

func copyProxyHeaders(dst http.Header, src http.Header) {
	copyHeader := func(name string) {
		if val := src.Get(name); val != "" {
			dst.Set(name, val)
		}
	}

	copyHeader("Content-Type")
	copyHeader("Content-Length")
	copyHeader("Content-Range")
	copyHeader("Accept-Ranges")
	copyHeader("Cache-Control")
	copyHeader("ETag")
	copyHeader("Last-Modified")
}

func parseSource(sourceArg string) (*scraper.ScraperType, string) {
	if sourceArg == "" || sourceArg == "all" {
		return nil, ""
	}

	switch sourceArg {
	case "allanime":
		t := scraper.AllAnimeType
		return &t, ""
	case "animefire":
		t := scraper.AnimefireType
		return &t, ""
	case "flixhq":
		t := scraper.FlixHQType
		return &t, ""
	case "animesonlinecc":
		t := scraper.AnimesOnlineccType
		return &t, ""
	case "anroll":
		t := scraper.AnrollType
		return &t, ""
	case "bakashi":
		t := scraper.BakashiType
		return &t, ""
	default:
		return nil, "invalid source; use: all, allanime, animefire, flixhq, animesonlinecc, anroll, bakashi"
	}
}

func filterByType(items []*models.Anime, typeArg string) []*models.Anime {
	if typeArg == "" || typeArg == "all" {
		return items
	}

	out := make([]*models.Anime, 0, len(items))
	for _, item := range items {
		if normalizeMediaType(item) == typeArg {
			out = append(out, item)
		}
	}
	return out
}

func normalizeMediaType(item *models.Anime) string {
	switch item.MediaType {
	case models.MediaTypeMovie:
		return "movie"
	case models.MediaTypeTV:
		return "tv"
	case models.MediaTypeAnime:
		return "anime"
	}

	if strings.EqualFold(item.Source, "FlixHQ") {
		return "movie"
	}
	return "anime"
}

func normalizeSource(source string) string {
	lower := strings.ToLower(strings.TrimSpace(source))
	switch {
	case strings.Contains(lower, "allanime"):
		return "AllAnime"
	case strings.Contains(lower, "animefire"):
		return "Animefire.io"
	case strings.Contains(lower, "flixhq"):
		return "FlixHQ"
	case strings.Contains(lower, "bakashi"):
		return "Bakashi"
	case strings.Contains(lower, "animedrive"):
		return "AnimeDrive"
	case strings.Contains(lower, "animesonlinecc"):
		return "AnimesOnlineCC"
	case strings.Contains(lower, "anroll"):
		return "Anroll"
	default:
		return source
	}
}

func parseMediaType(typeArg string) models.MediaType {
	switch strings.ToLower(strings.TrimSpace(typeArg)) {
	case "movie":
		return models.MediaTypeMovie
	case "tv":
		return models.MediaTypeTV
	default:
		return models.MediaTypeAnime
	}
}

func isAllAnimeMedia(anime *models.Anime) bool {
	if anime == nil {
		return false
	}
	if strings.EqualFold(anime.Source, "AllAnime") {
		return true
	}
	if strings.Contains(strings.ToLower(anime.URL), "allanime") {
		return true
	}
	if len(anime.URL) > 5 && len(anime.URL) < 30 && !strings.Contains(anime.URL, "http") {
		return true
	}
	return false
}

func extractAllAnimeID(value string) string {
	if value == "" {
		return ""
	}
	if !strings.Contains(value, "http") {
		return value
	}
	if !strings.Contains(strings.ToLower(value), "allanime") {
		return ""
	}

	parts := strings.Split(value, "/")
	for _, p := range parts {
		part := strings.TrimSpace(p)
		if len(part) > 5 && len(part) < 30 && !strings.Contains(part, ".") {
			return part
		}
	}
	return ""
}

func detectContentType(rawURL string) string {
	lower := strings.ToLower(rawURL)
	if strings.Contains(lower, ".m3u8") {
		return "application/vnd.apple.mpegurl"
	}
	if strings.Contains(lower, ".mp4") {
		return "video/mp4"
	}
	return "video/*"
}

func parseEpisodeNum(value string) int {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 1
	}
	if n, err := strconv.Atoi(trimmed); err == nil {
		return max(1, n)
	}

	digits := make([]rune, 0, 4)
	for _, r := range trimmed {
		if r >= '0' && r <= '9' {
			digits = append(digits, r)
		}
	}
	if len(digits) > 0 {
		if n, err := strconv.Atoi(string(digits)); err == nil {
			return max(1, n)
		}
	}
	return 1
}

func progressPercent(playbackTime int, duration int) float64 {
	if duration <= 0 {
		return 0
	}
	percent := (float64(playbackTime) / float64(duration)) * 100
	if percent < 0 {
		return 0
	}
	if percent > 100 {
		return 100
	}
	return percent
}
