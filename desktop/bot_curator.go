package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/alvarorichard/Goanime/internal/ai"
)

const maxCuratedReleases = 15

type curatorBot struct {
	aiClient    *ai.Client
	releasesBot *releasesBot

	mu      sync.RWMutex
	curated []CuratedRelease
	lastRun time.Time
}

func newCuratorBot(aiClient *ai.Client, relBot *releasesBot) *curatorBot {
	return &curatorBot{
		aiClient:    aiClient,
		releasesBot: relBot,
	}
}

func (b *curatorBot) refresh(ctx context.Context) {
	if !b.aiClient.IsAvailable() {
		return
	}

	releases := b.releasesBot.getReleases()
	if len(releases) == 0 {
		return
	}

	// Take up to 20 most recent for evaluation
	evalCount := min(20, len(releases))
	toEval := releases[:evalCount]

	var titles []string
	for i, r := range toEval {
		titles = append(titles, fmt.Sprintf("%d. %s (Seeders: %d, Tamanho: %s)", i+1, r.Title, r.Seeders, r.Size))
	}

	systemPrompt := "Voc\u00ea \u00e9 um curador de anime PT-BR. Avalie cada lan\u00e7amento por qualidade.\n\n" +
		"Crit\u00e9rios:\n" +
		"- Grupo de fansub conhecido = melhor\n" +
		"- 1080p > 720p > 480p\n" +
		"- Mais seeders = mais confi\u00e1vel\n" +
		"- Legendas PT-BR no t\u00edtulo = b\u00f4nus\n\n" +
		"Classifique cada um como: \"Excelente\", \"Bom\" ou \"Regular\"\n" +
		"Responda APENAS em JSON v\u00e1lido:\n" +
		"[\n  {\"index\": 1, \"quality\": \"Excelente\", \"summary\": \"Motivo curto\"},\n  ...\n]\n\n" +
		"Avalie TODOS os itens. Responda APENAS o JSON."

	userPrompt := fmt.Sprintf("Avalie estes lan\u00e7amentos:\n%s", strings.Join(titles, "\n"))

	aiCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	response, err := b.aiClient.Chat(aiCtx, systemPrompt, userPrompt)
	if err != nil {
		log.Printf("[bot:curator] AI error: %v", err)
		return
	}

	// Clean markdown
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

	var ratings []struct {
		Index   int    `json:"index"`
		Quality string `json:"quality"`
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal([]byte(response), &ratings); err != nil {
		log.Printf("[bot:curator] parse error: %v", err)
		return
	}

	type qualitySummary struct {
		quality string
		summary string
	}
	ratingMap := make(map[int]qualitySummary)
	for _, r := range ratings {
		ratingMap[r.Index] = qualitySummary{r.Quality, r.Summary}
	}

	var curated []CuratedRelease
	for i, release := range toEval {
		r, ok := ratingMap[i+1]
		if !ok {
			r = qualitySummary{"Regular", "Sem avalia\u00e7\u00e3o"}
		}
		curated = append(curated, CuratedRelease{
			Release: release,
			Quality: r.quality,
			Summary: r.summary,
		})
	}

	// Sort: Excelente first, then Bom, then Regular
	qualityOrder := map[string]int{"Excelente": 0, "Bom": 1, "Regular": 2}
	sort.Slice(curated, func(i, j int) bool {
		return qualityOrder[curated[i].Quality] < qualityOrder[curated[j].Quality]
	})

	if len(curated) > maxCuratedReleases {
		curated = curated[:maxCuratedReleases]
	}

	b.mu.Lock()
	b.curated = curated
	b.lastRun = time.Now()
	b.mu.Unlock()

	log.Printf("[bot:curator] curated %d releases", len(curated))
}

func (b *curatorBot) getCurated() []CuratedRelease {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]CuratedRelease, len(b.curated))
	copy(out, b.curated)
	return out
}
