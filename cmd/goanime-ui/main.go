package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/alvarorichard/Goanime/internal/api"
	"github.com/alvarorichard/Goanime/internal/models"
	"github.com/alvarorichard/Goanime/internal/player"
	"github.com/alvarorichard/Goanime/internal/scraper"
	"github.com/alvarorichard/Goanime/internal/util"
)

const (
	defaultAddr     = ":8090"
	maxSearchItems  = 72
	maxEpisodeItems = 300
)

//go:embed static/*
var staticFS embed.FS

type mediaResult struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	ImageURL  string `json:"imageUrl"`
	Source    string `json:"source"`
	MediaType string `json:"mediaType"`
	Year      string `json:"year"`
}

type searchResponse struct {
	Query   string        `json:"query"`
	Count   int           `json:"count"`
	Results []mediaResult `json:"results"`
}

type episodeResult struct {
	Number string `json:"number"`
	Num    int    `json:"num"`
	Title  string `json:"title"`
	URL    string `json:"url"`
}

type episodesResponse struct {
	Name      string          `json:"name"`
	Source    string          `json:"source"`
	MediaType string          `json:"mediaType"`
	Count     int             `json:"count"`
	Episodes  []episodeResult `json:"episodes"`
}

type subtitleResponse struct {
	URL      string `json:"url"`
	ProxyURL string `json:"proxyUrl"`
	Language string `json:"language"`
	Label    string `json:"label"`
}

type streamResponse struct {
	StreamURL   string             `json:"streamUrl"`
	ProxyURL    string             `json:"proxyUrl"`
	ContentType string             `json:"contentType"`
	Subtitles   []subtitleResponse `json:"subtitles,omitempty"`
	Note        string             `json:"note,omitempty"`
}

type mediaRequest struct {
	Name      string
	URL       string
	Source    string
	MediaType string
}

func main() {
	manager := scraper.NewScraperManager()
	httpClient := util.GetFastClient()
	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
	})
	mux.HandleFunc("/api/search", makeSearchHandler(manager))
	mux.HandleFunc("/api/episodes", makeEpisodesHandler())
	mux.HandleFunc("/api/stream", makeStreamHandler())
	mux.HandleFunc("/api/proxy", makeProxyHandler(httpClient))

	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatalf("failed to load static assets: %v", err)
	}
	mux.Handle("/", http.FileServer(http.FS(sub)))

	log.Printf("GoAnime UI running at http://localhost%s", defaultAddr)
	if err := http.ListenAndServe(defaultAddr, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func makeSearchHandler(manager *scraper.ScraperManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}

		query := strings.TrimSpace(r.URL.Query().Get("q"))
		if len(query) < 2 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "query must have at least 2 characters"})
			return
		}

		sourceArg := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("source")))
		typeArg := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("type")))
		sourceType, sourceErr := parseSource(sourceArg)
		if sourceErr != "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": sourceErr})
			return
		}

		results, err := manager.SearchAnime(query, sourceType)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}

		filtered := filterByType(results, typeArg)
		sort.Slice(filtered, func(i, j int) bool {
			if filtered[i].Source == filtered[j].Source {
				return strings.ToLower(filtered[i].Name) < strings.ToLower(filtered[j].Name)
			}
			return strings.ToLower(filtered[i].Source) < strings.ToLower(filtered[j].Source)
		})

		respItems := make([]mediaResult, 0, min(len(filtered), maxSearchItems))
		for _, item := range filtered {
			if len(respItems) >= maxSearchItems {
				break
			}

			respItems = append(respItems, mediaResult{
				Name:      item.Name,
				URL:       item.URL,
				ImageURL:  item.ImageURL,
				Source:    item.Source,
				MediaType: normalizeMediaType(item),
				Year:      item.Year,
			})
		}

		writeJSON(w, http.StatusOK, searchResponse{
			Query:   query,
			Count:   len(respItems),
			Results: respItems,
		})
	}
}

func makeEpisodesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}

		mediaReq, err := parseMediaRequest(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if mediaReq.MediaType == "tv" && strings.Contains(strings.ToLower(mediaReq.Source), "flixhq") {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "TV episodes from FlixHQ are not yet available in web mode"})
			return
		}

		anime := &models.Anime{
			Name:      mediaReq.Name,
			URL:       mediaReq.URL,
			Source:    mediaReq.Source,
			MediaType: parseMediaType(mediaReq.MediaType),
		}

		episodes, err := api.GetAnimeEpisodesEnhanced(anime)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}

		items := make([]episodeResult, 0, min(len(episodes), maxEpisodeItems))
		for _, ep := range episodes {
			if len(items) >= maxEpisodeItems {
				break
			}

			title := strings.TrimSpace(ep.Title.English)
			if title == "" {
				title = strings.TrimSpace(ep.Title.Romaji)
			}
			if title == "" {
				title = ep.Number
			}

			num := ep.Num
			if num <= 0 {
				num = parseEpisodeNum(ep.Number)
			}

			items = append(items, episodeResult{
				Number: ep.Number,
				Num:    num,
				Title:  title,
				URL:    ep.URL,
			})
		}

		writeJSON(w, http.StatusOK, episodesResponse{
			Name:      mediaReq.Name,
			Source:    mediaReq.Source,
			MediaType: mediaReq.MediaType,
			Count:     len(items),
			Episodes:  items,
		})
	}
}

func makeStreamHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}

		mediaReq, err := parseMediaRequest(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		episodeURL := strings.TrimSpace(r.URL.Query().Get("episode_url"))
		if episodeURL == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "episode_url is required"})
			return
		}

		episodeNumber := strings.TrimSpace(r.URL.Query().Get("episode_number"))
		if episodeNumber == "" {
			episodeNumber = strconv.Itoa(max(1, parseEpisodeNum(episodeURL)))
		}

		quality := strings.TrimSpace(r.URL.Query().Get("quality"))
		if quality == "" {
			quality = "best"
		}
		mode := strings.TrimSpace(r.URL.Query().Get("mode"))
		if mode == "" {
			mode = "sub"
		}

		anime := &models.Anime{
			Name:      mediaReq.Name,
			URL:       mediaReq.URL,
			Source:    mediaReq.Source,
			MediaType: parseMediaType(mediaReq.MediaType),
		}
		episode := &models.Episode{
			URL:    episodeURL,
			Number: episodeNumber,
			Num:    parseEpisodeNum(episodeNumber),
		}

		if anime.MediaType == models.MediaTypeMovie && strings.Contains(strings.ToLower(anime.Source), "flixhq") {
			streamURL, subtitles, streamErr := api.GetFlixHQStreamURL(anime, episode, quality)
			if streamErr != nil {
				writeJSON(w, http.StatusBadGateway, map[string]string{"error": streamErr.Error()})
				return
			}

			responseSubs := make([]subtitleResponse, 0, len(subtitles))
			for _, sub := range subtitles {
				if strings.TrimSpace(sub.URL) == "" {
					continue
				}
				responseSubs = append(responseSubs, subtitleResponse{
					URL:      sub.URL,
					ProxyURL: buildProxyURL(sub.URL),
					Language: sub.Language,
					Label:    sub.Label,
				})
			}

			writeJSON(w, http.StatusOK, streamResponse{
				StreamURL:   streamURL,
				ProxyURL:    buildProxyURL(streamURL),
				ContentType: detectContentType(streamURL),
				Subtitles:   responseSubs,
			})
			return
		}

		if isAllAnimeMedia(anime) {
			animeID := extractAllAnimeID(anime.URL)
			if animeID == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "could not resolve AllAnime ID"})
				return
			}

			client := scraper.NewAllAnimeClient()
			streamURL, _, streamErr := client.GetEpisodeURL(animeID, episodeNumber, mode, quality)
			if streamErr != nil {
				writeJSON(w, http.StatusBadGateway, map[string]string{"error": streamErr.Error()})
				return
			}

			writeJSON(w, http.StatusOK, streamResponse{
				StreamURL:   streamURL,
				ProxyURL:    buildProxyURL(streamURL),
				ContentType: detectContentType(streamURL),
			})
			return
		}

		streamURL, streamErr := player.GetVideoURLForEpisodeEnhanced(episode, anime)
		if streamErr != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": streamErr.Error()})
			return
		}

		writeJSON(w, http.StatusOK, streamResponse{
			StreamURL:   streamURL,
			ProxyURL:    buildProxyURL(streamURL),
			ContentType: detectContentType(streamURL),
		})
	}
}

func makeProxyHandler(client *http.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}

		targetRaw := strings.TrimSpace(r.URL.Query().Get("u"))
		if targetRaw == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "u query param is required"})
			return
		}

		targetURL, err := url.Parse(targetRaw)
		if err != nil || targetURL.Host == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid target URL"})
			return
		}
		if targetURL.Scheme != "http" && targetURL.Scheme != "https" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "only http/https targets are supported"})
			return
		}

		upstreamReq, _ := http.NewRequest(http.MethodGet, targetURL.String(), nil)
		upstreamReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0 Safari/537.36")
		upstreamReq.Header.Set("Accept", "*/*")
		upstreamReq.Header.Set("Referer", targetURL.Scheme+"://"+targetURL.Host+"/")
		if rangeHeader := r.Header.Get("Range"); rangeHeader != "" {
			upstreamReq.Header.Set("Range", rangeHeader)
		}

		resp, reqErr := client.Do(upstreamReq)
		if reqErr != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": reqErr.Error()})
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
				writeJSON(w, http.StatusBadGateway, map[string]string{"error": readErr.Error()})
				return
			}

			rewritten := rewriteM3U8(string(body), targetURL)
			w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
			w.WriteHeader(resp.StatusCode)
			_, _ = io.WriteString(w, rewritten)
			return
		}

		copyProxyHeaders(w.Header(), resp.Header)
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	}
}

func rewriteM3U8(content string, baseURL *url.URL) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if strings.HasPrefix(trimmed, "#") {
			if strings.Contains(trimmed, "URI=\"") {
				lines[i] = rewriteM3U8TagURI(line, baseURL)
			}
			continue
		}

		resolved := resolveReference(baseURL, trimmed)
		lines[i] = buildProxyURL(resolved)
	}

	return strings.Join(lines, "\n")
}

func rewriteM3U8TagURI(line string, baseURL *url.URL) string {
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
	proxied := buildProxyURL(resolved)

	return line[:valueStart] + proxied + line[valueEnd:]
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

func parseMediaRequest(r *http.Request) (*mediaRequest, error) {
	mediaURL := strings.TrimSpace(r.URL.Query().Get("media_url"))
	if mediaURL == "" {
		mediaURL = strings.TrimSpace(r.URL.Query().Get("url"))
	}
	if mediaURL == "" {
		return nil, fmt.Errorf("media_url is required")
	}

	source := normalizeSource(strings.TrimSpace(r.URL.Query().Get("source")))
	mediaType := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("media_type")))
	if mediaType == "" {
		mediaType = strings.ToLower(strings.TrimSpace(r.URL.Query().Get("type")))
	}
	if mediaType == "" {
		if strings.Contains(strings.ToLower(source), "flixhq") {
			mediaType = "movie"
		} else {
			mediaType = "anime"
		}
	}

	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		name = "Selected media"
	}

	return &mediaRequest{
		Name:      name,
		URL:       mediaURL,
		Source:    source,
		MediaType: mediaType,
	}, nil
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
	case strings.Contains(lower, "animedrive"):
		return "AnimeDrive"
	case strings.Contains(lower, "animesonlinecc"):
		return "AnimesOnlineCC"
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
	default:
		return nil, "invalid source; use: all, allanime, animefire, flixhq, animesonlinecc"
	}
}

func filterByType(items []*models.Anime, typeArg string) []*models.Anime {
	if typeArg == "" || typeArg == "all" {
		return items
	}

	out := make([]*models.Anime, 0, len(items))
	for _, item := range items {
		mediaType := normalizeMediaType(item)
		if mediaType == typeArg {
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

func buildProxyURL(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	return "/api/proxy?u=" + url.QueryEscape(raw)
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

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("json encode error: %v", err)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
