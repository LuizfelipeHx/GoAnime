package scraper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/alvarorichard/Goanime/internal/models"
	"github.com/alvarorichard/Goanime/internal/util"
)

const (
	AnrollBase = "https://www.anroll.tv"
)

type AnrollClient struct {
	client     *http.Client
	baseURL    string
	maxRetries int
	retryDelay time.Duration
}

func NewAnrollClient() *AnrollClient {
	return &AnrollClient{
		client:     util.GetFastClient(),
		baseURL:    AnrollBase,
		maxRetries: 2,
		retryDelay: 300 * time.Millisecond,
	}
}

func (c *AnrollClient) SearchAnime(query string) ([]*models.Anime, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("query vazia")
	}

	results, err := c.searchWordPress(query)
	if err == nil && len(results) > 0 {
		return results, nil
	}

	results, fallbackErr := c.searchCatalogPages(query)
	if fallbackErr != nil {
		if err != nil {
			return nil, fmt.Errorf("falha na busca: %v | fallback: %w", err, fallbackErr)
		}
		return nil, fmt.Errorf("falha na busca: %w", fallbackErr)
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("nenhum anime encontrado para: %s", query)
	}
	return results, nil
}

func (c *AnrollClient) searchWordPress(query string) ([]*models.Anime, error) {
	searchURLs := []string{
		fmt.Sprintf("%s/?s=%s&post_type=animes", c.baseURL, url.QueryEscape(query)),
		fmt.Sprintf("%s/animes/?s=%s", c.baseURL, url.QueryEscape(query)),
	}

	for _, searchURL := range searchURLs {
		doc, err := c.fetchPage(searchURL, c.baseURL+"/")
		if err != nil {
			continue
		}
		results := c.extractSearchResults(doc, query)
		if len(results) > 0 {
			return results, nil
		}
	}

	return nil, fmt.Errorf("nenhum resultado no search do wordpress")
}

func (c *AnrollClient) searchCatalogPages(query string) ([]*models.Anime, error) {
	pageURLs := []string{
		c.baseURL + "/animes/",
		c.baseURL + "/animes/page/2/",
		c.baseURL + "/animes/page/3/",
	}

	seen := make(map[string]bool)
	var results []*models.Anime
	for _, pageURL := range pageURLs {
		doc, err := c.fetchPage(pageURL, c.baseURL+"/animes/")
		if err != nil {
			continue
		}
		for _, item := range c.extractSearchResults(doc, query) {
			if item == nil {
				continue
			}
			key := strings.TrimSpace(item.URL)
			if key == "" || seen[key] {
				continue
			}
			seen[key] = true
			results = append(results, item)
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("nenhum anime encontrado no catálogo")
	}
	sortAnimeResultsByMatch(results, query)
	return results, nil
}

func (c *AnrollClient) extractSearchResults(doc *goquery.Document, query string) []*models.Anime {
	seen := make(map[string]bool)
	results := make([]*models.Anime, 0)

	doc.Find("article.item.tvshows, article.item").Each(func(_ int, s *goquery.Selection) {
		link := s.Find(".poster a, .data h3 a").First()
		href, exists := link.Attr("href")
		if !exists || href == "" || !strings.Contains(href, "/animes/") {
			return
		}

		title := strings.TrimSpace(s.Find(".data h3 a").First().Text())
		if title == "" {
			title = strings.TrimSpace(s.Find(".poster img").First().AttrOr("alt", ""))
		}
		if title == "" {
			return
		}

		matchScore := simpleTitleScore(title, query)
		if matchScore <= 0 {
			return
		}

		imageURL := strings.TrimSpace(s.Find(".poster img").First().AttrOr("src", ""))
		year := extractYear(s.Find(".data span").First().Text())
		key := strings.TrimSpace(href)
		if seen[key] {
			return
		}
		seen[key] = true

		results = append(results, &models.Anime{
			Name:      title,
			URL:       href,
			ImageURL:  imageURL,
			Source:    "Anroll",
			MediaType: models.MediaTypeAnime,
			Year:      year,
		})
	})

	sortAnimeResultsByMatch(results, query)
	return results
}

func (c *AnrollClient) GetAnimeEpisodes(animeURL string) ([]models.Episode, error) {
	doc, err := c.fetchPage(animeURL, c.baseURL+"/")
	if err != nil {
		return nil, fmt.Errorf("falha ao buscar episódios: %w", err)
	}

	seasonBlocks := doc.Find("div.se-c")
	if seasonBlocks.Length() == 0 {
		return c.extractFlatEpisodes(doc)
	}

	var episodes []models.Episode
	sequence := 1
	seasonRE := regexp.MustCompile(`(\d+)\s*-\s*(\d+)`)

	seasonBlocks.Each(func(_ int, block *goquery.Selection) {
		seasonNumber := parseInt(block.Find(".se-q .se-t").First().Text())
		block.Find("ul.episodios li").Each(func(_ int, item *goquery.Selection) {
			link := item.Find(".episodiotitle a").First()
			href, exists := link.Attr("href")
			if !exists || href == "" {
				return
			}

			label := strings.TrimSpace(link.Text())
			if label == "" {
				label = fmt.Sprintf("Episódio %d", sequence)
			}

			localEpisode := 0
			if match := seasonRE.FindStringSubmatch(item.Find(".numerando").First().Text()); len(match) == 3 {
				if seasonNumber == 0 {
					seasonNumber = parseInt(match[1])
				}
				localEpisode = parseInt(match[2])
			}
			if localEpisode == 0 {
				localEpisode = parseEpisodeFromURL(href)
			}
			if seasonNumber > 0 {
				label = fmt.Sprintf("T%d · %s", seasonNumber, label)
			}

			episodes = append(episodes, models.Episode{
				Number: strconv.Itoa(sequence),
				Num:    sequence,
				URL:    href,
				Title:  models.TitleDetails{Romaji: label},
				Aired:  strings.TrimSpace(item.Find(".episodiotitle .date").First().Text()),
			})
			sequence++
		})
	})

	if len(episodes) == 0 {
		return nil, fmt.Errorf("nenhum episódio encontrado em: %s", animeURL)
	}
	return episodes, nil
}

func (c *AnrollClient) extractFlatEpisodes(doc *goquery.Document) ([]models.Episode, error) {
	seen := make(map[string]bool)
	var episodes []models.Episode
	sequence := 1

	doc.Find("a[href*='/episodios/']").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || href == "" || seen[href] {
			return
		}
		seen[href] = true

		label := strings.TrimSpace(s.Text())
		if label == "" {
			episodeNum := parseEpisodeFromURL(href)
			if episodeNum > 0 {
				label = fmt.Sprintf("Episódio %d", episodeNum)
			} else {
				label = fmt.Sprintf("Episódio %d", sequence)
			}
		}

		episodes = append(episodes, models.Episode{
			Number: strconv.Itoa(sequence),
			Num:    sequence,
			URL:    href,
			Title:  models.TitleDetails{Romaji: label},
		})
		sequence++
	})

	if len(episodes) == 0 {
		return nil, fmt.Errorf("nenhum episódio encontrado")
	}
	return episodes, nil
}

func (c *AnrollClient) GetEpisodeStreamURL(episodeURL string) (string, error) {
	doc, err := c.fetchPage(episodeURL, c.baseURL+"/")
	if err != nil {
		return "", fmt.Errorf("falha ao carregar episodio do Anroll: %w", err)
	}

	options, err := c.extractPlayerOptions(doc)
	if err != nil {
		return "", err
	}

	var failures []string
	for _, option := range options {
		embedURL, err := c.fetchPlayerEmbedURL(episodeURL, option)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", option.Label, err))
			continue
		}

		streamURL, err := extractURLWithYtDlp(embedURL)
		if err == nil && strings.TrimSpace(streamURL) != "" {
			return streamURL, nil
		}
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", option.Label, err))
		}
	}

	if len(failures) > 0 {
		return "", fmt.Errorf("falha ao resolver stream do Anroll: %s", strings.Join(failures, " | "))
	}
	return "", fmt.Errorf("nenhuma fonte de player disponivel em %s", episodeURL)
}

type anrollPlayerOption struct {
	PostID    string
	Nume      string
	MediaType string
	Label     string
}

type anrollPlayerResponse struct {
	EmbedURL string `json:"embed_url"`
	Type     string `json:"type"`
}

func (c *AnrollClient) extractPlayerOptions(doc *goquery.Document) ([]anrollPlayerOption, error) {
	postID := strings.TrimSpace(doc.Find("meta#dooplay-ajax-counter").AttrOr("data-postid", ""))
	var options []anrollPlayerOption

	doc.Find("ul#playeroptionsul li.dooplay_player_option").Each(func(_ int, item *goquery.Selection) {
		nume := strings.TrimSpace(item.AttrOr("data-nume", ""))
		if nume == "" {
			return
		}

		optionPostID := strings.TrimSpace(item.AttrOr("data-post", postID))
		if optionPostID == "" {
			return
		}

		mediaType := strings.TrimSpace(item.AttrOr("data-type", "tv"))
		label := strings.TrimSpace(item.Find("span.title").First().Text())
		if label == "" {
			label = strings.TrimSpace(item.Text())
		}
		if label == "" {
			label = "Player " + nume
		}

		options = append(options, anrollPlayerOption{
			PostID:    optionPostID,
			Nume:      nume,
			MediaType: mediaType,
			Label:     label,
		})
	})

	if len(options) == 0 {
		return nil, fmt.Errorf("nenhuma opcao de player encontrada")
	}
	return options, nil
}

func (c *AnrollClient) fetchPlayerEmbedURL(referer string, option anrollPlayerOption) (string, error) {
	form := url.Values{}
	form.Set("action", "doo_player_ajax")
	form.Set("post", option.PostID)
	form.Set("nume", option.Nume)
	form.Set("type", option.MediaType)

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/wp-admin/admin-ajax.php", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("Accept-Language", "pt-BR,pt;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Origin", c.baseURL)
	req.Header.Set("Referer", referer)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d no player ajax", resp.StatusCode)
	}

	var payload anrollPlayerResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("resposta invalida do player ajax: %w", err)
	}
	embedURL := strings.TrimSpace(payload.EmbedURL)

	// Dooplay às vezes retorna HTML de iframe em vez de URL direta
	if strings.Contains(embedURL, "<iframe") || strings.Contains(embedURL, "<IFRAME") {
		re := regexp.MustCompile(`(?i)<iframe[^>]+\bsrc=["']([^"']+)["']`)
		if m := re.FindStringSubmatch(embedURL); len(m) == 2 {
			embedURL = strings.TrimSpace(m[1])
		}
	}

	// Normalizar URLs protocol-relative (//player.com/... → https://player.com/...)
	if strings.HasPrefix(embedURL, "//") {
		embedURL = "https:" + embedURL
	}

	if embedURL == "" {
		return "", fmt.Errorf("embed_url vazio para %s", option.Label)
	}
	return embedURL, nil
}

func (c *AnrollClient) fetchPage(pageURL, referer string) (*goquery.Document, error) {
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryDelay)
		}

		req, err := http.NewRequest(http.MethodGet, pageURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0 Safari/537.36")
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		req.Header.Set("Accept-Language", "pt-BR,pt;q=0.9,en-US;q=0.8,en;q=0.7")
		if referer != "" {
			req.Header.Set("Referer", referer)
		}

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode != http.StatusOK {
			_ = resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %d para %s", resp.StatusCode, pageURL)
			continue
		}

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}
		return doc, nil
	}
	return nil, lastErr
}

func sortAnimeResultsByMatch(items []*models.Anime, query string) {
	sort.SliceStable(items, func(i, j int) bool {
		left := simpleTitleScore(items[i].Name, query)
		right := simpleTitleScore(items[j].Name, query)
		if left != right {
			return left > right
		}
		return strings.ToLower(strings.TrimSpace(items[i].Name)) < strings.ToLower(strings.TrimSpace(items[j].Name))
	})
}

func simpleTitleScore(title string, query string) int {
	titleNorm := normalizeLoose(title)
	queryNorm := normalizeLoose(query)
	if titleNorm == "" || queryNorm == "" {
		return 0
	}
	if titleNorm == queryNorm {
		return 1000
	}
	if strings.Contains(titleNorm, queryNorm) {
		return 800
	}

	score := 0
	for _, token := range strings.Fields(queryNorm) {
		if strings.Contains(titleNorm, token) {
			score += 120
		}
	}
	if score == 0 {
		return 0
	}
	return score
}

func normalizeLoose(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(
		":", " ",
		"-", " ",
		"_", " ",
		"(", " ",
		")", " ",
		"[", " ",
		"]", " ",
		".", " ",
		",", " ",
		"'", "",
		"’", "",
	)
	value = replacer.Replace(value)
	return strings.Join(strings.Fields(value), " ")
}

func extractYear(value string) string {
	re := regexp.MustCompile(`\b(19|20)\d{2}\b`)
	return strings.TrimSpace(re.FindString(value))
}

func parseEpisodeFromURL(value string) int {
	re := regexp.MustCompile(`(?:episodio|ep)[-_ ]?(\d+)\b`)
	match := re.FindStringSubmatch(strings.ToLower(value))
	if len(match) != 2 {
		return 0
	}
	return parseInt(match[1])
}

func parseInt(value string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(value))
	return n
}
