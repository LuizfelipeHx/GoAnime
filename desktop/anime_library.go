package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/alvarorichard/Goanime/internal/util"
)

// animeLibrary holds the in-memory anime library keyed by AniList ID.
type animeLibrary struct {
	mu      sync.RWMutex
	entries map[int]*AnimeLibraryEntry
}

func newAnimeLibrary() *animeLibrary {
	return &animeLibrary{
		entries: make(map[int]*AnimeLibraryEntry),
	}
}

func animeLibraryPath() string {
	if runtime.GOOS == "windows" {
		if d := os.Getenv("LOCALAPPDATA"); d != "" {
			return filepath.Join(d, "GoAnime", "anime_library.json")
		}
	}
	if h := os.Getenv("HOME"); h != "" {
		return filepath.Join(h, ".local", "goanime", "anime_library.json")
	}
	return ""
}

// loadLibrary reads the library from disk into memory.
func (lib *animeLibrary) loadLibrary() {
	lib.mu.Lock()
	defer lib.mu.Unlock()

	p := animeLibraryPath()
	if p == "" {
		return
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return
	}
	var loaded map[int]*AnimeLibraryEntry
	if err := json.Unmarshal(data, &loaded); err != nil {
		log.Printf("anime_library: failed to parse: %v", err)
		return
	}
	if loaded != nil {
		lib.entries = loaded
	}
}

// saveLibrary persists the library to disk.
func (lib *animeLibrary) saveLibrary() {
	lib.mu.RLock()
	data, err := json.MarshalIndent(lib.entries, "", "  ")
	lib.mu.RUnlock()
	if err != nil {
		log.Printf("anime_library: failed to marshal: %v", err)
		return
	}

	p := animeLibraryPath()
	if p == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		log.Printf("anime_library: failed to create dir: %v", err)
		return
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		log.Printf("anime_library: failed to write: %v", err)
	}
}

// getEntry returns a library entry by AniList ID, or nil if not found.
func (lib *animeLibrary) getEntry(anilistID int) *AnimeLibraryEntry {
	if anilistID <= 0 {
		return nil
	}
	lib.mu.RLock()
	defer lib.mu.RUnlock()
	entry := lib.entries[anilistID]
	if entry == nil {
		return nil
	}
	cp := *entry
	cp.Sources = append([]SourceMapping(nil), entry.Sources...)
	cp.Genres = append([]string(nil), entry.Genres...)
	return &cp
}

// getOrCreateEntry returns an existing entry or creates a new empty one.
func (lib *animeLibrary) getOrCreateEntry(anilistID int) *AnimeLibraryEntry {
	if anilistID <= 0 {
		return nil
	}
	lib.mu.Lock()
	defer lib.mu.Unlock()
	entry := lib.entries[anilistID]
	if entry != nil {
		return entry
	}
	entry = &AnimeLibraryEntry{
		AniListID:   anilistID,
		Sources:     []SourceMapping{},
		LastUpdated: time.Now().Format(time.RFC3339),
	}
	lib.entries[anilistID] = entry
	return entry
}

// addSourceMapping adds a source URL to an entry, deduplicating by source+URL.
func (lib *animeLibrary) addSourceMapping(anilistID int, mapping SourceMapping) {
	if anilistID <= 0 {
		return
	}
	mapping.Source = strings.TrimSpace(mapping.Source)
	mapping.URL = strings.TrimSpace(mapping.URL)
	if mapping.Source == "" || mapping.URL == "" {
		return
	}

	lib.mu.Lock()
	defer lib.mu.Unlock()

	entry := lib.entries[anilistID]
	if entry == nil {
		entry = &AnimeLibraryEntry{
			AniListID:   anilistID,
			Sources:     []SourceMapping{},
			LastUpdated: time.Now().Format(time.RFC3339),
		}
		lib.entries[anilistID] = entry
	}

	srcLower := strings.ToLower(mapping.Source)
	urlTrimmed := mapping.URL
	for _, existing := range entry.Sources {
		if strings.ToLower(existing.Source) == srcLower && existing.URL == urlTrimmed {
			return // already exists
		}
	}

	entry.Sources = append(entry.Sources, mapping)
	entry.LastUpdated = time.Now().Format(time.RFC3339)
}

// updateFromAniListData updates a library entry with metadata from an AniList response.
func (lib *animeLibrary) updateFromAniListData(anilistID int, data *aniListDetailEnvelope) {
	if anilistID <= 0 || data == nil || data.Data.Media.ID == 0 {
		return
	}
	media := data.Data.Media

	lib.mu.Lock()
	defer lib.mu.Unlock()

	entry := lib.entries[anilistID]
	if entry == nil {
		entry = &AnimeLibraryEntry{
			AniListID: anilistID,
			Sources:   []SourceMapping{},
		}
		lib.entries[anilistID] = entry
	}

	entry.MalID = media.IDMal
	entry.TitleRomaji = strings.TrimSpace(media.Title.Romaji)
	entry.TitleEnglish = strings.TrimSpace(media.Title.English)
	if entry.Title == "" {
		entry.Title = firstNonEmpty(media.Title.UserPreferred, media.Title.English, media.Title.Romaji)
	}
	if media.CoverImage.Large != "" {
		entry.CoverImage = media.CoverImage.Large
	}
	if media.BannerImage != "" {
		entry.BannerImage = media.BannerImage
	}
	if len(media.Genres) > 0 {
		entry.Genres = append([]string(nil), media.Genres...)
	}
	desc := strings.TrimSpace(media.Description)
	if desc != "" {
		// Strip any residual HTML tags
		entry.Description = htmlTagsRe.ReplaceAllString(desc, "")
	}
	if media.Episodes > 0 {
		entry.TotalEpisodes = media.Episodes
	}
	if media.AverageScore > 0 {
		entry.Score = float64(media.AverageScore) / 10.0
	}
	if media.Status != "" {
		entry.Status = media.Status
	}
	if media.Format != "" {
		entry.Format = media.Format
	}
	if media.SeasonYear > 0 {
		entry.Year = media.SeasonYear
	}
	entry.LastUpdated = time.Now().Format(time.RFC3339)
}

// lookupByTitle does a fuzzy lookup across all library entries.
func (lib *animeLibrary) lookupByTitle(title string) *AnimeLibraryEntry {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil
	}
	norm := normalizeSearchText(title)
	if norm == "" {
		return nil
	}

	lib.mu.RLock()
	defer lib.mu.RUnlock()

	var best *AnimeLibraryEntry
	bestScore := 0
	for _, entry := range lib.entries {
		candidates := []string{entry.Title, entry.TitleRomaji, entry.TitleEnglish}
		for _, c := range candidates {
			if c == "" {
				continue
			}
			score := titleSimilarityScore(norm, c)
			if score > bestScore {
				bestScore = score
				cp := *entry
				best = &cp
			}
		}
	}

	// Require a minimum match quality
	if bestScore < 700 {
		return nil
	}
	return best
}

// ─── AniList Detail Query ───

type aniListDetailEnvelope struct {
	Data struct {
		Media struct {
			ID    int `json:"id"`
			IDMal int `json:"idMal"`
			Title struct {
				Romaji        string `json:"romaji"`
				English       string `json:"english"`
				Native        string `json:"native"`
				UserPreferred string `json:"userPreferred"`
			} `json:"title"`
			CoverImage struct {
				Large string `json:"large"`
			} `json:"coverImage"`
			BannerImage  string   `json:"bannerImage"`
			Genres       []string `json:"genres"`
			Description  string   `json:"description"`
			Episodes     int      `json:"episodes"`
			AverageScore int      `json:"averageScore"`
			Status       string   `json:"status"`
			Format       string   `json:"format"`
			SeasonYear   int      `json:"seasonYear"`
		} `json:"Media"`
	} `json:"data"`
}

const aniListDetailQuery = `query ($id: Int) {
  Media(id: $id, type: ANIME) {
    id
    idMal
    title { romaji english native userPreferred }
    coverImage { large }
    bannerImage
    genres
    description(asHtml: false)
    episodes
    averageScore
    status
    format
    seasonYear
  }
}`

// fetchAniListDetail fetches full metadata for an anime by AniList ID.
func fetchAniListDetail(client *http.Client, anilistID int) *aniListDetailEnvelope {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	if anilistID <= 0 {
		return nil
	}

	body, err := json.Marshal(map[string]any{
		"query":     aniListDetailQuery,
		"variables": map[string]any{"id": anilistID},
	})
	if err != nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://graphql.anilist.co", bytes.NewReader(body))
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

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	var out aniListDetailEnvelope
	if err := json.Unmarshal(payload, &out); err != nil || out.Data.Media.ID == 0 {
		return nil
	}
	return &out
}

// ─── Wails-exposed methods on App ───

// GetAnimeDetails returns library entry for an AniList ID, fetching from AniList if not cached.
func (a *App) GetAnimeDetails(anilistID int) (*AnimeLibraryEntry, error) {
	if anilistID <= 0 {
		return nil, fmt.Errorf("invalid AniList ID")
	}

	if a.library == nil {
		return nil, fmt.Errorf("library not initialized")
	}

	// Check if we already have a rich entry
	entry := a.library.getEntry(anilistID)
	if entry != nil && entry.Title != "" && entry.CoverImage != "" {
		return entry, nil
	}

	// Fetch from AniList
	detail := fetchAniListDetail(a.httpClient, anilistID)
	if detail == nil || detail.Data.Media.ID == 0 {
		// If we have a partial entry, return it
		if entry != nil {
			return entry, nil
		}
		return nil, fmt.Errorf("anime not found on AniList (ID: %d)", anilistID)
	}

	a.library.updateFromAniListData(anilistID, detail)
	go a.library.saveLibrary()

	return a.library.getEntry(anilistID), nil
}
