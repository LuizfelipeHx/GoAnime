// Package scraper provides web scraping functionality for animesonlinecc.to
package scraper

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/alvarorichard/Goanime/internal/models"
	"github.com/alvarorichard/Goanime/internal/util"
	"github.com/lrstanley/go-ytdlp"
)

const (
	AnimesonlineccBase = "https://animesonlinecc.to"
)

// AnimesonlineccClient handles interactions with animesonlinecc.to
type AnimesonlineccClient struct {
	client     *http.Client
	baseURL    string
	maxRetries int
	retryDelay time.Duration
}

// NewAnimesonlineccClient creates a new animesonlinecc.to client
func NewAnimesonlineccClient() *AnimesonlineccClient {
	return &AnimesonlineccClient{
		client:     util.GetFastClient(),
		baseURL:    AnimesonlineccBase,
		maxRetries: 2,
		retryDelay: 300 * time.Millisecond,
	}
}

// SearchAnime searches for anime on animesonlinecc.to
func (c *AnimesonlineccClient) SearchAnime(query string) ([]*models.Anime, error) {
	searchURL := fmt.Sprintf("%s/?s=%s", c.baseURL, url.QueryEscape(query))
	util.Debug("AnimesOnlineCC search", "url", searchURL)

	doc, err := c.fetchPage(searchURL, c.baseURL+"/")
	if err != nil {
		return nil, fmt.Errorf("falha na busca: %w", err)
	}

	var results []*models.Anime
	doc.Find("article.item").Each(func(_ int, s *goquery.Selection) {
		link := s.Find(".poster a, .data h3 a").First()
		href, exists := link.Attr("href")
		if !exists || href == "" {
			return
		}
		title := strings.TrimSpace(s.Find(".data h3 a, .poster img").First().AttrOr("alt", link.Text()))
		if title == "" {
			title = strings.TrimSpace(link.Text())
		}
		imageURL, _ := s.Find(".poster img").Attr("src")

		results = append(results, &models.Anime{
			Name:     title,
			URL:      href,
			ImageURL: imageURL,
			Source:   "AnimesOnlineCC",
		})
	})

	if len(results) == 0 {
		return nil, fmt.Errorf("nenhum anime encontrado para: %s", query)
	}
	return results, nil
}

// GetAnimeEpisodes fetches the episode list for the given anime URL
func (c *AnimesonlineccClient) GetAnimeEpisodes(animeURL string) ([]models.Episode, error) {
	doc, err := c.fetchPage(animeURL, c.baseURL+"/")
	if err != nil {
		return nil, fmt.Errorf("falha ao buscar episódios: %w", err)
	}

	epNumRe := regexp.MustCompile(`episodio[-_](\d+)`)
	seen := make(map[string]bool)
	var episodes []models.Episode

	doc.Find("a[href*='/episodio/']").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || seen[href] {
			return
		}
		seen[href] = true

		label := strings.TrimSpace(s.Text())
		numStr := ""
		if m := epNumRe.FindStringSubmatch(href); len(m) >= 2 {
			numStr = m[1]
		}
		num := 0
		if n, err := strconv.Atoi(numStr); err == nil {
			num = n
		}
		if label == "" {
			label = "Episodio " + numStr
		}

		episodes = append(episodes, models.Episode{
			Number: numStr,
			Num:    num,
			URL:    href,
			Title:  models.TitleDetails{Romaji: label},
		})
	})

	if len(episodes) == 0 {
		return nil, fmt.Errorf("nenhum episódio encontrado em: %s", animeURL)
	}
	return episodes, nil
}

// GetEpisodeStreamURL extracts the playable video URL from an episode page.
// The site embeds videos via Blogger; yt-dlp is used to resolve the actual stream URL.
func (c *AnimesonlineccClient) GetEpisodeStreamURL(episodeURL string) (string, error) {
	doc, err := c.fetchPage(episodeURL, c.baseURL+"/")
	if err != nil {
		return "", fmt.Errorf("falha ao carregar episódio: %w", err)
	}

	// Extract Blogger embed URL from iframe
	bloggerURL, exists := doc.Find("iframe.metaframe, iframe[src*='blogger.com']").First().Attr("src")
	if !exists || bloggerURL == "" {
		// Fallback: any iframe
		bloggerURL, exists = doc.Find("iframe[src]").First().Attr("src")
		if !exists || bloggerURL == "" {
			return "", fmt.Errorf("player não encontrado na página: %s", episodeURL)
		}
	}

	util.Debugf("AnimesOnlineCC: usando player URL: %s", bloggerURL)

	// Use yt-dlp to extract the actual video stream URL from the embed
	return extractURLWithYtDlp(bloggerURL)
}

// extractURLWithYtDlp uses the installed yt-dlp binary to extract a direct video URL
// from a web page URL (e.g., Blogger embed) without downloading the file.
func extractURLWithYtDlp(pageURL string) (string, error) {
	ytdlpPath := util.FindYtDlpBinary()
	if ytdlpPath == "" {
		installCtx, installCancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer installCancel()

		if _, err := ytdlp.Install(installCtx, nil); err != nil {
			return "", fmt.Errorf("yt-dlp nao encontrado e a instalacao automatica falhou: %w", err)
		}

		ytdlpPath = util.FindYtDlpBinary()
		if ytdlpPath == "" {
			return "", fmt.Errorf("yt-dlp foi instalado, mas o binario nao foi localizado")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, ytdlpPath, "--no-warnings", "--no-playlist", "-g", "--format", "best", pageURL)
	// Hide console window on Windows to avoid visible CMD flashing
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
	}
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("yt-dlp nao conseguiu extrair URL de %s: %w", pageURL, err)
	}

	videoURL := strings.TrimSpace(string(out))
	if idx := strings.Index(videoURL, "\n"); idx > 0 {
		videoURL = videoURL[:idx]
	}
	if videoURL == "" {
		return "", fmt.Errorf("yt-dlp nao retornou URL para: %s", pageURL)
	}
	return videoURL, nil
}

// fetchPage fetches a URL and returns a goquery document
func (c *AnimesonlineccClient) fetchPage(pageURL, referer string) (*goquery.Document, error) {
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryDelay)
		}

		req, err := http.NewRequest("GET", pageURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
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
