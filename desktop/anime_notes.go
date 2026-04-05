package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

func notesPath() string {
	if runtime.GOOS == "windows" {
		if d := os.Getenv("LOCALAPPDATA"); d != "" {
			return filepath.Join(d, "GoAnime", "notes.json")
		}
	}
	if h := os.Getenv("HOME"); h != "" {
		return filepath.Join(h, ".local", "goanime", "notes.json")
	}
	return ""
}

func (a *App) loadNotes() {
	a.notesMu.Lock()
	defer a.notesMu.Unlock()

	a.notes = make(map[string]AnimeNote)

	p := notesPath()
	if p == "" {
		return
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return
	}

	var notes map[string]AnimeNote
	if err := json.Unmarshal(data, &notes); err != nil {
		log.Printf("notes: erro ao ler: %v", err)
		return
	}
	a.notes = notes
}

func writeNotes(notes map[string]AnimeNote) error {
	p := notesPath()
	if p == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(notes, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0644)
}

// GetAnimeNote returns the note for a given anime title, or nil if not found.
func (a *App) GetAnimeNote(title string) *AnimeNote {
	a.notesMu.RLock()
	defer a.notesMu.RUnlock()

	note, ok := a.notes[title]
	if !ok {
		return nil
	}
	cp := note
	return &cp
}

// SaveAnimeNote saves or updates a note for an anime.
func (a *App) SaveAnimeNote(note AnimeNote) error {
	if note.Title == "" {
		return fmt.Errorf("titulo e obrigatorio")
	}
	if note.Rating < 0 || note.Rating > 10 {
		return fmt.Errorf("nota deve ser entre 0 e 10")
	}

	note.UpdatedAt = time.Now().Format(time.RFC3339)

	a.notesMu.Lock()
	defer a.notesMu.Unlock()

	a.notes[note.Title] = note
	return writeNotes(a.notes)
}

// GetAllNotes returns all saved notes as a slice.
func (a *App) GetAllNotes() []AnimeNote {
	a.notesMu.RLock()
	defer a.notesMu.RUnlock()

	result := make([]AnimeNote, 0, len(a.notes))
	for _, n := range a.notes {
		result = append(result, n)
	}
	return result
}

// DeleteAnimeNote removes a note for the given anime title.
func (a *App) DeleteAnimeNote(title string) error {
	a.notesMu.Lock()
	defer a.notesMu.Unlock()

	if _, ok := a.notes[title]; !ok {
		return nil
	}
	delete(a.notes, title)
	return writeNotes(a.notes)
}
