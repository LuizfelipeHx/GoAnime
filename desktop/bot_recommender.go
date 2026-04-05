package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/alvarorichard/Goanime/internal/ai"
)

const (
	recsRefreshInterval = 24 * time.Hour
	maxRecommendations  = 10
)

type recommenderBot struct {
	aiClient *ai.Client
	tracker  func() []WatchProgressEntry // getter for watch history

	mu        sync.RWMutex
	recs      []AIRecommendation
	lastRun   time.Time
	available bool
}

func newRecommenderBot(aiClient *ai.Client, historyFn func() []WatchProgressEntry) *recommenderBot {
	return &recommenderBot{
		aiClient: aiClient,
		tracker:  historyFn,
	}
}

func (b *recommenderBot) refresh(ctx context.Context) {
	if !b.aiClient.IsAvailable() {
		b.mu.Lock()
		b.available = false
		b.mu.Unlock()
		return
	}

	// Check if we already have fresh recs
	b.mu.RLock()
	if len(b.recs) > 0 && time.Since(b.lastRun) < recsRefreshInterval {
		b.mu.RUnlock()
		return
	}
	b.mu.RUnlock()

	// Load from cache first
	cached := b.loadCache()
	if len(cached.Recs) > 0 && time.Since(cached.UpdatedAt) < recsRefreshInterval {
		b.mu.Lock()
		b.recs = cached.Recs
		b.lastRun = cached.UpdatedAt
		b.available = true
		b.mu.Unlock()
		return
	}

	history := b.tracker()
	if len(history) < 2 {
		return
	}

	// Build list of watched titles
	var titles []string
	seen := make(map[string]bool)
	for _, h := range history {
		t := strings.TrimSpace(h.Title)
		if t == "" || seen[strings.ToLower(t)] {
			continue
		}
		seen[strings.ToLower(t)] = true
		titles = append(titles, t)
		if len(titles) >= 15 {
			break
		}
	}

	systemPrompt := "Voc\u00ea \u00e9 um especialista em anime. O usu\u00e1rio vai te dar uma lista de animes que assistiu.\n" +
		"Recomende exatamente 5 animes que ele provavelmente vai gostar, baseado nos g\u00eaneros e temas em comum.\n\n" +
		"Responda APENAS em JSON v\u00e1lido, neste formato exato:\n" +
		"[\n  {\"title\": \"Nome do Anime\", \"reason\": \"Motivo curto em portugu\u00eas\", \"genres\": \"G\u00eanero1, G\u00eanero2\"},\n  ...\n]\n\n" +
		"N\u00e3o inclua animes que o usu\u00e1rio j\u00e1 assistiu. Responda APENAS o JSON, sem texto adicional."

	userPrompt := fmt.Sprintf("Animes que eu assisti:\n%s", strings.Join(titles, "\n"))

	aiCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	response, err := b.aiClient.Chat(aiCtx, systemPrompt, userPrompt)
	if err != nil {
		log.Printf("[bot:recommender] AI error: %v", err)
		b.mu.Lock()
		b.available = false
		b.mu.Unlock()
		return
	}

	// Parse JSON from response (handle markdown code blocks)
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "```") {
		lines := strings.Split(response, "\n")
		var cleaned []string
		for _, l := range lines {
			if strings.HasPrefix(strings.TrimSpace(l), "```") {
				continue
			}
			cleaned = append(cleaned, l)
		}
		response = strings.Join(cleaned, "\n")
	}

	var recs []AIRecommendation
	if err := json.Unmarshal([]byte(response), &recs); err != nil {
		truncated := response
		if len(truncated) > 200 {
			truncated = truncated[:200]
		}
		log.Printf("[bot:recommender] parse error: %v (response: %s)", err, truncated)
		return
	}

	if len(recs) > maxRecommendations {
		recs = recs[:maxRecommendations]
	}

	b.mu.Lock()
	b.recs = recs
	b.lastRun = time.Now()
	b.available = true
	b.mu.Unlock()

	b.saveCache(recsCache{Recs: recs, UpdatedAt: time.Now()})
	log.Printf("[bot:recommender] generated %d recommendations", len(recs))
}

func (b *recommenderBot) getRecs() []AIRecommendation {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]AIRecommendation, len(b.recs))
	copy(out, b.recs)
	return out
}

func (b *recommenderBot) isAvailable() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.available
}

type recsCache struct {
	Recs      []AIRecommendation `json:"recs"`
	UpdatedAt time.Time          `json:"updatedAt"`
}

func recsCachePath() string {
	if runtime.GOOS == "windows" {
		if d := os.Getenv("LOCALAPPDATA"); d != "" {
			return filepath.Join(d, "GoAnime", "bot_recommendations.json")
		}
	}
	if h := os.Getenv("HOME"); h != "" {
		return filepath.Join(h, ".local", "goanime", "bot_recommendations.json")
	}
	return ""
}

func (b *recommenderBot) loadCache() recsCache {
	p := recsCachePath()
	if p == "" {
		return recsCache{}
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return recsCache{}
	}
	var c recsCache
	_ = json.Unmarshal(data, &c)
	return c
}

func (b *recommenderBot) saveCache(c recsCache) {
	p := recsCachePath()
	if p == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return
	}
	data, _ := json.MarshalIndent(c, "", "  ")
	_ = os.WriteFile(p, data, 0o644)
}
