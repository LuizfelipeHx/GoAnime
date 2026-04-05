package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/alvarorichard/Goanime/internal/models"
	"github.com/alvarorichard/Goanime/internal/scraper"
	"github.com/alvarorichard/Goanime/internal/util"
)

const (
	anilistEndpoint      = "https://graphql.anilist.co"
	calendarCacheTTL     = 6 * time.Hour
)

var htmlTagsRe = regexp.MustCompile(`<[^>]+>`)

const catalogQuery = `
query ($page: Int, $perPage: Int, $sort: [MediaSort], $season: MediaSeason, $seasonYear: Int) {
  Page(page: $page, perPage: $perPage) {
    media(sort: $sort, type: ANIME, isAdult: false, season: $season, seasonYear: $seasonYear) {
      id
      title { romaji english }
      coverImage { large }
      bannerImage
      averageScore
      genres
      episodes
      description(asHtml: false)
      status
    }
  }
}
`

const genreCatalogQuery = `
query ($page: Int, $perPage: Int, $sort: [MediaSort], $genre: String) {
  Page(page: $page, perPage: $perPage) {
    media(sort: $sort, type: ANIME, isAdult: false, genre: $genre) {
      id
      title { romaji english }
      coverImage { large }
      bannerImage
      averageScore
      genres
      episodes
      description(asHtml: false)
      status
    }
  }
}
`

type anilistPageResp struct {
	Data struct {
		Page struct {
			Media []anilistMediaNode `json:"media"`
		} `json:"Page"`
	} `json:"data"`
}

type anilistMediaNode struct {
	ID    int `json:"id"`
	Title struct {
		Romaji  string `json:"romaji"`
		English string `json:"english"`
	} `json:"title"`
	CoverImage struct {
		Large string `json:"large"`
	} `json:"coverImage"`
	BannerImage  string   `json:"bannerImage"`
	AverageScore int      `json:"averageScore"`
	Genres       []string `json:"genres"`
	Episodes     int      `json:"episodes"`
	Description  string   `json:"description"`
	Status       string   `json:"status"`
}

// GetCatalog returns trending, popular, seasonal (AniList) plus MAL top/airing/upcoming sections.
// Results are cached for 10 minutes.
func (a *App) GetCatalog() []CatalogSection {
	a.catalogMu.RLock()
	if len(a.catalogCache) > 0 && time.Now().Before(a.catalogExpiry) {
		cached := a.catalogCache
		a.catalogMu.RUnlock()
		return cached
	}
	a.catalogMu.RUnlock()

	// Double-check after acquiring write lock to prevent duplicate rebuilds
	a.catalogMu.Lock()
	if len(a.catalogCache) > 0 && time.Now().Before(a.catalogExpiry) {
		cached := a.catalogCache
		a.catalogMu.Unlock()
		return cached
	}
	a.catalogMu.Unlock()

	now := time.Now()
	season := currentAniListSeason(now.Month())
	year := now.Year()
	bakashiSections := a.fetchBakashiCatalogSections()

	type task struct {
		idx   int
		label string
		// AniList params (jikanFilter == "" means use AniList)
		sort   []string
		season string
		year   int
		// Jikan params
		jikanFilter string // "score", "airing", "upcoming", "bypopularity", "season"
	}

	tasks := []task{
		{0, "Em alta agora", []string{"TRENDING_DESC"}, "", 0, ""},
		{1, "Mais populares", []string{"POPULARITY_DESC"}, "", 0, ""},
		{2, "Temporada atual", []string{"POPULARITY_DESC"}, season, year, ""},
		{3, "Top Anime (MAL)", nil, "", 0, "score"},
		{4, "Em exibição agora (MAL)", nil, "", 0, "airing"},
		{5, "Próximos lançamentos", nil, "", 0, "upcoming"},
	}

	type indexedResult struct {
		idx   int
		label string
		items []CatalogItem
	}

	ch := make(chan indexedResult, len(tasks))
	var wg sync.WaitGroup

	for _, t := range tasks {
		wg.Add(1)
		go func(t task) {
			defer wg.Done()
			var items []CatalogItem
			var err error
			if t.jikanFilter != "" {
				// Stagger Jikan requests to respect rate limit (tasks 3,4,5 → delays 0,600,1200ms)
				jikanIdx := t.idx - 3
				if jikanIdx > 0 {
					time.Sleep(time.Duration(jikanIdx) * 600 * time.Millisecond)
				}
				items, err = fetchJikanCatalogSection(a.httpClient, t.jikanFilter, 25)
			} else {
				items, err = a.fetchCatalogItems(t.sort, t.season, t.year, 20)
			}
			if err != nil {
				log.Printf("catalog task %q failed: %v", t.label, err)
				return
			}
			if len(items) == 0 {
				return
			}
			ch <- indexedResult{t.idx, t.label, items}
		}(t)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	ordered := make([]CatalogSection, len(tasks))
	for r := range ch {
		ordered[r.idx] = CatalogSection{Label: r.label, Items: r.items}
	}

	sections := make([]CatalogSection, 0, len(bakashiSections)+len(ordered))
	sections = append(sections, bakashiSections...)
	for _, s := range ordered {
		if len(s.Items) > 0 {
			sections = append(sections, s)
		}
	}

	a.catalogMu.Lock()
	a.catalogCache = sections
	a.catalogExpiry = time.Now().Add(10 * time.Minute)
	a.catalogMu.Unlock()

	return sections
}

// jikanStatusMap maps Jikan status strings to AniList-compatible status keys
// (reused by the existing Catalog frontend component).
var jikanStatusMap = map[string]string{
	"Finished Airing":  "FINISHED",
	"Currently Airing": "RELEASING",
	"Not yet aired":    "NOT_YET_RELEASED",
}

func fetchJikanCatalogSection(client *http.Client, filter string, limit int) ([]CatalogItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 14*time.Second)
	defer cancel()

	var reqURL string
	switch filter {
	case "season":
		reqURL = fmt.Sprintf("https://api.jikan.moe/v4/seasons/now?limit=%d", limit)
	case "score":
		reqURL = fmt.Sprintf("https://api.jikan.moe/v4/top/anime?limit=%d&sfw=true", limit)
	default:
		reqURL = fmt.Sprintf(
			"https://api.jikan.moe/v4/top/anime?filter=%s&limit=%d&sfw=true",
			url.QueryEscape(filter), limit,
		)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	util.GetJikanLimiter().Wait()
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jikan returned %d", resp.StatusCode)
	}

	var result struct {
		Data []struct {
			MalID        int    `json:"mal_id"`
			Title        string `json:"title"`
			TitleEnglish string `json:"title_english"`
			Images       struct {
				Jpg struct {
					LargeImageURL string `json:"large_image_url"`
					ImageURL      string `json:"image_url"`
				} `json:"jpg"`
			} `json:"images"`
			Score  float64 `json:"score"`
			Genres []struct {
				Name string `json:"name"`
			} `json:"genres"`
			Episodes int    `json:"episodes"`
			Synopsis string `json:"synopsis"`
			Status   string `json:"status"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	items := make([]CatalogItem, 0, len(result.Data))
	for i, m := range result.Data {
		title := m.TitleEnglish
		if title == "" {
			title = m.Title
		}

		img := m.Images.Jpg.LargeImageURL
		if img == "" {
			img = m.Images.Jpg.ImageURL
		}

		genres := make([]string, 0, len(m.Genres))
		for _, g := range m.Genres {
			genres = append(genres, g.Name)
		}

		status := jikanStatusMap[m.Status]
		if status == "" {
			status = m.Status
		}

		desc := strings.TrimSpace(m.Synopsis)
		runes := []rune(desc)
		if len(runes) > 220 {
			desc = string(runes[:220]) + "..."
		}

		items = append(items, CatalogItem{
			// Offset MAL IDs by 10_000_000 to avoid collision with AniList IDs
			ID:          10_000_000 + m.MalID + i,
			Title:       title,
			CoverImage:  img,
			Score:       m.Score,
			Genres:      genres,
			Episodes:    m.Episodes,
			Description: desc,
			Status:      status,
		})
	}
	return items, nil
}

func (a *App) fetchCatalogItems(sortBy []string, season string, seasonYear int, perPage int) ([]CatalogItem, error) {
	vars := map[string]interface{}{
		"page":    1,
		"perPage": perPage,
		"sort":    sortBy,
	}
	if season != "" {
		vars["season"] = season
		vars["seasonYear"] = seasonYear
	}

	body, err := json.Marshal(map[string]interface{}{
		"query":     catalogQuery,
		"variables": vars,
	})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, anilistEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; GoAnime/1.0)")

	util.GetAniListLimiter().Wait()
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("anilist request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anilist returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var page anilistPageResp
	if err := json.Unmarshal(data, &page); err != nil {
		return nil, err
	}

	items := make([]CatalogItem, 0, len(page.Data.Page.Media))
	for _, m := range page.Data.Page.Media {
		title := m.Title.English
		if title == "" {
			title = m.Title.Romaji
		}
		items = append(items, CatalogItem{
			ID:          m.ID,
			Title:       title,
			CoverImage:  m.CoverImage.Large,
			BannerImage: m.BannerImage,
			Score:       float64(m.AverageScore) / 10.0,
			Genres:      m.Genres,
			Episodes:    m.Episodes,
			Description: cleanCatalogDesc(m.Description),
			Status:      m.Status,
		})
	}
	return items, nil
}

func cleanCatalogDesc(s string) string {
	s = htmlTagsRe.ReplaceAllString(s, " ")
	s = strings.Join(strings.Fields(s), " ")
	runes := []rune(s)
	if len(runes) > 220 {
		s = string(runes[:220]) + "..."
	}
	return s
}

func currentAniListSeason(m time.Month) string {
	switch {
	case m >= 1 && m <= 3:
		return "WINTER"
	case m >= 4 && m <= 6:
		return "SPRING"
	case m >= 7 && m <= 9:
		return "SUMMER"
	default:
		return "FALL"
	}
}

func (a *App) fetchBakashiCatalogSections() []CatalogSection {
	client := scraper.NewBakashiClient()
	rawSections, err := client.GetCatalogSections()
	if err != nil || len(rawSections) == 0 {
		return nil
	}

	sections := make([]CatalogSection, 0, len(rawSections))
	for sectionIdx, raw := range rawSections {
		items := make([]CatalogItem, 0, len(raw.Items))
		for itemIdx, media := range raw.Items {
			if media == nil {
				continue
			}
			items = append(items, bakashiCatalogItem(sectionIdx, itemIdx, media))
		}
		if len(items) == 0 {
			continue
		}
		sections = append(sections, CatalogSection{
			Label: raw.Label,
			Items: items,
		})
	}
	return sections
}

func bakashiCatalogItem(sectionIdx int, itemIdx int, media *models.Anime) CatalogItem {
	title := strings.TrimSpace(media.Name)
	description := strings.TrimSpace(media.Overview)
	if description == "" {
		switch media.MediaType {
		case models.MediaTypeMovie:
			description = "Filme de anime no Bakashi"
		default:
			description = "Catalogo PT-BR do Bakashi"
		}
	}

	status := "RELEASING"
	if media.MediaType == models.MediaTypeMovie {
		status = "FINISHED"
	}

	genres := []string{"Bakashi"}
	if media.MediaType == models.MediaTypeMovie {
		genres = append(genres, "Filme")
	} else {
		genres = append(genres, "Anime")
	}
	if strings.TrimSpace(media.Year) != "" {
		genres = append(genres, media.Year)
	}

	cover := strings.TrimSpace(media.ImageURL)
	return CatalogItem{
		ID:          20000000 + sectionIdx*1000 + itemIdx,
		Title:       title,
		CoverImage:  cover,
		BannerImage: cover,
		Score:       media.Rating,
		Genres:      genres,
		Description: description,
		Status:      status,
	}
}

// GetGenres returns the list of available anime genres for filtering.
func (a *App) GetGenres() []string {
	return []string{
		"Action", "Adventure", "Comedy", "Drama", "Ecchi", "Fantasy",
		"Horror", "Mahou Shoujo", "Mecha", "Music", "Mystery",
		"Psychological", "Romance", "Sci-Fi", "Slice of Life",
		"Sports", "Supernatural", "Thriller",
	}
}

// GetCatalogByGenre returns catalog sections filtered by a specific genre.
// Results are cached for 10 minutes per genre.
func (a *App) GetCatalogByGenre(genre string) []CatalogSection {
	genre = strings.TrimSpace(genre)
	if genre == "" {
		return a.GetCatalog()
	}

	a.genreCacheMu.RLock()
	if cached, ok := a.genreCache[genre]; ok {
		if exp, ok2 := a.genreCacheTime[genre]; ok2 && time.Now().Before(exp) {
			a.genreCacheMu.RUnlock()
			return cached
		}
	}
	a.genreCacheMu.RUnlock()

	// Double-check after acquiring write lock to prevent duplicate rebuilds
	a.genreCacheMu.Lock()
	if cached, ok := a.genreCache[genre]; ok {
		if exp, ok2 := a.genreCacheTime[genre]; ok2 && time.Now().Before(exp) {
			a.genreCacheMu.Unlock()
			return cached
		}
	}
	a.genreCacheMu.Unlock()

	type genreTask struct {
		idx   int
		label string
		sort  []string
	}

	tasks := []genreTask{
		{0, "Em alta de " + genre, []string{"TRENDING_DESC"}},
		{1, "Populares de " + genre, []string{"POPULARITY_DESC"}},
		{2, "Melhores avaliados de " + genre, []string{"SCORE_DESC"}},
	}

	type indexedResult struct {
		idx   int
		label string
		items []CatalogItem
	}

	ch := make(chan indexedResult, len(tasks))
	var wg sync.WaitGroup

	for _, t := range tasks {
		wg.Add(1)
		go func(t genreTask) {
			defer wg.Done()
			items, err := a.fetchCatalogItemsByGenre(t.sort, genre, 25)
			if err != nil {
				log.Printf("catalog genre task %q failed: %v", t.label, err)
				return
			}
			if len(items) == 0 {
				return
			}
			ch <- indexedResult{t.idx, t.label, items}
		}(t)
	}

	go func() { wg.Wait(); close(ch) }()

	ordered := make([]CatalogSection, len(tasks))
	for r := range ch {
		ordered[r.idx] = CatalogSection{Label: r.label, Items: r.items}
	}

	var sections []CatalogSection
	for _, s := range ordered {
		if len(s.Items) > 0 {
			sections = append(sections, s)
		}
	}

	a.genreCacheMu.Lock()
	if a.genreCache == nil {
		a.genreCache = make(map[string][]CatalogSection)
		a.genreCacheTime = make(map[string]time.Time)
	}
	a.genreCache[genre] = sections
	a.genreCacheTime[genre] = time.Now().Add(10 * time.Minute)
	a.genreCacheMu.Unlock()

	return sections
}

func (a *App) fetchCatalogItemsByGenre(sortBy []string, genre string, perPage int) ([]CatalogItem, error) {
	vars := map[string]interface{}{
		"page":    1,
		"perPage": perPage,
		"sort":    sortBy,
		"genre":   genre,
	}

	body, err := json.Marshal(map[string]interface{}{
		"query":     genreCatalogQuery,
		"variables": vars,
	})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, anilistEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; GoAnime/1.0)")

	util.GetAniListLimiter().Wait()
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("anilist genre request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anilist returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var page anilistPageResp
	if err := json.Unmarshal(data, &page); err != nil {
		return nil, err
	}

	items := make([]CatalogItem, 0, len(page.Data.Page.Media))
	for _, m := range page.Data.Page.Media {
		title := m.Title.English
		if title == "" {
			title = m.Title.Romaji
		}
		desc := htmlTagsRe.ReplaceAllString(m.Description, "")
		runes := []rune(desc)
		if len(runes) > 220 {
			desc = string(runes[:220]) + "..."
		}
		items = append(items, CatalogItem{
			ID:          m.ID,
			Title:       title,
			CoverImage:  m.CoverImage.Large,
			BannerImage: m.BannerImage,
			Score:       float64(m.AverageScore) / 10.0,
			Genres:      m.Genres,
			Episodes:    m.Episodes,
			Description: desc,
			Status:      m.Status,
		})
	}
	return items, nil
}

const airingScheduleQuery = `
query ($start: Int, $end: Int) {
  Page(perPage: 50) {
    airingSchedules(airingAt_greater: $start, airingAt_lesser: $end, sort: TIME) {
      airingAt
      episode
      media {
        id
        title { romaji english native }
        coverImage { large }
        episodes
        format
        status
      }
    }
  }
}
`

type airingScheduleResp struct {
	Data struct {
		Page struct {
			AiringSchedules []struct {
				AiringAt int64 `json:"airingAt"`
				Episode  int   `json:"episode"`
				Media    struct {
					ID    int `json:"id"`
					Title struct {
						Romaji  string `json:"romaji"`
						English string `json:"english"`
						Native  string `json:"native"`
					} `json:"title"`
					CoverImage struct {
						Large string `json:"large"`
					} `json:"coverImage"`
					Episodes int    `json:"episodes"`
					Format   string `json:"format"`
					Status   string `json:"status"`
				} `json:"media"`
			} `json:"airingSchedules"`
		} `json:"Page"`
	} `json:"data"`
}

var weekdayNamesPTBR = [7]string{
	"Domingo", "Segunda", "Terça", "Quarta", "Quinta", "Sexta", "Sábado",
}

// GetSeasonCalendar returns the airing schedule for the current week, grouped by day.
func (a *App) GetSeasonCalendar() []CalendarDay {
	a.calendarMu.RLock()
	if len(a.calendarCache) > 0 && time.Now().Before(a.calendarExpiry) {
		cached := a.calendarCache
		a.calendarMu.RUnlock()
		return cached
	}
	a.calendarMu.RUnlock()

	// Double-check after acquiring write lock
	a.calendarMu.Lock()
	if len(a.calendarCache) > 0 && time.Now().Before(a.calendarExpiry) {
		cached := a.calendarCache
		a.calendarMu.Unlock()
		return cached
	}
	a.calendarMu.Unlock()

	now := time.Now()
	// Find Monday of the current week
	weekday := now.Weekday()
	offset := int(weekday) - int(time.Monday)
	if offset < 0 {
		offset += 7
	}
	monday := time.Date(now.Year(), now.Month(), now.Day()-offset, 0, 0, 0, 0, now.Location())
	sunday := monday.AddDate(0, 0, 7).Add(-time.Second)

	startEpoch := monday.Unix()
	endEpoch := sunday.Unix()

	vars := map[string]interface{}{
		"start": startEpoch,
		"end":   endEpoch,
	}

	body, err := json.Marshal(map[string]interface{}{
		"query":     airingScheduleQuery,
		"variables": vars,
	})
	if err != nil {
		log.Printf("calendar: marshal error: %v", err)
		return []CalendarDay{}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, anilistEndpoint, bytes.NewReader(body))
	if err != nil {
		log.Printf("calendar: request error: %v", err)
		return []CalendarDay{}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; GoAnime/1.0)")

	util.GetAniListLimiter().Wait()
	resp, err := a.httpClient.Do(req)
	if err != nil {
		log.Printf("calendar: fetch error: %v", err)
		return []CalendarDay{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("calendar: anilist returned status %d", resp.StatusCode)
		return []CalendarDay{}
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("calendar: read error: %v", err)
		return []CalendarDay{}
	}

	var page airingScheduleResp
	if err := json.Unmarshal(data, &page); err != nil {
		log.Printf("calendar: parse error: %v", err)
		return []CalendarDay{}
	}

	// Group by day of week
	dayMap := make(map[int][]CalendarEntry) // key = Weekday int (0=Sun..6=Sat)
	seen := make(map[int]bool)              // deduplicate by media ID per day
	for _, sched := range page.Data.Page.AiringSchedules {
		t := time.Unix(sched.AiringAt, 0).In(now.Location())
		dayIdx := int(t.Weekday())

		mediaID := sched.Media.ID
		dedupeKey := dayIdx*100000 + mediaID
		if seen[dedupeKey] {
			continue
		}
		seen[dedupeKey] = true

		title := sched.Media.Title.English
		if title == "" {
			title = sched.Media.Title.Romaji
		}
		if title == "" {
			title = sched.Media.Title.Native
		}

		dayMap[dayIdx] = append(dayMap[dayIdx], CalendarEntry{
			Title:         title,
			ImageURL:      sched.Media.CoverImage.Large,
			Episode:       sched.Episode,
			TotalEpisodes: sched.Media.Episodes,
			AiringAt:      sched.AiringAt,
			Format:        sched.Media.Format,
		})
	}

	// Build ordered result starting from Monday
	dayOrder := []int{1, 2, 3, 4, 5, 6, 0} // Mon, Tue, Wed, Thu, Fri, Sat, Sun
	var days []CalendarDay
	for _, d := range dayOrder {
		entries, ok := dayMap[d]
		if !ok || len(entries) == 0 {
			continue
		}
		days = append(days, CalendarDay{
			Day:     weekdayNamesPTBR[d],
			Entries: entries,
		})
	}

	a.calendarMu.Lock()
	a.calendarCache = days
	a.calendarExpiry = time.Now().Add(calendarCacheTTL)
	a.calendarMu.Unlock()

	return days
}
