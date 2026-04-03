package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	movieapi "github.com/alvarorichard/Goanime/internal/api/movie"
	"github.com/alvarorichard/Goanime/internal/models"
)

const movieCatalogCacheTTL = 20 * time.Minute

func (a *App) GetMovieCatalog() ([]CatalogSection, error) {
	a.movieCatalogMu.RLock()
	if len(a.movieCatalogCache) > 0 && time.Now().Before(a.movieCatalogExpiry) {
		cached := a.movieCatalogCache
		a.movieCatalogMu.RUnlock()
		return cached, nil
	}
	a.movieCatalogMu.RUnlock()

	client := movieapi.NewTMDBClient()
	if !client.IsConfigured() {
		return nil, fmt.Errorf("TMDB_API_KEY not configured")
	}

	genreMap, err := client.GetMovieGenres()
	if err != nil {
		genreMap = map[int]string{}
	}

	type task struct {
		idx   int
		label string
		fetch func() (*models.TMDBSearchResult, error)
	}

	tasks := []task{
		{idx: 0, label: "Em alta esta semana", fetch: func() (*models.TMDBSearchResult, error) {
			return client.GetTrending("movie", "week")
		}},
		{idx: 1, label: "Mais populares", fetch: func() (*models.TMDBSearchResult, error) {
			return client.GetPopular("movie")
		}},
		{idx: 2, label: "Em cartaz", fetch: func() (*models.TMDBSearchResult, error) {
			return client.GetNowPlaying()
		}},
		{idx: 3, label: "Proximos lancamentos", fetch: func() (*models.TMDBSearchResult, error) {
			return client.GetUpcoming()
		}},
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
			result, err := t.fetch()
			if err != nil || result == nil || len(result.Results) == 0 {
				return
			}

			items := tmdbToCatalogItems(result.Results, genreMap, t.idx)
			if len(items) == 0 {
				return
			}
			ch <- indexedResult{idx: t.idx, label: t.label, items: items}
		}(t)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	ordered := make([]CatalogSection, len(tasks))
	for result := range ch {
		ordered[result.idx] = CatalogSection{
			Label: result.label,
			Items: result.items,
		}
	}

	sections := make([]CatalogSection, 0, len(ordered))
	for _, section := range ordered {
		if len(section.Items) > 0 {
			sections = append(sections, section)
		}
	}

	if len(sections) == 0 {
		return nil, fmt.Errorf("TMDb returned no movie catalog sections")
	}

	a.movieCatalogMu.Lock()
	a.movieCatalogCache = sections
	a.movieCatalogExpiry = time.Now().Add(movieCatalogCacheTTL)
	a.movieCatalogMu.Unlock()

	return sections, nil
}

func tmdbToCatalogItems(results []models.TMDBMedia, genreMap map[int]string, sectionIndex int) []CatalogItem {
	items := make([]CatalogItem, 0, min(len(results), 20))
	seen := make(map[int]bool, len(results))

	for _, media := range results {
		if len(items) >= 20 {
			break
		}
		if media.ID == 0 || seen[media.ID] {
			continue
		}

		title := strings.TrimSpace(media.GetDisplayTitle())
		if title == "" {
			continue
		}

		seen[media.ID] = true
		items = append(items, CatalogItem{
			ID:          20_000_000 + sectionIndex*1000 + media.ID,
			Title:       title,
			CoverImage:  media.GetPosterURL("w500"),
			BannerImage: media.GetBackdropURL("w1280"),
			Score:       media.VoteAverage,
			Genres:      tmdbGenreNames(media.GenreIDs, genreMap),
			Description: cleanCatalogDesc(media.Overview),
			Status:      tmdbMovieStatus(media.ReleaseDate),
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Score == items[j].Score {
			return strings.ToLower(items[i].Title) < strings.ToLower(items[j].Title)
		}
		return items[i].Score > items[j].Score
	})

	return items
}

func tmdbGenreNames(ids []int, genreMap map[int]string) []string {
	if len(ids) == 0 || len(genreMap) == 0 {
		return nil
	}

	genres := make([]string, 0, len(ids))
	for _, id := range ids {
		name := strings.TrimSpace(genreMap[id])
		if name == "" {
			continue
		}
		genres = append(genres, name)
	}
	return genres
}

func tmdbMovieStatus(releaseDate string) string {
	releaseDate = strings.TrimSpace(releaseDate)
	if releaseDate == "" {
		return ""
	}

	releaseAt, err := time.Parse("2006-01-02", releaseDate)
	if err != nil {
		return ""
	}
	if releaseAt.After(time.Now()) {
		return "NOT_YET_RELEASED"
	}
	return ""
}
