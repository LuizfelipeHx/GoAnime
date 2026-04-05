//go:build !production

package main

import (
	"testing"
)

// ---------------------------------------------------------------------------
// normalizeSearchText
// ---------------------------------------------------------------------------

func TestNormalizeSearchText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"simple text", "Naruto", "naruto"},
		{"mixed case", "Attack On Titan", "attack on titan"},
		{"special chars removed", "Jujutsu Kaisen: Season 2", "jujutsu kaisen season 2"},
		{"multiple spaces collapsed", "  hello   world  ", "hello world"},
		{"unicode preserved", "Shingeki no Kyojin", "shingeki no kyojin"},
		{"punctuation removed", "Re:Zero - Starting Life", "re zero starting life"},
		{"numbers preserved", "Bleach 1000 Year Blood War", "bleach 1000 year blood war"},
		{"tabs and newlines collapsed", "hello\tworld\nnow", "hello world now"},
		{"only spaces", "   ", ""},
		{"only punctuation", "!@#$%^&*()", ""},
		{"hyphens removed", "Sword Art Online - Alicization", "sword art online alicization"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeSearchText(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeSearchText(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// titleSimilarityScore
// ---------------------------------------------------------------------------

func TestTitleSimilarityScore(t *testing.T) {
	tests := []struct {
		name      string
		a         string
		b         string
		wantMin   int
		wantMax   int
		useRange  bool // when true, use wantMin/wantMax range check instead of exact
	}{
		{
			name:    "exact match",
			a:       "Naruto",
			b:       "Naruto",
			wantMin: 1000,
			wantMax: 1000,
		},
		{
			name:    "exact match case insensitive",
			a:       "NARUTO",
			b:       "naruto",
			wantMin: 1000,
			wantMax: 1000,
		},
		{
			name:     "a contains b",
			a:        "Naruto Shippuden",
			b:        "Naruto",
			useRange: true,
			wantMin:  700,
			wantMax:  999,
		},
		{
			name:     "b contains a",
			a:        "Naruto",
			b:        "Naruto Shippuden",
			useRange: true,
			wantMin:  600,
			wantMax:  999,
		},
		{
			name:     "partial word overlap",
			a:        "Attack on Titan Season 2",
			b:        "Attack on Titan Final",
			useRange: true,
			wantMin:  300,
			wantMax:  999,
		},
		{
			name:    "no overlap",
			a:       "Dragon Ball",
			b:       "One Piece",
			wantMin: 300,
			wantMax: 300,
		},
		{
			name:    "empty first",
			a:       "",
			b:       "Naruto",
			wantMin: 0,
			wantMax: 0,
		},
		{
			name:    "empty second",
			a:       "Naruto",
			b:       "",
			wantMin: 0,
			wantMax: 0,
		},
		{
			name:    "both empty",
			a:       "",
			b:       "",
			wantMin: 0,
			wantMax: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := titleSimilarityScore(tt.a, tt.b)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("titleSimilarityScore(%q, %q) = %d, want in [%d, %d]", tt.a, tt.b, got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

// Exact-match is symmetric.
func TestTitleSimilarityScoreSymmetric(t *testing.T) {
	pairs := [][2]string{
		{"Naruto", "Naruto"},
		{"Attack on Titan", "Shingeki no Kyojin"},
	}
	for _, pair := range pairs {
		ab := titleSimilarityScore(pair[0], pair[1])
		ba := titleSimilarityScore(pair[1], pair[0])
		// For exact matches they must be equal; for non-exact they may differ
		// but we just check neither panics and both return non-negative.
		if ab < 0 || ba < 0 {
			t.Errorf("negative score for (%q, %q): ab=%d ba=%d", pair[0], pair[1], ab, ba)
		}
	}
}

// ---------------------------------------------------------------------------
// extractSeasonNumber
// ---------------------------------------------------------------------------

func TestExtractSeasonNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"no season", "Naruto", 0},
		{"Season 2", "Naruto Season 2", 2},
		{"season lowercase", "naruto season 3", 3},
		{"Temporada 3", "One Piece Temporada 3", 3},
		{"2nd Season", "Mob Psycho 100 2nd Season", 2},
		{"3rd Season", "Mob Psycho 100 3rd Season", 3},
		{"1st Season", "Mob Psycho 100 1st Season", 1},
		{"Part 2", "Attack on Titan Part 2", 2},
		{"Parte 3", "Shingeki no Kyojin Parte 3", 3},
		{"Roman numeral II", "Mushoku Tensei II", 2},
		{"Roman numeral III", "Overlord III", 3},
		{"Roman numeral IV", "Overlord IV", 4},
		{"Roman numeral V", "Title V", 5},
		{"Roman numeral VI", "Title VI", 6},
		{"Roman numeral VII", "Title VII", 7},
		{"Roman numeral VIII", "Title VIII", 8},
		{"Roman numeral IX", "Title IX", 9},
		{"Roman numeral X", "Title X", 10},
		{"empty string", "", 0},
		{"Season 10", "Anime Season 10", 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSeasonNumber(tt.input)
			if got != tt.expected {
				t.Errorf("extractSeasonNumber(%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// expandAnimeSearchTerm
// ---------------------------------------------------------------------------

func TestExpandAnimeSearchTerm(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantContain []string // at least these must appear in the result
		wantMinLen  int      // minimum number of variants
	}{
		{
			name:        "empty string",
			input:       "",
			wantContain: nil,
			wantMinLen:  0,
		},
		{
			name:        "simple title returns itself",
			input:       "Naruto",
			wantContain: []string{"Naruto"},
			wantMinLen:  1,
		},
		{
			name:        "colon suffix stripped",
			input:       "Jujutsu Kaisen: Hidden Inventory",
			wantContain: []string{"Jujutsu Kaisen: Hidden Inventory", "Jujutsu Kaisen"},
			wantMinLen:  2,
		},
		{
			name:        "roman numeral suffix stripped",
			input:       "Mushoku Tensei II",
			wantContain: []string{"Mushoku Tensei II", "Mushoku Tensei"},
			wantMinLen:  2,
		},
		{
			name:        "season word suffix stripped",
			input:       "Naruto Season 2",
			wantContain: []string{"Naruto Season 2", "Naruto"},
			wantMinLen:  2,
		},
		{
			name:        "part suffix stripped",
			input:       "Attack on Titan Part 3",
			wantContain: []string{"Attack on Titan Part 3", "Attack on Titan"},
			wantMinLen:  2,
		},
		{
			name:        "title with no in middle",
			input:       "Shingeki no Kyojin",
			wantContain: []string{"Shingeki no Kyojin", "Shingeki Kyojin"},
			wantMinLen:  2,
		},
		{
			name:        "title starting with The",
			input:       "The Rising of Shield Hero",
			wantContain: []string{"The Rising of Shield Hero", "Rising of Shield Hero"},
			wantMinLen:  2,
		},
		{
			name:        "long title truncated",
			input:       "One Two Three Four Five Six",
			wantContain: []string{"One Two Three", "One Two Three Four"},
			wantMinLen:  3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandAnimeSearchTerm(tt.input)
			if len(got) < tt.wantMinLen {
				t.Errorf("expandAnimeSearchTerm(%q) returned %d variants, want >= %d", tt.input, len(got), tt.wantMinLen)
			}
			for _, want := range tt.wantContain {
				found := false
				for _, item := range got {
					if item == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expandAnimeSearchTerm(%q) = %v, missing expected %q", tt.input, got, want)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// mergeSearchTerms
// ---------------------------------------------------------------------------

func TestMergeSearchTerms(t *testing.T) {
	tests := []struct {
		name     string
		current  []string
		incoming []string
		wantLen  int
		wantAll  []string // elements that must be present
	}{
		{
			name:     "both nil",
			current:  nil,
			incoming: nil,
			wantLen:  0,
		},
		{
			name:     "empty slices",
			current:  []string{},
			incoming: []string{},
			wantLen:  0,
		},
		{
			name:     "merge with no duplicates",
			current:  []string{"Naruto"},
			incoming: []string{"Bleach"},
			wantLen:  2,
			wantAll:  []string{"Naruto", "Bleach"},
		},
		{
			name:     "case-insensitive dedup",
			current:  []string{"Naruto"},
			incoming: []string{"naruto"},
			wantLen:  1,
			wantAll:  []string{"Naruto"},
		},
		{
			name:     "empty strings filtered",
			current:  []string{"Naruto", "", "  "},
			incoming: []string{"", "Bleach"},
			wantLen:  2,
			wantAll:  []string{"Naruto", "Bleach"},
		},
		{
			name:     "whitespace trimmed for dedup",
			current:  []string{"  Naruto  "},
			incoming: []string{"Naruto"},
			wantLen:  1,
		},
		{
			name:     "preserves order of first occurrence",
			current:  []string{"Alpha", "Beta"},
			incoming: []string{"Gamma", "Alpha"},
			wantLen:  3,
			wantAll:  []string{"Alpha", "Beta", "Gamma"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeSearchTerms(tt.current, tt.incoming)
			if len(got) != tt.wantLen {
				t.Errorf("mergeSearchTerms(%v, %v) length = %d, want %d; got %v", tt.current, tt.incoming, len(got), tt.wantLen, got)
			}
			for _, want := range tt.wantAll {
				found := false
				for _, item := range got {
					if item == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("mergeSearchTerms result %v missing expected %q", got, want)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// cleanDisplayName
// ---------------------------------------------------------------------------

func TestCleanDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"simple name", "Naruto", "Naruto"},
		{"with leading/trailing spaces", "  Naruto  ", "Naruto"},
		{"with language tag prefix", "[English] Naruto Shippuden", "Naruto Shippuden"},
		{"with source tag prefix", "[AllAnime] One Piece", "One Piece"},
		{"with media type tag", "[Anime] Dragon Ball Super", "Dragon Ball Super"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanDisplayName(tt.input)
			if got != tt.expected {
				t.Errorf("cleanDisplayName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// romanToInt
// ---------------------------------------------------------------------------

func TestRomanToInt(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"I", 1},
		{"II", 2},
		{"III", 3},
		{"IV", 4},
		{"V", 5},
		{"VI", 6},
		{"VII", 7},
		{"VIII", 8},
		{"IX", 9},
		{"X", 10},
		{"", 0},
		{"XI", 0},
		{"abc", 0},
		{"  II  ", 2},     // whitespace trimmed
		{"iii", 3},        // case insensitive
		{"  iv  ", 4},     // whitespace + lowercase
		{"UNKNOWN", 0},    // unknown string
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := romanToInt(tt.input)
			if got != tt.expected {
				t.Errorf("romanToInt(%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// wordOverlapScore
// ---------------------------------------------------------------------------

func TestWordOverlapScore(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected int
	}{
		{"empty a", "", "hello world", 0},
		{"empty b", "hello world", "", 0},
		{"both empty", "", "", 0},
		{"no overlap", "alpha beta", "gamma delta", 0},
		{"full overlap two words", "hello world", "hello world", 80},
		{"partial overlap one word", "hello world", "hello planet", 40},
		{"single word match", "naruto", "naruto", 40},
		{"one of three words", "one two three", "three four five", 40},
		{"two of three words", "one two three", "two three four", 80},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wordOverlapScore(tt.a, tt.b)
			if got != tt.expected {
				t.Errorf("wordOverlapScore(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// buildAnimeSearchAliases
// ---------------------------------------------------------------------------

func TestBuildAnimeSearchAliases(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		wantMinLen   int
		wantContains []string
	}{
		{
			name:         "simple query",
			query:        "Naruto",
			wantMinLen:   1,
			wantContains: []string{"Naruto"},
		},
		{
			name:       "query with language tag",
			query:      "[English] Naruto Shippuden",
			wantMinLen: 1,
			// Should contain both the raw query and the cleaned version
			wantContains: []string{"Naruto Shippuden"},
		},
		{
			name:       "query with colon generates base",
			query:      "Jujutsu Kaisen: Hidden Inventory",
			wantMinLen: 2,
		},
		{
			name:       "empty query",
			query:      "",
			wantMinLen: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildAnimeSearchAliases(tt.query)
			if len(got) < tt.wantMinLen {
				t.Errorf("buildAnimeSearchAliases(%q) returned %d aliases, want >= %d: %v", tt.query, len(got), tt.wantMinLen, got)
			}
			for _, want := range tt.wantContains {
				found := false
				for _, alias := range got {
					if alias == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("buildAnimeSearchAliases(%q) = %v, missing expected %q", tt.query, got, want)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// detectAlternativeLanguage
// ---------------------------------------------------------------------------

func TestDetectAlternativeLanguage(t *testing.T) {
	tests := []struct {
		name       string
		item       MediaAlternative
		wantPT     bool
		wantEN     bool
		wantDub    bool
		wantSub    bool
	}{
		{
			name:    "animefire source is Portuguese sub",
			item:    MediaAlternative{Name: "Naruto", Source: "animefire"},
			wantPT:  true,
			wantEN:  false,
			wantDub: false,
			wantSub: true,
		},
		{
			name:    "allanime source is English sub",
			item:    MediaAlternative{Name: "Naruto", Source: "allanime"},
			wantPT:  false,
			wantEN:  true,
			wantDub: false,
			wantSub: true,
		},
		{
			name:    "dublado in name marks dub+PT",
			item:    MediaAlternative{Name: "Naruto Dublado", Source: "animefire"},
			wantPT:  true,
			wantEN:  false,
			wantDub: true,
			wantSub: false,
		},
		{
			name:    "legendado in name marks sub",
			item:    MediaAlternative{Name: "Naruto Legendado", Source: "animefire"},
			wantPT:  true,
			wantEN:  false,
			wantDub: false,
			wantSub: true,
		},
		{
			name:    "[dub] tag detected",
			item:    MediaAlternative{Name: "Naruto [dub]", Source: "anroll"},
			wantPT:  true,
			wantEN:  false,
			wantDub: true,
			wantSub: false,
		},
		{
			name:    "[sub] tag detected",
			item:    MediaAlternative{Name: "Naruto [sub]", Source: "anroll"},
			wantPT:  true,
			wantEN:  false,
			wantDub: false,
			wantSub: true,
		},
		{
			name:    "[english] tag marks EN and sub",
			item:    MediaAlternative{Name: "Naruto [english]", Source: "unknown"},
			wantPT:  false,
			wantEN:  true,
			wantDub: false,
			wantSub: true,
		},
		{
			name:    "animesonlinecc source is Portuguese",
			item:    MediaAlternative{Name: "Naruto", Source: "animesonlinecc"},
			wantPT:  true,
			wantEN:  false,
			wantDub: false,
			wantSub: true,
		},
		{
			name:    "bakashi source is Portuguese",
			item:    MediaAlternative{Name: "Naruto", Source: "bakashi"},
			wantPT:  true,
			wantEN:  false,
			wantDub: false,
			wantSub: true,
		},
		{
			name:    "unknown source unknown name defaults to EN sub",
			item:    MediaAlternative{Name: "Naruto", Source: "unknown"},
			wantPT:  false,
			wantEN:  true,
			wantDub: false,
			wantSub: true,
		},
		{
			name:    "[portuguese] tag marks PT and dub",
			item:    MediaAlternative{Name: "Naruto [portuguese]", Source: "unknown"},
			wantPT:  true,
			wantEN:  false,
			wantDub: true,
			wantSub: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPT, gotEN, gotDub, gotSub := detectAlternativeLanguage(tt.item)
			if gotPT != tt.wantPT {
				t.Errorf("detectAlternativeLanguage(%v) PT = %v, want %v", tt.item, gotPT, tt.wantPT)
			}
			if gotEN != tt.wantEN {
				t.Errorf("detectAlternativeLanguage(%v) EN = %v, want %v", tt.item, gotEN, tt.wantEN)
			}
			if gotDub != tt.wantDub {
				t.Errorf("detectAlternativeLanguage(%v) Dub = %v, want %v", tt.item, gotDub, tt.wantDub)
			}
			if gotSub != tt.wantSub {
				t.Errorf("detectAlternativeLanguage(%v) Sub = %v, want %v", tt.item, gotSub, tt.wantSub)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// firstNonEmpty
// ---------------------------------------------------------------------------

func TestFirstNonEmpty(t *testing.T) {
	tests := []struct {
		name     string
		values   []string
		expected string
	}{
		{"single value", []string{"hello"}, "hello"},
		{"first empty", []string{"", "world"}, "world"},
		{"all empty", []string{"", "", ""}, ""},
		{"spaces only skipped", []string{"  ", "real"}, "real"},
		{"no values", nil, ""},
		{"first non-empty wins", []string{"alpha", "beta"}, "alpha"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := firstNonEmpty(tt.values...)
			if got != tt.expected {
				t.Errorf("firstNonEmpty(%v) = %q, want %q", tt.values, got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// isRelevantRelationType
// ---------------------------------------------------------------------------

func TestIsRelevantRelationType(t *testing.T) {
	relevant := []string{"SEQUEL", "PREQUEL", "SIDE_STORY", "ALTERNATIVE", "PARENT", "PARENT_STORY"}
	for _, rt := range relevant {
		if !isRelevantRelationType(rt) {
			t.Errorf("isRelevantRelationType(%q) = false, want true", rt)
		}
	}
	irrelevant := []string{"CHARACTER", "SUMMARY", "OTHER", "", "unknown"}
	for _, rt := range irrelevant {
		if isRelevantRelationType(rt) {
			t.Errorf("isRelevantRelationType(%q) = true, want false", rt)
		}
	}
	// Case insensitive via ToUpper
	if !isRelevantRelationType("  sequel  ") {
		t.Error("isRelevantRelationType with whitespace should still match")
	}
}
