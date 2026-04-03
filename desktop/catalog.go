package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/alvarorichard/Goanime/internal/models"
	"github.com/alvarorichard/Goanime/internal/scraper"
)

const anilistEndpoint = "https://graphql.anilist.co"

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
				// Stagger Jikan requests slightly (tasks 3,4,5 → delays 0,400,800ms)
				jikanIdx := t.idx - 3
				if jikanIdx > 0 {
					time.Sleep(time.Duration(jikanIdx) * 400 * time.Millisecond)
				}
				items, err = fetchJikanCatalogSection(a.httpClient, t.jikanFilter, 25)
			} else {
				items, err = a.fetchCatalogItems(t.sort, t.season, t.year, 20)
			}
			if err != nil || len(items) == 0 {
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
			if err != nil || len(items) == 0 {
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
