package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
)

func watchedPath() string {
	if runtime.GOOS == "windows" {
		if d := os.Getenv("LOCALAPPDATA"); d != "" {
			return filepath.Join(d, "GoAnime", "watched.json")
		}
	}
	if h := os.Getenv("HOME"); h != "" {
		return filepath.Join(h, ".local", "goanime", "watched.json")
	}
	return ""
}

func (a *App) loadWatchedMarks() {
	marks := make(map[string][]int)
	p := watchedPath()
	if p != "" {
		data, err := os.ReadFile(p)
		if err == nil {
			if err := json.Unmarshal(data, &marks); err != nil {
				log.Printf("watched: parse error: %v", err)
				marks = make(map[string][]int)
			}
		}
	}
	a.watchedMu.Lock()
	a.watchedMarks = marks
	a.watchedMu.Unlock()
}

func (a *App) GetWatchedEpisodes(groupKey string) []int {
	a.watchedMu.RLock()
	defer a.watchedMu.RUnlock()
	eps := a.watchedMarks[groupKey]
	if eps == nil {
		return []int{}
	}
	out := make([]int, len(eps))
	copy(out, eps)
	return out
}

func (a *App) SetEpisodeWatched(groupKey string, episodeNum int, watched bool) error {
	a.watchedMu.Lock()
	defer a.watchedMu.Unlock()

	eps := a.watchedMarks[groupKey]
	if watched {
		// Add if not already present
		for _, e := range eps {
			if e == episodeNum {
				return nil
			}
		}
		eps = append(eps, episodeNum)
		sort.Ints(eps)
	} else {
		// Remove
		filtered := make([]int, 0, len(eps))
		for _, e := range eps {
			if e != episodeNum {
				filtered = append(filtered, e)
			}
		}
		eps = filtered
	}
	a.watchedMarks[groupKey] = eps
	return writeWatchedMarks(a.watchedMarks)
}

func (a *App) SetAllWatchedUpTo(groupKey string, upToEpisode int) error {
	a.watchedMu.Lock()
	defer a.watchedMu.Unlock()

	eps := make([]int, 0, upToEpisode)
	for i := 1; i <= upToEpisode; i++ {
		eps = append(eps, i)
	}
	a.watchedMarks[groupKey] = eps
	return writeWatchedMarks(a.watchedMarks)
}

func writeWatchedMarks(marks map[string][]int) error {
	p := watchedPath()
	if p == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(marks, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0644)
}
