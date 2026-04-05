package main

import (
	"fmt"
	"testing"

	"github.com/alvarorichard/Goanime/internal/scraper"
)

func TestSearchSourceDiscovery(t *testing.T) {
	mgr := scraper.NewScraperManager()

	query := "Summer Pockets"

	// Test each PT-BR source individually
	sources := []struct {
		name string
		st   scraper.ScraperType
	}{
		{"AnimeFire", scraper.AnimefireType},
		{"AnimesonlineCC", scraper.AnimesOnlineccType},
		{"Anroll", scraper.AnrollType},
		{"Bakashi", scraper.BakashiType},
	}

	for _, src := range sources {
		t.Run(src.name, func(t *testing.T) {
			results, err := mgr.SearchAnime(query, &src.st)
			if err != nil {
				t.Logf("%s: error: %v", src.name, err)
				return
			}
			fmt.Printf("%s: %d results\n", src.name, len(results))
			for i, r := range results {
				if i >= 3 {
					break
				}
				fmt.Printf("  [%d] %s  url=%s\n", i, r.Name, r.URL)
			}
		})
	}
}
