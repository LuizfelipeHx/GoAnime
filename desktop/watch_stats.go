package main

import (
	"sort"
	"time"
)

const estimatedMinutesPerEpisode = 24

// GetWatchStats computes dashboard statistics from existing watch progress
// and custom lists data.
func (a *App) GetWatchStats() WatchStats {
	stats := WatchStats{
		TopGenres:      []string{},
		RecentActivity: []ActivityDay{},
	}

	progress := a.GetWatchProgress()
	if len(progress) == 0 {
		// Fill recent activity with zeros
		stats.RecentActivity = emptyRecentActivity(14)
		return stats
	}

	// ── Basic counts ──
	animeSet := make(map[string]bool)
	for _, p := range progress {
		animeSet[p.Title] = true
		stats.TotalEpisodes += p.EpisodeNumber
	}
	stats.TotalAnime = len(animeSet)
	stats.TotalMinutes = stats.TotalEpisodes * estimatedMinutesPerEpisode

	// ── Completed anime: progress >= 90% on the last tracked episode ──
	for _, p := range progress {
		if p.ProgressPercent >= 90.0 {
			stats.CompletedAnime++
		}
	}

	// ── Top genres from custom lists ──
	stats.TopGenres = a.computeTopGenres()

	// ── Recent activity & streaks ──
	dayMap := buildDayMap(progress)
	stats.RecentActivity = buildRecentActivity(dayMap, 14)
	stats.CurrentStreak, stats.LongestStreak = computeStreaks(dayMap)

	return stats
}

// computeTopGenres gathers genres from the catalog items that match titles
// in custom lists. Returns up to 5 genres sorted by frequency.
func (a *App) computeTopGenres() []string {
	a.customListsMu.RLock()
	titles := make(map[string]bool)
	for _, entries := range a.customLists.Lists {
		for _, e := range entries {
			titles[e.Name] = true
		}
	}
	a.customListsMu.RUnlock()

	if len(titles) == 0 {
		return []string{}
	}

	// We don't have genre info stored per list entry, so return empty
	// unless we can derive from favorites or progress data in the future.
	return []string{}
}

// buildDayMap groups watch progress entries by date (YYYY-MM-DD).
func buildDayMap(progress []WatchProgressEntry) map[string]int {
	dayMap := make(map[string]int)
	for _, p := range progress {
		t, err := time.Parse(time.RFC3339, p.LastUpdated)
		if err != nil {
			continue
		}
		day := t.Format("2006-01-02")
		dayMap[day] += p.EpisodeNumber
	}
	return dayMap
}

// buildRecentActivity returns the last N days of activity.
func buildRecentActivity(dayMap map[string]int, days int) []ActivityDay {
	now := time.Now()
	activity := make([]ActivityDay, days)
	for i := 0; i < days; i++ {
		d := now.AddDate(0, 0, -(days-1-i))
		dateStr := d.Format("2006-01-02")
		eps := dayMap[dateStr]
		activity[i] = ActivityDay{
			Date:     dateStr,
			Episodes: eps,
			Minutes:  eps * estimatedMinutesPerEpisode,
		}
	}
	return activity
}

// emptyRecentActivity returns N days of zero activity.
func emptyRecentActivity(days int) []ActivityDay {
	return buildRecentActivity(nil, days)
}

// computeStreaks calculates the current and longest consecutive-day streak.
func computeStreaks(dayMap map[string]int) (current int, longest int) {
	if len(dayMap) == 0 {
		return 0, 0
	}

	// Collect all active dates
	dates := make([]string, 0, len(dayMap))
	for d, eps := range dayMap {
		if eps > 0 {
			dates = append(dates, d)
		}
	}
	if len(dates) == 0 {
		return 0, 0
	}

	sort.Strings(dates)

	// Parse sorted dates
	parsed := make([]time.Time, 0, len(dates))
	for _, d := range dates {
		t, err := time.Parse("2006-01-02", d)
		if err != nil {
			continue
		}
		parsed = append(parsed, t)
	}
	if len(parsed) == 0 {
		return 0, 0
	}

	// Calculate streaks
	streak := 1
	longest = 1
	for i := 1; i < len(parsed); i++ {
		diff := parsed[i].Sub(parsed[i-1]).Hours() / 24
		if diff == 1 {
			streak++
			if streak > longest {
				longest = streak
			}
		} else {
			streak = 1
		}
	}

	// Check if current streak includes today or yesterday
	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	lastDate := parsed[len(parsed)-1].Format("2006-01-02")

	if lastDate == today || lastDate == yesterday {
		current = streak
	} else {
		current = 0
	}

	return current, longest
}
