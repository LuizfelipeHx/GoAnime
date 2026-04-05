package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

var defaultListNames = []string{"Favoritos", "Para assistir", "Concluído", "Abandonado"}

func customListsPath() string {
	if runtime.GOOS == "windows" {
		if d := os.Getenv("LOCALAPPDATA"); d != "" {
			return filepath.Join(d, "GoAnime", "custom_lists.json")
		}
	}
	if h := os.Getenv("HOME"); h != "" {
		return filepath.Join(h, ".local", "goanime", "custom_lists.json")
	}
	return ""
}

type customListsData struct {
	Order []string               `json:"order"`
	Lists map[string][]ListEntry `json:"lists"`
}

func (a *App) loadCustomLists() {
	a.customListsMu.Lock()
	defer a.customListsMu.Unlock()

	a.customLists = customListsData{
		Order: append([]string{}, defaultListNames...),
		Lists: make(map[string][]ListEntry),
	}
	for _, name := range defaultListNames {
		a.customLists.Lists[name] = []ListEntry{}
	}

	p := customListsPath()
	if p == "" {
		a.migrateFromFavorites()
		return
	}

	data, err := os.ReadFile(p)
	if err != nil {
		// File doesn't exist — try migrating from favorites
		a.migrateFromFavorites()
		return
	}

	var loaded customListsData
	if err := json.Unmarshal(data, &loaded); err != nil {
		log.Printf("custom_lists: erro ao ler: %v", err)
		a.migrateFromFavorites()
		return
	}

	// Ensure all default lists exist
	if loaded.Lists == nil {
		loaded.Lists = make(map[string][]ListEntry)
	}
	if len(loaded.Order) == 0 {
		loaded.Order = append([]string{}, defaultListNames...)
	}
	for _, name := range defaultListNames {
		if _, ok := loaded.Lists[name]; !ok {
			loaded.Lists[name] = []ListEntry{}
			// Add to order if missing
			found := false
			for _, o := range loaded.Order {
				if o == name {
					found = true
					break
				}
			}
			if !found {
				loaded.Order = append(loaded.Order, name)
			}
		}
	}

	a.customLists = loaded
}

// migrateFromFavorites imports existing favorites into the "Favoritos" list.
// Must be called with customListsMu already held.
func (a *App) migrateFromFavorites() {
	fp := favoritesPath()
	if fp == "" {
		return
	}
	data, err := os.ReadFile(fp)
	if err != nil {
		return
	}
	var favs []FavoriteEntry
	if err := json.Unmarshal(data, &favs); err != nil {
		log.Printf("custom_lists: erro ao migrar favoritos: %v", err)
		return
	}
	if len(favs) == 0 {
		return
	}

	entries := make([]ListEntry, 0, len(favs))
	for _, f := range favs {
		entries = append(entries, ListEntry{
			Name:     f.Title,
			URL:      f.URL,
			ImageURL: f.ImageURL,
			Source:   f.Source,
			ListName: "Favoritos",
		})
	}
	a.customLists.Lists["Favoritos"] = entries

	// Persist the migrated data
	_ = writeCustomLists(a.customLists)
	log.Printf("custom_lists: migrados %d favoritos", len(entries))
}

func writeCustomLists(data customListsData) error {
	p := customListsPath()
	if p == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, raw, 0644)
}

// GetCustomLists returns all lists keyed by list name.
func (a *App) GetCustomLists() map[string][]ListEntry {
	a.customListsMu.RLock()
	defer a.customListsMu.RUnlock()

	result := make(map[string][]ListEntry, len(a.customLists.Lists))
	for k, v := range a.customLists.Lists {
		cp := make([]ListEntry, len(v))
		copy(cp, v)
		result[k] = cp
	}
	return result
}

// GetListNames returns list names in their defined order.
func (a *App) GetListNames() []string {
	a.customListsMu.RLock()
	defer a.customListsMu.RUnlock()

	names := make([]string, len(a.customLists.Order))
	copy(names, a.customLists.Order)
	return names
}

// AddToList adds an entry to a named list, deduplicating by name.
func (a *App) AddToList(listName string, entry ListEntry) error {
	a.customListsMu.Lock()
	defer a.customListsMu.Unlock()

	entries, ok := a.customLists.Lists[listName]
	if !ok {
		return fmt.Errorf("lista %q nao encontrada", listName)
	}
	for _, e := range entries {
		if e.Name == entry.Name {
			return nil // already exists
		}
	}
	entry.ListName = listName
	a.customLists.Lists[listName] = append(entries, entry)
	return writeCustomLists(a.customLists)
}

// RemoveFromList removes an entry by anime name from a named list.
func (a *App) RemoveFromList(listName string, name string) error {
	a.customListsMu.Lock()
	defer a.customListsMu.Unlock()

	entries, ok := a.customLists.Lists[listName]
	if !ok {
		return fmt.Errorf("lista %q nao encontrada", listName)
	}
	filtered := make([]ListEntry, 0, len(entries))
	for _, e := range entries {
		if e.Name != name {
			filtered = append(filtered, e)
		}
	}
	a.customLists.Lists[listName] = filtered
	return writeCustomLists(a.customLists)
}

// MoveToList moves an entry from one list to another.
func (a *App) MoveToList(fromList string, toList string, name string) error {
	a.customListsMu.Lock()
	defer a.customListsMu.Unlock()

	srcEntries, ok := a.customLists.Lists[fromList]
	if !ok {
		return fmt.Errorf("lista origem %q nao encontrada", fromList)
	}
	if _, ok := a.customLists.Lists[toList]; !ok {
		return fmt.Errorf("lista destino %q nao encontrada", toList)
	}

	var moved *ListEntry
	filtered := make([]ListEntry, 0, len(srcEntries))
	for i := range srcEntries {
		if srcEntries[i].Name == name {
			cp := srcEntries[i]
			moved = &cp
		} else {
			filtered = append(filtered, srcEntries[i])
		}
	}
	if moved == nil {
		return fmt.Errorf("anime %q nao encontrado na lista %q", name, fromList)
	}

	// Check for duplicates in the destination
	for _, e := range a.customLists.Lists[toList] {
		if e.Name == name {
			// Already in destination — just remove from source
			a.customLists.Lists[fromList] = filtered
			return writeCustomLists(a.customLists)
		}
	}

	moved.ListName = toList
	a.customLists.Lists[fromList] = filtered
	a.customLists.Lists[toList] = append(a.customLists.Lists[toList], *moved)
	return writeCustomLists(a.customLists)
}

