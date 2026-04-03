package util

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

const maxSearchHistory = 5

var historyMu sync.Mutex

// SaveSearch salva uma busca bem-sucedida no histórico (máximo de 5 entradas, sem duplicatas).
func SaveSearch(name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}

	historyMu.Lock()
	defer historyMu.Unlock()

	history := loadHistoryUnsafe()

	// Remover duplicata (case-insensitive) e adicionar no início
	filtered := make([]string, 0, len(history))
	for _, h := range history {
		if !strings.EqualFold(h, name) {
			filtered = append(filtered, h)
		}
	}
	history = append([]string{name}, filtered...)

	// Limitar a maxSearchHistory
	if len(history) > maxSearchHistory {
		history = history[:maxSearchHistory]
	}

	_ = saveHistoryUnsafe(history)
}

// LoadSearchHistory retorna as últimas buscas salvas (mais recente primeiro).
func LoadSearchHistory() []string {
	historyMu.Lock()
	defer historyMu.Unlock()
	return loadHistoryUnsafe()
}

func loadHistoryUnsafe() []string {
	path := searchHistoryPath()
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var history []string
	if err := json.Unmarshal(data, &history); err != nil {
		return nil
	}
	return history
}

func saveHistoryUnsafe(history []string) error {
	path := searchHistoryPath()
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// searchHistoryPath retorna o caminho do arquivo JSON de histórico.
func searchHistoryPath() string {
	if runtime.GOOS == "windows" {
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			return ""
		}
		return filepath.Join(localAppData, "GoAnime", "search_history.json")
	}
	home := os.Getenv("HOME")
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".local", "goanime", "search_history.json")
}
