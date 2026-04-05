package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

func settingsPath() string {
	if runtime.GOOS == "windows" {
		if d := os.Getenv("LOCALAPPDATA"); d != "" {
			return filepath.Join(d, "GoAnime", "settings.json")
		}
	}
	if h := os.Getenv("HOME"); h != "" {
		return filepath.Join(h, ".local", "goanime", "settings.json")
	}
	return ""
}

func defaultSettings() AppSettings {
	return AppSettings{
		DefaultMode:          "sub",
		DefaultQuality:       "best",
		AutoplayNext:         true,
		NotificationsEnabled: true,
		PlaybackSpeed:        1.0,
	}
}

func (a *App) loadSettings() {
	s := defaultSettings()
	p := settingsPath()
	if p == "" {
		a.settingsMu.Lock()
		a.settings = s
		a.settingsMu.Unlock()
		return
	}
	data, err := os.ReadFile(p)
	if err != nil {
		a.settingsMu.Lock()
		a.settings = s
		a.settingsMu.Unlock()
		return
	}
	if err := json.Unmarshal(data, &s); err != nil {
		log.Printf("settings: parse error: %v", err)
		s = defaultSettings()
	}
	// Ensure sensible defaults for zero-value fields
	if s.DefaultMode == "" {
		s.DefaultMode = "sub"
	}
	if s.DefaultQuality == "" {
		s.DefaultQuality = "best"
	}
	if s.PlaybackSpeed <= 0 {
		s.PlaybackSpeed = 1.0
	}
	a.settingsMu.Lock()
	a.settings = s
	a.settingsMu.Unlock()
}

func (a *App) GetSettings() AppSettings {
	a.settingsMu.RLock()
	defer a.settingsMu.RUnlock()
	return a.settings
}

func (a *App) SaveSettings(s AppSettings) error {
	// Validate
	if s.DefaultMode == "" {
		s.DefaultMode = "sub"
	}
	if s.DefaultQuality == "" {
		s.DefaultQuality = "best"
	}
	if s.PlaybackSpeed <= 0 || s.PlaybackSpeed > 4.0 {
		s.PlaybackSpeed = 1.0
	}

	a.settingsMu.Lock()
	a.settings = s
	a.settingsMu.Unlock()

	return writeSettings(s)
}

func writeSettings(s AppSettings) error {
	p := settingsPath()
	if p == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0644)
}
