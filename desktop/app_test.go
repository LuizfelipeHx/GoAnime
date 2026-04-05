//go:build !production

package main

import (
	"net/url"
	"testing"

	"github.com/alvarorichard/Goanime/internal/models"
	"github.com/alvarorichard/Goanime/internal/scraper"
)

// ---------------------------------------------------------------------------
// parseEpisodeNum
// ---------------------------------------------------------------------------

func TestParseEpisodeNum(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"simple number", "1", 1},
		{"larger number", "25", 25},
		{"zero becomes 1", "0", 1},
		{"negative becomes 1", "-5", 1},
		{"Episode prefix", "Episode 5", 5},
		{"ep prefix", "ep3", 3},
		{"Ep with space", "Ep 12", 12},
		{"decimal episode", "12.5", 125},
		{"empty string returns 1", "", 1},
		{"no digits returns 1", "abc", 1},
		{"mixed text with number", "Episode-3", 3},
		{"whitespace only returns 1", "   ", 1},
		{"leading zeros", "007", 7},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseEpisodeNum(tt.input)
			if got != tt.expected {
				t.Errorf("parseEpisodeNum(%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// progressPercent
// ---------------------------------------------------------------------------

func TestProgressPercent(t *testing.T) {
	tests := []struct {
		name         string
		playbackTime int
		duration     int
		expected     float64
	}{
		{"zero duration", 100, 0, 0},
		{"negative duration", 100, -10, 0},
		{"normal 50%", 50, 100, 50.0},
		{"full 100%", 100, 100, 100.0},
		{"over 100% capped", 200, 100, 100.0},
		{"zero playback", 0, 100, 0.0},
		{"negative playback", -10, 100, 0.0},
		{"small fraction", 1, 1000, 0.1},
		{"both zero", 0, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := progressPercent(tt.playbackTime, tt.duration)
			if got != tt.expected {
				t.Errorf("progressPercent(%d, %d) = %f, want %f", tt.playbackTime, tt.duration, got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// detectContentType
// ---------------------------------------------------------------------------

func TestDetectContentType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"m3u8 URL", "https://example.com/stream.m3u8", "application/vnd.apple.mpegurl"},
		{"m3u8 with params", "https://example.com/stream.m3u8?token=abc", "application/vnd.apple.mpegurl"},
		{"mp4 URL", "https://example.com/video.mp4", "video/mp4"},
		{"mp4 uppercase", "https://example.com/VIDEO.MP4", "video/mp4"},
		{"unknown URL", "https://example.com/stream", "video/*"},
		{"empty URL", "", "video/*"},
		{"m3u8 in path", "https://cdn.example.com/hls/index.m3u8/segment", "application/vnd.apple.mpegurl"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectContentType(tt.input)
			if got != tt.expected {
				t.Errorf("detectContentType(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// normalizeMediaType
// ---------------------------------------------------------------------------

func TestNormalizeMediaType(t *testing.T) {
	tests := []struct {
		name      string
		item      *models.Anime
		expected  string
	}{
		{
			name:     "anime media type",
			item:     &models.Anime{MediaType: models.MediaTypeAnime},
			expected: "anime",
		},
		{
			name:     "movie media type",
			item:     &models.Anime{MediaType: models.MediaTypeMovie},
			expected: "movie",
		},
		{
			name:     "tv media type",
			item:     &models.Anime{MediaType: models.MediaTypeTV},
			expected: "tv",
		},
		{
			name:     "FlixHQ source defaults to movie",
			item:     &models.Anime{Source: "FlixHQ"},
			expected: "movie",
		},
		{
			name:     "empty defaults to anime",
			item:     &models.Anime{},
			expected: "anime",
		},
		{
			name:     "unknown source defaults to anime",
			item:     &models.Anime{Source: "SomeOtherSource"},
			expected: "anime",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeMediaType(tt.item)
			if got != tt.expected {
				t.Errorf("normalizeMediaType(%+v) = %q, want %q", tt.item, got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// parseSource
// ---------------------------------------------------------------------------

func TestParseSource(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantType   *scraper.ScraperType
		wantErrMsg string
	}{
		{"empty returns nil", "", nil, ""},
		{"all returns nil", "all", nil, ""},
		{"allanime", "allanime", ptrScraperType(scraper.AllAnimeType), ""},
		{"animefire", "animefire", ptrScraperType(scraper.AnimefireType), ""},
		{"flixhq", "flixhq", ptrScraperType(scraper.FlixHQType), ""},
		{"animesonlinecc", "animesonlinecc", ptrScraperType(scraper.AnimesOnlineccType), ""},
		{"anroll", "anroll", ptrScraperType(scraper.AnrollType), ""},
		{"bakashi", "bakashi", ptrScraperType(scraper.BakashiType), ""},
		{"invalid source", "invalid", nil, "invalid source"},
		{"uppercase rejected", "AllAnime", nil, "invalid source"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotErr := parseSource(tt.input)
			if tt.wantErrMsg != "" {
				if gotErr == "" {
					t.Errorf("parseSource(%q) error = %q, want containing %q", tt.input, gotErr, tt.wantErrMsg)
				}
				return
			}
			if gotErr != "" {
				t.Errorf("parseSource(%q) unexpected error: %q", tt.input, gotErr)
				return
			}
			if tt.wantType == nil {
				if gotType != nil {
					t.Errorf("parseSource(%q) = %v, want nil", tt.input, *gotType)
				}
			} else {
				if gotType == nil {
					t.Errorf("parseSource(%q) = nil, want %v", tt.input, *tt.wantType)
				} else if *gotType != *tt.wantType {
					t.Errorf("parseSource(%q) = %v, want %v", tt.input, *gotType, *tt.wantType)
				}
			}
		})
	}
}

func ptrScraperType(st scraper.ScraperType) *scraper.ScraperType {
	return &st
}

// ---------------------------------------------------------------------------
// resolveReference
// ---------------------------------------------------------------------------

func TestResolveReference(t *testing.T) {
	base, _ := url.Parse("https://cdn.example.com/hls/master.m3u8")
	tests := []struct {
		name     string
		base     *url.URL
		raw      string
		expected string
	}{
		{"empty string", base, "", ""},
		{"absolute URL unchanged", base, "https://other.com/stream.ts", "https://other.com/stream.ts"},
		{"relative path resolved", base, "segment0.ts", "https://cdn.example.com/hls/segment0.ts"},
		{"relative with directory", base, "../other/seg.ts", "https://cdn.example.com/other/seg.ts"},
		{"protocol-relative", base, "//cdn2.example.com/seg.ts", "https://cdn2.example.com/seg.ts"},
		{"absolute path from root", base, "/root/segment.ts", "https://cdn.example.com/root/segment.ts"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveReference(tt.base, tt.raw)
			if got != tt.expected {
				t.Errorf("resolveReference(%q, %q) = %q, want %q", tt.base.String(), tt.raw, got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// normalizeSource
// ---------------------------------------------------------------------------

func TestNormalizeSource(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"allanime lowercase", "allanime", "AllAnime"},
		{"AllAnime mixed", "AllAnime", "AllAnime"},
		{"animefire", "animefire", "Animefire.io"},
		{"Animefire.io", "Animefire.io", "Animefire.io"},
		{"flixhq", "flixhq", "FlixHQ"},
		{"FlixHQ caps", "FlixHQ", "FlixHQ"},
		{"bakashi", "bakashi", "Bakashi"},
		{"animedrive", "animedrive", "AnimeDrive"},
		{"animesonlinecc", "animesonlinecc", "AnimesOnlineCC"},
		{"anroll", "anroll", "Anroll"},
		{"unknown source passthrough", "MySource", "MySource"},
		{"empty string", "", ""},
		{"with whitespace", "  allanime  ", "AllAnime"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeSource(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeSource(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
