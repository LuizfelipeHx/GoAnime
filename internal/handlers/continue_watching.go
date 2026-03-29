package handlers

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"

	"github.com/alvarorichard/Goanime/internal/tracking"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

const (
	searchSentinel   = "__search__"
	maxContinueItems = 5
)

var (
	cwTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6366F1")).
			Bold(true).
			MarginLeft(2)

	cwSubStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A9A9A9")).
			Italic(true).
			MarginLeft(2)
)

// Regexps compilados uma vez para limpar sufixos de episódio do título
var episodeSuffixRe = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\s*-\s*episode\s*\d+`),
	regexp.MustCompile(`(?i)\s*-\s*ep\s*\d+`),
	regexp.MustCompile(`(?i)\s*ep\.?\s*\d+$`),
	regexp.MustCompile(`\s+\d+$`),
}

// ShowContinueWatching exibe a tela de "Continue Watching" se houver histórico.
// Retorna (nome, true) se o usuário escolheu um anime, ou ("", false) para ir à busca normal.
func ShowContinueWatching(animeName string) (string, bool) {
	tracker := getOrInitTracker()
	if tracker == nil {
		return "", false
	}

	recent := recentWatched(tracker)
	if len(recent) == 0 {
		return "", false
	}

	fmt.Println(cwTitleStyle.Render("▶  Continue Watching"))
	fmt.Println(cwSubStyle.Render("Continue de onde parou, ou busque algo novo."))
	fmt.Println()

	var choice string
	menu := huh.NewSelect[string]().
		Title("Recentes").
		Options(buildMenuOptions(recent)...).
		Value(&choice)

	if err := menu.Run(); err != nil {
		// ESC ou Ctrl+C → vai para o fluxo normal de busca
		return "", false
	}

	if choice == searchSentinel {
		return "", false
	}

	return choice, true
}

func getOrInitTracker() *tracking.LocalTracker {
	if t := tracking.GetGlobalTracker(); t != nil {
		return t
	}
	dbPath := resolveTrackerDBPath()
	if dbPath == "" {
		return nil
	}
	return tracking.NewLocalTracker(dbPath)
}

func resolveTrackerDBPath() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "GoAnime", "tracking", "progress.db")
	}
	u, err := user.Current()
	if err != nil {
		return ""
	}
	return filepath.Join(u.HomeDir, ".local", "goanime", "tracking", "progress.db")
}

func recentWatched(tracker *tracking.LocalTracker) []tracking.Anime {
	all, err := tracker.GetAllAnime()
	if err != nil || len(all) == 0 {
		return nil
	}

	// Deduplica por nome limpo, mantendo a entrada mais recente de cada anime
	best := make(map[string]tracking.Anime)
	for _, a := range all {
		name := extractAnimeName(a.Title)
		if existing, ok := best[name]; !ok || a.LastUpdated.After(existing.LastUpdated) {
			best[name] = a
		}
	}

	deduped := make([]tracking.Anime, 0, len(best))
	for _, a := range best {
		deduped = append(deduped, a)
	}

	// Ordena do mais recente para o mais antigo
	sort.Slice(deduped, func(i, j int) bool {
		return deduped[i].LastUpdated.After(deduped[j].LastUpdated)
	})

	if len(deduped) > maxContinueItems {
		deduped = deduped[:maxContinueItems]
	}
	return deduped
}

func extractAnimeName(title string) string {
	result := title
	for _, re := range episodeSuffixRe {
		result = strings.TrimSpace(re.ReplaceAllString(result, ""))
	}
	if result == "" {
		return title
	}
	return result
}

func buildMenuOptions(recent []tracking.Anime) []huh.Option[string] {
	opts := make([]huh.Option[string], 0, len(recent)+1)
	for _, a := range recent {
		name := extractAnimeName(a.Title)
		date := a.LastUpdated.Format("02/01")
		label := fmt.Sprintf("%-38s EP %-3d · %s", name, a.EpisodeNumber, date)
		opts = append(opts, huh.NewOption(label, name))
	}
	opts = append(opts, huh.NewOption("🔍  Buscar novo anime", searchSentinel))
	return opts
}
