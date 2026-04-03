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
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alvarorichard/Goanime/internal/api"
	"github.com/alvarorichard/Goanime/internal/models"
	"github.com/alvarorichard/Goanime/internal/scraper"
)

type animeSearchContext struct {
	mu             sync.Mutex
	Query          string
	CanonicalTitle string
	Aliases        []string
	RelatedTitles  []string
	SeasonNumber   int
	Year           int
	AniListID      int
	MalID          int
}

type aniListSearchEnvelope struct {
	Data struct {
		Media struct {
			ID         int `json:"id"`
			IDMal      int `json:"idMal"`
			SeasonYear int `json:"seasonYear"`
			StartDate  struct {
				Year int `json:"year"`
			} `json:"startDate"`
			Title struct {
				Romaji        string `json:"romaji"`
				English       string `json:"english"`
				Native        string `json:"native"`
				UserPreferred string `json:"userPreferred"`
			} `json:"title"`
			Synonyms  []string `json:"synonyms"`
			Relations struct {
				Edges []struct {
					RelationType string `json:"relationType"`
				} `json:"edges"`
				Nodes []struct {
					Title struct {
						Romaji        string `json:"romaji"`
						English       string `json:"english"`
						Native        string `json:"native"`
						UserPreferred string `json:"userPreferred"`
					} `json:"title"`
				} `json:"nodes"`
			} `json:"relations"`
		} `json:"Media"`
	} `json:"data"`
}

type jikanSearchEnvelope struct {
	Data []struct {
		MalID         int      `json:"mal_id"`
		Title         string   `json:"title"`
		TitleEnglish  string   `json:"title_english"`
		TitleJapanese string   `json:"title_japanese"`
		TitleSynonyms []string `json:"title_synonyms"`
		Year          int      `json:"year"`
		Titles        []struct {
			Title string `json:"title"`
		} `json:"titles"`
	} `json:"data"`
}

func (a *App) searchAnimeResolved(query string, sourceType *scraper.ScraperType) ([]*models.Anime, *animeSearchContext, error) {
	ctx := a.resolveAnimeSearchContext(query)
	searchTerms := append([]string(nil), ctx.Aliases...)
	searchTerms = mergeSearchTerms(searchTerms, ctx.RelatedTitles)
	// Limit to 5 terms to avoid sequential multi-minute searches
	if len(searchTerms) > 5 {
		searchTerms = searchTerms[:5]
	}

	seen := make(map[string]bool)
	var seenMu sync.Mutex
	results := make([]*models.Anime, 0, maxSearchItems)
	var searchErrors []string
	deadline := time.Now().Add(14 * time.Second)

	for _, term := range searchTerms {
		if time.Now().After(deadline) || len(results) >= maxSearchItems {
			break
		}
		batch, err := a.manager.SearchAnime(term, sourceType)
		if err != nil {
			searchErrors = append(searchErrors, err.Error())
			continue
		}

		seenMu.Lock()
		for _, item := range batch {
			if item == nil {
				continue
			}
			key := strings.ToLower(strings.TrimSpace(item.Source)) + "|" + strings.TrimSpace(item.URL)
			if key == "|" || seen[key] {
				continue
			}
			seen[key] = true
			results = append(results, item)
			if len(results) >= maxSearchItems {
				break
			}
		}
		seenMu.Unlock()

		if len(results) >= maxSearchItems {
			break
		}
	}

	if len(results) == 0 {
		if len(searchErrors) > 0 {
			return nil, ctx, fmt.Errorf("no anime found with name: %s (%s)", query, strings.Join(searchErrors, "; "))
		}
		return nil, ctx, fmt.Errorf("no anime found with name: %s", query)
	}

	rankAnimeResults(results, ctx)
	return results, ctx, nil
}

func (a *App) resolveAnimeSearchContext(query string) *animeSearchContext {
	ctx := &animeSearchContext{
		Query:        query,
		Aliases:      buildAnimeSearchAliases(query),
		SeasonNumber: extractSeasonNumber(query),
	}

	// Run AniList and Jikan enrichment in parallel to avoid 8s+8s+6s sequential wait
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		a.enrichAnimeSearchContextFromAniList(ctx)
	}()
	go func() {
		defer wg.Done()
		a.enrichAnimeSearchContextFromJikan(ctx)
	}()
	wg.Wait()

	if ctx.CanonicalTitle == "" && len(ctx.Aliases) > 0 {
		ctx.CanonicalTitle = ctx.Aliases[0]
	}
	if ctx.CanonicalTitle != "" {
		ctx.Aliases = mergeSearchTerms([]string{ctx.CanonicalTitle}, ctx.Aliases)
	}

	return ctx
}

func (a *App) enrichAnimeSearchContextFromAniList(ctx *animeSearchContext) {
	searchTerms := buildAnimeSearchAliases(ctx.Query)
	resp := fetchAniListSearchMetadata(a.httpClient, searchTerms)
	if resp == nil || resp.Data.Media.ID == 0 {
		return
	}

	titles := []string{
		resp.Data.Media.Title.UserPreferred,
		resp.Data.Media.Title.English,
		resp.Data.Media.Title.Romaji,
		resp.Data.Media.Title.Native,
	}

	related := make([]string, 0, len(resp.Data.Media.Relations.Nodes)*2)
	for i, node := range resp.Data.Media.Relations.Nodes {
		relationType := ""
		if i < len(resp.Data.Media.Relations.Edges) {
			relationType = strings.ToUpper(strings.TrimSpace(resp.Data.Media.Relations.Edges[i].RelationType))
		}
		if relationType == "" || !isRelevantRelationType(relationType) {
			continue
		}
		related = append(related,
			node.Title.UserPreferred,
			node.Title.English,
			node.Title.Romaji,
			node.Title.Native,
		)
	}

	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.AniListID = resp.Data.Media.ID
	if ctx.MalID == 0 {
		ctx.MalID = resp.Data.Media.IDMal
	}
	if ctx.Year == 0 {
		if resp.Data.Media.SeasonYear > 0 {
			ctx.Year = resp.Data.Media.SeasonYear
		} else {
			ctx.Year = resp.Data.Media.StartDate.Year
		}
	}
	ctx.Aliases = mergeSearchTerms(ctx.Aliases, titles)
	ctx.Aliases = mergeSearchTerms(ctx.Aliases, resp.Data.Media.Synonyms)
	if ctx.CanonicalTitle == "" {
		ctx.CanonicalTitle = firstNonEmpty(titles...)
	}
	ctx.RelatedTitles = mergeSearchTerms(ctx.RelatedTitles, related)
}

func (a *App) enrichAnimeSearchContextFromJikan(ctx *animeSearchContext) {
	ctx.mu.Lock()
	query := ctx.Query
	ctx.mu.Unlock()

	resp := fetchJikanSearchMetadata(a.httpClient, query)
	if resp == nil || len(resp.Data) == 0 {
		return
	}

	ctx.mu.Lock()
	aliases := append([]string(nil), ctx.Aliases...)
	canonical := ctx.CanonicalTitle
	ctx.mu.Unlock()

	candidate := pickBestJikanCandidate(resp.Data, mergeSearchTerms(aliases, []string{canonical}))
	if candidate == nil {
		return
	}

	var malID int
	ctx.mu.Lock()
	if ctx.MalID == 0 {
		ctx.MalID = candidate.MalID
	}
	malID = ctx.MalID
	if ctx.Year == 0 {
		ctx.Year = candidate.Year
	}
	ctx.Aliases = mergeSearchTerms(ctx.Aliases, collectJikanTitles(*candidate))
	if ctx.CanonicalTitle == "" {
		ctx.CanonicalTitle = firstNonEmpty(candidate.TitleEnglish, candidate.Title, candidate.TitleJapanese)
	}
	ctx.mu.Unlock()

	if malID > 0 {
		relations := jikanFetchRelations(a.httpClient, malID)
		titles := make([]string, 0, len(relations))
		for _, rel := range relations {
			titles = append(titles, rel.Name)
		}
		ctx.mu.Lock()
		ctx.RelatedTitles = mergeSearchTerms(ctx.RelatedTitles, titles)
		ctx.mu.Unlock()
	}
}

func fetchAniListSearchMetadata(client *http.Client, searchTerms []string) *aniListSearchEnvelope {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	const query = `query ($search: String) {
  Media(search: $search, type: ANIME) {
    id
    idMal
    seasonYear
    startDate { year }
    title { romaji english native userPreferred }
    synonyms
    relations {
      edges { relationType }
      nodes { title { romaji english native userPreferred } }
    }
  }
}`

	for _, term := range searchTerms {
		body, err := json.Marshal(map[string]any{
			"query":     query,
			"variables": map[string]any{"search": term},
		})
		if err != nil {
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://graphql.anilist.co", bytes.NewReader(body))
		if err != nil {
			cancel()
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			cancel()
			continue
		}
		payload, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		cancel()
		if readErr != nil || resp.StatusCode != http.StatusOK {
			continue
		}

		var out aniListSearchEnvelope
		if err := json.Unmarshal(payload, &out); err == nil && out.Data.Media.ID != 0 {
			return &out
		}
	}
	return nil
}

func fetchJikanSearchMetadata(client *http.Client, query string) *jikanSearchEnvelope {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	endpoint := "https://api.jikan.moe/v4/anime?q=" + url.QueryEscape(strings.TrimSpace(query)) + "&limit=5"
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Accept", "application/json")

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

	var out jikanSearchEnvelope
	if err := json.Unmarshal(payload, &out); err != nil {
		return nil
	}
	return &out
}

func pickBestJikanCandidate(items []struct {
	MalID         int      `json:"mal_id"`
	Title         string   `json:"title"`
	TitleEnglish  string   `json:"title_english"`
	TitleJapanese string   `json:"title_japanese"`
	TitleSynonyms []string `json:"title_synonyms"`
	Year          int      `json:"year"`
	Titles        []struct {
		Title string `json:"title"`
	} `json:"titles"`
}, aliases []string) *struct {
	MalID         int      `json:"mal_id"`
	Title         string   `json:"title"`
	TitleEnglish  string   `json:"title_english"`
	TitleJapanese string   `json:"title_japanese"`
	TitleSynonyms []string `json:"title_synonyms"`
	Year          int      `json:"year"`
	Titles        []struct {
		Title string `json:"title"`
	} `json:"titles"`
} {
	var best *struct {
		MalID         int      `json:"mal_id"`
		Title         string   `json:"title"`
		TitleEnglish  string   `json:"title_english"`
		TitleJapanese string   `json:"title_japanese"`
		TitleSynonyms []string `json:"title_synonyms"`
		Year          int      `json:"year"`
		Titles        []struct {
			Title string `json:"title"`
		} `json:"titles"`
	}
	bestScore := -1
	for i := range items {
		candidate := &items[i]
		score := 0
		for _, title := range collectJikanTitles(*candidate) {
			for _, alias := range aliases {
				score = max(score, titleSimilarityScore(title, alias))
			}
		}
		if score > bestScore {
			bestScore = score
			best = candidate
		}
	}
	return best
}

func collectJikanTitles(item struct {
	MalID         int      `json:"mal_id"`
	Title         string   `json:"title"`
	TitleEnglish  string   `json:"title_english"`
	TitleJapanese string   `json:"title_japanese"`
	TitleSynonyms []string `json:"title_synonyms"`
	Year          int      `json:"year"`
	Titles        []struct {
		Title string `json:"title"`
	} `json:"titles"`
}) []string {
	titles := []string{item.Title, item.TitleEnglish, item.TitleJapanese}
	titles = append(titles, item.TitleSynonyms...)
	for _, title := range item.Titles {
		titles = append(titles, title.Title)
	}
	return titles
}

func buildAnimeSearchAliases(query string) []string {
	cleaned := api.CleanTitle(query)
	aliases := mergeSearchTerms(nil, []string{query, cleaned})
	aliases = mergeSearchTerms(aliases, expandAnimeSearchTerm(cleaned))
	return aliases
}

func expandAnimeSearchTerm(term string) []string {
	term = strings.TrimSpace(term)
	if term == "" {
		return nil
	}

	variants := []string{term}
	colonRE := regexp.MustCompile(`:\s*.+$`)
	romanRE := regexp.MustCompile(`\s+(?:II|III|IV|V|VI|VII|VIII|IX|X)\s*$`)
	seasonNumRE := regexp.MustCompile(`\s+\d+\s*$`)
	seasonWordRE := regexp.MustCompile(`(?i)\s+(?:season|temporada)\s*\d+\s*$`)
	partRE := regexp.MustCompile(`(?i)\s+(?:part|parte)\s*\d+\s*$`)

	if colonRE.MatchString(term) {
		variants = append(variants, strings.TrimSpace(colonRE.ReplaceAllString(term, "")))
	}
	if romanRE.MatchString(term) {
		variants = append(variants, strings.TrimSpace(romanRE.ReplaceAllString(term, "")))
	}
	if seasonWordRE.MatchString(term) {
		variants = append(variants, strings.TrimSpace(seasonWordRE.ReplaceAllString(term, "")))
	}
	if partRE.MatchString(term) {
		variants = append(variants, strings.TrimSpace(partRE.ReplaceAllString(term, "")))
	}
	if seasonNumRE.MatchString(term) {
		variants = append(variants, strings.TrimSpace(seasonNumRE.ReplaceAllString(term, "")))
	}
	if strings.Contains(strings.ToLower(term), " no ") {
		variants = append(variants, strings.ReplaceAll(term, " no ", " "))
	}
	if strings.HasPrefix(strings.ToLower(term), "the ") {
		variants = append(variants, term[4:])
	}

	words := strings.Fields(term)
	if len(words) > 4 {
		variants = append(variants, strings.Join(words[:3], " "))
		variants = append(variants, strings.Join(words[:4], " "))
	}

	return variants
}

func mergeSearchTerms(current []string, incoming []string) []string {
	seen := make(map[string]bool, len(current)+len(incoming))
	merged := make([]string, 0, len(current)+len(incoming))
	for _, item := range append(current, incoming...) {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		norm := normalizeSearchText(item)
		if norm == "" || seen[norm] {
			continue
		}
		seen[norm] = true
		merged = append(merged, item)
	}
	return merged
}

func rankAnimeResults(items []*models.Anime, ctx *animeSearchContext) {
	if len(items) < 2 {
		return
	}

	sort.SliceStable(items, func(i, j int) bool {
		scoreI := animeMatchScore(items[i], ctx)
		scoreJ := animeMatchScore(items[j], ctx)
		if scoreI != scoreJ {
			return scoreI > scoreJ
		}
		if items[i].Source != items[j].Source {
			return strings.ToLower(items[i].Source) < strings.ToLower(items[j].Source)
		}
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
}

func animeMatchScore(item *models.Anime, ctx *animeSearchContext) int {
	if item == nil {
		return 0
	}
	name := normalizeSearchText(api.CleanTitle(item.Name))
	if name == "" {
		return 0
	}

	best := 0
	for _, alias := range mergeSearchTerms(ctx.Aliases, ctx.RelatedTitles) {
		candidate := normalizeSearchText(alias)
		if candidate == "" {
			continue
		}
		score := titleSimilarityScore(name, candidate)
		if score > best {
			best = score
		}
	}

	if ctx.SeasonNumber > 0 {
		itemSeason := extractSeasonNumber(item.Name)
		switch {
		case itemSeason == ctx.SeasonNumber:
			best += 120
		case itemSeason > 0 && itemSeason != ctx.SeasonNumber:
			best -= 80
		}
	}
	if ctx.Year > 0 && item.Year != "" {
		if year, _ := strconv.Atoi(strings.TrimSpace(item.Year)); year == ctx.Year {
			best += 40
		}
	}

	alt := mediaAlternativeFromAnime(item)
	best += animeActionScore("watch", ctxGroupKey(ctx), alt)
	return best
}

func buildAnimeDisplayResults(items []*models.Anime, ctx *animeSearchContext) []MediaResult {
	if len(items) == 0 {
		return nil
	}

	type animeGroup struct {
		key       string
		title     string
		season    int
		variants  []MediaAlternative
		represent *models.Anime
		bestScore int
		available map[string]bool
	}

	groups := make(map[string]*animeGroup)
	orderedKeys := make([]string, 0)
	for _, item := range items {
		if item == nil {
			continue
		}
		key, title, season := deriveAnimeGroupInfo(item, ctx)
		group := groups[key]
		if group == nil {
			group = &animeGroup{key: key, title: title, season: season, available: make(map[string]bool)}
			groups[key] = group
			orderedKeys = append(orderedKeys, key)
		}
		candidate := mediaAlternativeFromAnime(item)
		candidate.MediaType = normalizeMediaType(item)
		candidate.Name = cleanDisplayName(candidate.Name)
		group.variants = appendUniqueAlternative(group.variants, candidate)
		group.available[strings.TrimSpace(candidate.Source)] = true
		score := animeMatchScore(item, ctx)
		if group.represent == nil || score > group.bestScore {
			group.represent = item
			group.bestScore = score
		}
	}

	out := make([]MediaResult, 0, len(groups))
	for _, key := range orderedKeys {
		group := groups[key]
		if group == nil || len(group.variants) == 0 || group.represent == nil {
			continue
		}
		watchAlt := pickBestAlternative(group.variants, group.key, "watch")
		downloadAlt := pickBestAlternative(group.variants, group.key, "download")
		dubAlt := pickBestAlternative(group.variants, group.key, "dub")
		subAlt := pickBestAlternative(group.variants, group.key, "sub")
		availableSources := make([]string, 0, len(group.available))
		for source := range group.available {
			availableSources = append(availableSources, source)
		}
		sort.Strings(availableSources)

		represent := group.represent
		hasPortuguese, hasEnglish, hasDub, hasSub := summarizeAlternativeLanguages(group.variants)
		watchHasPortuguese, watchHasEnglish, watchHasDub, watchHasSub := detectAlternativeLanguage(watchAlt)
		result := MediaResult{
			Name:               cleanDisplayName(watchAlt.Name),
			URL:                watchAlt.URL,
			ImageURL:           represent.ImageURL,
			Source:             watchAlt.Source,
			MediaType:          normalizeMediaType(represent),
			Year:               represent.Year,
			Score:              float64(represent.Details.AverageScore) / 10,
			Description:        strings.TrimSpace(represent.Details.Description),
			Genres:             append([]string(nil), represent.Details.Genres...),
			CanonicalTitle:     chooseCanonicalGroupTitle(group.title, ctx),
			GroupKey:           group.key,
			SeasonNumber:       group.season,
			AvailableSources:   availableSources,
			WatchSource:        watchAlt.Source,
			DownloadSource:     downloadAlt.Source,
			DubSource:          dubAlt.Source,
			SubSource:          subAlt.Source,
			Alternatives:       group.variants,
			HasPortuguese:      hasPortuguese,
			HasEnglish:         hasEnglish,
			HasDub:             hasDub,
			HasSub:             hasSub,
			WatchHasPortuguese: watchHasPortuguese,
			WatchHasEnglish:    watchHasEnglish,
			WatchHasDub:        watchHasDub,
			WatchHasSub:        watchHasSub,
		}
		out = append(out, result)
	}

	sort.SliceStable(out, func(i, j int) bool {
		left := mediaResultPriority(out[i])
		right := mediaResultPriority(out[j])
		if left != right {
			return left > right
		}
		return strings.ToLower(out[i].CanonicalTitle) < strings.ToLower(out[j].CanonicalTitle)
	})

	if len(out) > maxSearchItems {
		out = out[:maxSearchItems]
	}
	return out
}

func buildAnimeSourceResults(items []*models.Anime, ctx *animeSearchContext) []MediaResult {
	if len(items) == 0 {
		return nil
	}

	type animeGroup struct {
		key       string
		title     string
		season    int
		variants  []MediaAlternative
		available map[string]bool
	}
	type rawEntry struct {
		item *models.Anime
		key  string
	}

	groups := make(map[string]*animeGroup)
	entries := make([]rawEntry, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		key, title, season := deriveAnimeGroupInfo(item, ctx)
		group := groups[key]
		if group == nil {
			group = &animeGroup{key: key, title: title, season: season, available: make(map[string]bool)}
			groups[key] = group
		}
		candidate := mediaAlternativeFromAnime(item)
		candidate.MediaType = normalizeMediaType(item)
		candidate.Name = cleanDisplayName(candidate.Name)
		group.variants = appendUniqueAlternative(group.variants, candidate)
		group.available[strings.TrimSpace(candidate.Source)] = true
		entries = append(entries, rawEntry{item: item, key: key})
	}

	sort.SliceStable(entries, func(i, j int) bool {
		leftAlt := mediaAlternativeFromAnime(entries[i].item)
		leftAlt.Name = cleanDisplayName(leftAlt.Name)
		rightAlt := mediaAlternativeFromAnime(entries[j].item)
		rightAlt.Name = cleanDisplayName(rightAlt.Name)

		leftPT, _, leftDub, _ := detectAlternativeLanguage(leftAlt)
		rightPT, _, rightDub, _ := detectAlternativeLanguage(rightAlt)
		if leftPT != rightPT {
			return leftPT
		}
		if leftDub != rightDub {
			return leftDub
		}

		scoreI := animeMatchScore(entries[i].item, ctx)
		scoreJ := animeMatchScore(entries[j].item, ctx)
		if scoreI != scoreJ {
			return scoreI > scoreJ
		}
		if entries[i].item.Source != entries[j].item.Source {
			return strings.ToLower(entries[i].item.Source) < strings.ToLower(entries[j].item.Source)
		}
		return strings.ToLower(cleanDisplayName(entries[i].item.Name)) < strings.ToLower(cleanDisplayName(entries[j].item.Name))
	})

	out := make([]MediaResult, 0, min(len(entries), maxSearchItems))
	for _, entry := range entries {
		if len(out) >= maxSearchItems {
			break
		}

		item := entry.item
		group := groups[entry.key]
		if item == nil || group == nil {
			continue
		}

		alt := mediaAlternativeFromAnime(item)
		alt.MediaType = normalizeMediaType(item)
		alt.Name = cleanDisplayName(alt.Name)
		watchHasPortuguese, watchHasEnglish, watchHasDub, watchHasSub := detectAlternativeLanguage(alt)
		hasPortuguese, hasEnglish, hasDub, hasSub := summarizeAlternativeLanguages(group.variants)

		availableSources := make([]string, 0, len(group.available))
		for source := range group.available {
			availableSources = append(availableSources, source)
		}
		sort.Strings(availableSources)

		alternatives := make([]MediaAlternative, 0, len(group.variants))
		for _, candidate := range group.variants {
			if strings.EqualFold(strings.TrimSpace(candidate.Source), strings.TrimSpace(alt.Source)) && strings.TrimSpace(candidate.URL) == strings.TrimSpace(alt.URL) {
				continue
			}
			alternatives = append(alternatives, candidate)
		}

		out = append(out, MediaResult{
			Name:               cleanDisplayName(item.Name),
			URL:                item.URL,
			ImageURL:           item.ImageURL,
			Source:             item.Source,
			MediaType:          normalizeMediaType(item),
			Year:               item.Year,
			Score:              float64(item.Details.AverageScore) / 10,
			Description:        strings.TrimSpace(item.Details.Description),
			Genres:             append([]string(nil), item.Details.Genres...),
			CanonicalTitle:     chooseCanonicalGroupTitle(group.title, ctx),
			GroupKey:           entry.key,
			SeasonNumber:       group.season,
			AvailableSources:   availableSources,
			Alternatives:       alternatives,
			HasPortuguese:      hasPortuguese,
			HasEnglish:         hasEnglish,
			HasDub:             hasDub,
			HasSub:             hasSub,
			WatchHasPortuguese: watchHasPortuguese,
			WatchHasEnglish:    watchHasEnglish,
			WatchHasDub:        watchHasDub,
			WatchHasSub:        watchHasSub,
		})
	}

	return out
}
func mediaResultPriority(item MediaResult) int {
	base := 0
	if item.Score > 0 {
		base += int(item.Score * 10)
	}
	if len(item.AvailableSources) > 0 {
		base += len(item.AvailableSources) * 20
	}
	base += sourceBasePreference(item.WatchSource, "watch")
	return base
}

func deriveAnimeGroupInfo(item *models.Anime, ctx *animeSearchContext) (string, string, int) {
	cleaned := cleanDisplayName(item.Name)
	season := extractSeasonNumber(cleaned)
	if season == 0 {
		season = ctx.SeasonNumber
	}
	aliasPool := mergeSearchTerms([]string{ctx.CanonicalTitle}, mergeSearchTerms(ctx.Aliases, ctx.RelatedTitles))
	bestAlias := cleaned
	bestScore := -1
	for _, alias := range aliasPool {
		score := titleSimilarityScore(cleaned, alias)
		if score > bestScore {
			bestScore = score
			bestAlias = alias
		}
	}
	if bestAlias == "" {
		bestAlias = cleaned
	}
	key := normalizeSearchText(bestAlias)
	if season > 0 {
		key = fmt.Sprintf("%s#s%d", key, season)
	}
	return key, bestAlias, season
}

func chooseCanonicalGroupTitle(title string, ctx *animeSearchContext) string {
	if title = strings.TrimSpace(title); title != "" {
		return cleanDisplayName(title)
	}
	if ctx != nil && strings.TrimSpace(ctx.CanonicalTitle) != "" {
		return cleanDisplayName(ctx.CanonicalTitle)
	}
	return cleanDisplayName(title)
}

func pickBestAlternative(items []MediaAlternative, groupKey string, action string) MediaAlternative {
	best := items[0]
	bestScore := -1 << 30
	for _, item := range items {
		score := animeActionScore(action, groupKey, item)
		if score > bestScore {
			bestScore = score
			best = item
		}
	}
	return best
}

func animeActionScore(action string, groupKey string, item MediaAlternative) int {
	score := sourceBasePreference(item.Source, action)
	title := strings.ToLower(item.Name)
	isDubTag := strings.Contains(title, "dublado") || strings.Contains(title, "[dub]")
	isSubTag := strings.Contains(title, "legendado") || strings.Contains(title, "[sub]") || strings.Contains(title, "[english]")
	switch action {
	case "watch":
		if isDubTag || isSubTag {
			score += 10
		}
	case "download":
		if strings.EqualFold(item.Source, "allanime") {
			score += 25
		}
	case "dub":
		if isDubTag {
			score += 40
		}
	case "sub":
		if isSubTag || !isDubTag {
			score += 20
		}
	}
	failurePenalty := sourceFailureCount(groupKey, action, item.Source) * 40
	return score - failurePenalty
}

func sourceBasePreference(source string, action string) int {
	source = strings.ToLower(strings.TrimSpace(source))
	switch action {
	case "watch":
		switch source {
		case "allanime":
			return 120
		case "animefire":
			return 110
		case "animesonlinecc":
			return 95
		case "anroll":
			return 55
		}
	case "download":
		switch source {
		case "allanime":
			return 140
		case "animefire":
			return 90
		case "animesonlinecc":
			return 80
		case "anroll":
			return 45
		}
	case "dub":
		switch source {
		case "animefire":
			return 140
		case "animesonlinecc":
			return 120
		case "anroll":
			return 100
		case "allanime":
			return 60
		}
	case "sub":
		switch source {
		case "allanime":
			return 140
		case "animefire":
			return 100
		case "animesonlinecc":
			return 85
		case "anroll":
			return 90
		}
	}
	return 50
}

func mediaAlternativeFromAnime(item *models.Anime) MediaAlternative {
	if item == nil {
		return MediaAlternative{}
	}
	return MediaAlternative{
		Name:      item.Name,
		URL:       item.URL,
		Source:    item.Source,
		MediaType: normalizeMediaType(item),
	}
}

func summarizeAlternativeLanguages(items []MediaAlternative) (bool, bool, bool, bool) {
	hasPortuguese := false
	hasEnglish := false
	hasDub := false
	hasSub := false
	for _, item := range items {
		pt, en, dub, sub := detectAlternativeLanguage(item)
		hasPortuguese = hasPortuguese || pt
		hasEnglish = hasEnglish || en
		hasDub = hasDub || dub
		hasSub = hasSub || sub
	}
	return hasPortuguese, hasEnglish, hasDub, hasSub
}

func detectAlternativeLanguage(item MediaAlternative) (bool, bool, bool, bool) {
	source := strings.ToLower(strings.TrimSpace(item.Source))
	name := strings.ToLower(strings.TrimSpace(item.Name))
	isDub := strings.Contains(name, "dublado") || strings.Contains(name, "[dub]") || strings.Contains(name, "[portuguese]") || strings.Contains(name, "[portugues]")
	isSub := strings.Contains(name, "legendado") || strings.Contains(name, "[sub]") || strings.Contains(name, "[english]")
	isPortuguese := isDub || strings.Contains(name, "portuguese") || strings.Contains(name, "portugues") || source == "animefire" || source == "animesonlinecc" || source == "anroll"
	isEnglish := strings.Contains(name, "english") || source == "allanime" || (!isPortuguese && !isDub)
	if isPortuguese && !isDub && !isSub {
		isSub = true
	}
	if isEnglish && !isDub {
		isSub = true
	}
	return isPortuguese, isEnglish, isDub, isSub
}

func appendUniqueAlternative(current []MediaAlternative, incoming MediaAlternative) []MediaAlternative {
	key := strings.ToLower(strings.TrimSpace(incoming.Source)) + "|" + strings.TrimSpace(incoming.URL)
	if key == "|" {
		return current
	}
	for _, item := range current {
		if strings.EqualFold(strings.TrimSpace(item.Source), strings.TrimSpace(incoming.Source)) && strings.TrimSpace(item.URL) == strings.TrimSpace(incoming.URL) {
			return current
		}
	}
	return append(current, incoming)
}

func buildOrderedMediaAlternatives(req MediaRequest, action string) []MediaAlternative {
	primary := MediaAlternative{
		Name:      req.Name,
		URL:       req.URL,
		Source:    req.Source,
		MediaType: req.MediaType,
	}
	items := append([]MediaAlternative{primary}, req.Alternatives...)
	deduped := make([]MediaAlternative, 0, len(items))
	for _, item := range items {
		deduped = appendUniqueAlternative(deduped, item)
	}
	groupKey := req.GroupKey
	if strings.TrimSpace(groupKey) == "" {
		groupKey = normalizeSearchText(cleanDisplayName(req.Name))
	}
	sort.SliceStable(deduped, func(i, j int) bool {
		left := animeActionScore(action, groupKey, deduped[i])
		right := animeActionScore(action, groupKey, deduped[j])
		if left != right {
			return left > right
		}
		return strings.ToLower(deduped[i].Source) < strings.ToLower(deduped[j].Source)
	})
	return deduped
}

func ctxGroupKey(ctx *animeSearchContext) string {
	if ctx == nil {
		return ""
	}
	base := ctx.CanonicalTitle
	if strings.TrimSpace(base) == "" && len(ctx.Aliases) > 0 {
		base = ctx.Aliases[0]
	}
	key := normalizeSearchText(base)
	if ctx.SeasonNumber > 0 {
		key = fmt.Sprintf("%s#s%d", key, ctx.SeasonNumber)
	}
	return key
}

func shouldUseRelatedTitles(ctx *animeSearchContext) bool {
	if ctx == nil {
		return false
	}
	query := strings.ToLower(strings.TrimSpace(ctx.Query))
	return ctx.SeasonNumber > 1 || strings.Contains(query, "season") || strings.Contains(query, "temporada") || strings.Contains(query, "parte") || strings.Contains(query, "part ")
}

func isRelevantRelationType(value string) bool {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "SEQUEL", "PREQUEL", "SIDE_STORY", "ALTERNATIVE", "PARENT", "PARENT_STORY":
		return true
	default:
		return false
	}
}

func extractSeasonNumber(value string) int {
	value = strings.TrimSpace(value)
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:season|temporada|part|parte)\s*(\d+)`),
		regexp.MustCompile(`(?i)(\d+)(?:st|nd|rd|th)\s+season`),
	}
	for _, pattern := range patterns {
		if match := pattern.FindStringSubmatch(value); len(match) > 1 {
			if n, err := strconv.Atoi(match[1]); err == nil && n > 0 {
				return n
			}
		}
	}
	roman := regexp.MustCompile(`\b(II|III|IV|V|VI|VII|VIII|IX|X)\b\s*$`).FindStringSubmatch(strings.ToUpper(value))
	if len(roman) > 1 {
		return romanToInt(roman[1])
	}
	return 0
}

func romanToInt(value string) int {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "I":
		return 1
	case "II":
		return 2
	case "III":
		return 3
	case "IV":
		return 4
	case "V":
		return 5
	case "VI":
		return 6
	case "VII":
		return 7
	case "VIII":
		return 8
	case "IX":
		return 9
	case "X":
		return 10
	default:
		return 0
	}
}

func titleSimilarityScore(a string, b string) int {
	aNorm := normalizeSearchText(api.CleanTitle(a))
	bNorm := normalizeSearchText(api.CleanTitle(b))
	if aNorm == "" || bNorm == "" {
		return 0
	}
	switch {
	case aNorm == bNorm:
		return 1000
	case strings.Contains(aNorm, bNorm):
		return 760 - len(aNorm) + len(bNorm)
	case strings.Contains(bNorm, aNorm):
		return 700 - len(bNorm) + len(aNorm)
	default:
		return 300 + wordOverlapScore(aNorm, bNorm)
	}
}

func wordOverlapScore(a string, b string) int {
	wordsA := strings.Fields(a)
	wordsB := strings.Fields(b)
	if len(wordsA) == 0 || len(wordsB) == 0 {
		return 0
	}
	setB := make(map[string]bool, len(wordsB))
	for _, word := range wordsB {
		setB[word] = true
	}
	overlap := 0
	for _, word := range wordsA {
		if setB[word] {
			overlap += 40
		}
	}
	return overlap
}

func normalizeSearchText(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = regexp.MustCompile(`[^\p{L}\p{N}\s]+`).ReplaceAllString(value, " ")
	value = regexp.MustCompile(`\s+`).ReplaceAllString(value, " ")
	return strings.TrimSpace(value)
}

func cleanDisplayName(value string) string {
	return strings.TrimSpace(api.CleanTitle(value))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
