package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

func favoritesPath() string {
	if runtime.GOOS == "windows" {
		if d := os.Getenv("LOCALAPPDATA"); d != "" {
			return filepath.Join(d, "GoAnime", "favorites.json")
		}
	}
	if h := os.Getenv("HOME"); h != "" {
		return filepath.Join(h, ".local", "goanime", "favorites.json")
	}
	return ""
}

func (a *App) GetFavorites() []FavoriteEntry {
	p := favoritesPath()
	if p == "" {
		return []FavoriteEntry{}
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return []FavoriteEntry{}
	}
	var favs []FavoriteEntry
	if err := json.Unmarshal(data, &favs); err != nil {
		return []FavoriteEntry{}
	}
	return favs
}

func (a *App) AddFavorite(entry FavoriteEntry) error {
	favs := a.GetFavorites()
	for _, f := range favs {
		if f.Title == entry.Title {
			return nil
		}
	}
	entry.AddedAt = time.Now().Format(time.RFC3339)
	favs = append(favs, entry)
	return writeFavorites(favs)
}

func (a *App) RemoveFavorite(title string) error {
	favs := a.GetFavorites()
	newFavs := make([]FavoriteEntry, 0, len(favs))
	for _, f := range favs {
		if f.Title != title {
			newFavs = append(newFavs, f)
		}
	}
	return writeFavorites(newFavs)
}

func writeFavorites(favs []FavoriteEntry) error {
	p := favoritesPath()
	if p == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(favs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0644)
}
