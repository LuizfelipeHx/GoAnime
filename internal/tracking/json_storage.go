// json_storage.go - JSON-based storage backend for when SQLite/CGO is unavailable.
package tracking

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// jsonStorage is a simple JSON-backed store for anime progress.
// It is used as a fallback when CGO/SQLite is not available.
type jsonStorage struct {
	mu   sync.RWMutex
	data map[string]Anime // keyed by AllanimeID
	path string
}

// newJsonStorage creates a jsonStorage backed by the given path.
// The path is derived from the SQLite DB path but with a .json extension.
func newJsonStorage(dbPath string) *jsonStorage {
	ext := filepath.Ext(dbPath)
	jsonPath := strings.TrimSuffix(dbPath, ext) + ".json"

	s := &jsonStorage{
		data: make(map[string]Anime),
		path: jsonPath,
	}
	_ = s.load() // ignore error if file doesn't exist yet
	return s
}

func (s *jsonStorage) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	raw, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(raw, &s.data)
}

func (s *jsonStorage) save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0600)
}

func (s *jsonStorage) update(a Anime) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if a.Duration <= 0 {
		return fmt.Errorf("invalid duration value (%d): must be greater than 0", a.Duration)
	}
	if a.PlaybackTime < 0 {
		a.PlaybackTime = 0
	}
	if a.MediaType == "" {
		if strings.Contains(a.Title, "[Movies/TV]") || strings.Contains(a.Title, "[Movie]") {
			a.MediaType = "movie"
		} else {
			a.MediaType = "anime"
		}
	}

	// Preserve AnilistID if it was previously set
	if existing, ok := s.data[a.AllanimeID]; ok {
		if existing.AnilistID > 0 && a.AnilistID == 0 {
			a.AnilistID = existing.AnilistID
		}
	}

	s.data[a.AllanimeID] = a
	return s.save()
}

func (s *jsonStorage) get(allanimeID string) (*Anime, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	a, ok := s.data[allanimeID]
	if !ok {
		return nil, nil
	}
	copy := a
	return &copy, nil
}

func (s *jsonStorage) getAll() ([]Anime, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Anime, 0, len(s.data))
	for _, a := range s.data {
		result = append(result, a)
	}
	return result, nil
}

func (s *jsonStorage) remove(allanimeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, allanimeID)
	return s.save()
}
