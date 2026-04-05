package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type sourceFailureRecord struct {
	GroupKey  string `json:"groupKey"`
	Action    string `json:"action"`
	Source    string `json:"source"`
	Count     int    `json:"count"`
	LastError string `json:"lastError,omitempty"`
	UpdatedAt string `json:"updatedAt"`
}

func sourceFailuresPath() string {
	if runtime.GOOS == "windows" {
		if d := os.Getenv("LOCALAPPDATA"); d != "" {
			return filepath.Join(d, "GoAnime", "source_failures.json")
		}
	}
	if h := os.Getenv("HOME"); h != "" {
		return filepath.Join(h, ".local", "goanime", "source_failures.json")
	}
	return ""
}

func readSourceFailures() []sourceFailureRecord {
	p := sourceFailuresPath()
	if p == "" {
		return nil
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	var items []sourceFailureRecord
	if err := json.Unmarshal(data, &items); err != nil {
		return nil
	}
	return items
}

func writeSourceFailures(items []sourceFailureRecord) error {
	p := sourceFailuresPath()
	if p == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

func normalizeFailureKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func sourceFailureCount(groupKey string, action string, source string) int {
	groupKey = normalizeFailureKey(groupKey)
	action = normalizeFailureKey(action)
	source = normalizeFailureKey(source)
	if groupKey == "" || action == "" || source == "" {
		return 0
	}
	for _, item := range readSourceFailures() {
		if normalizeFailureKey(item.GroupKey) == groupKey && normalizeFailureKey(item.Action) == action && normalizeFailureKey(item.Source) == source {
			return item.Count
		}
	}
	return 0
}

func recordSourceFailure(groupKey string, action string, source string, err error) {
	groupKey = normalizeFailureKey(groupKey)
	action = normalizeFailureKey(action)
	source = normalizeFailureKey(source)
	if groupKey == "" || action == "" || source == "" {
		return
	}

	items := readSourceFailures()
	message := ""
	if err != nil {
		message = strings.TrimSpace(err.Error())
	}
	now := time.Now().Format(time.RFC3339)
	for i := range items {
		if normalizeFailureKey(items[i].GroupKey) == groupKey && normalizeFailureKey(items[i].Action) == action && normalizeFailureKey(items[i].Source) == source {
			items[i].Count++
			items[i].LastError = message
			items[i].UpdatedAt = now
			if err := writeSourceFailures(items); err != nil {
				log.Printf("failed to write source failures: %v", err)
			}
			return
		}
	}

	items = append(items, sourceFailureRecord{
		GroupKey:  groupKey,
		Action:    action,
		Source:    source,
		Count:     1,
		LastError: message,
		UpdatedAt: now,
	})
	if err := writeSourceFailures(items); err != nil {
		log.Printf("failed to write source failures: %v", err)
	}
}

func clearSourceFailure(groupKey string, action string, source string) {
	groupKey = normalizeFailureKey(groupKey)
	action = normalizeFailureKey(action)
	source = normalizeFailureKey(source)
	if groupKey == "" || action == "" || source == "" {
		return
	}

	items := readSourceFailures()
	if len(items) == 0 {
		return
	}

	filtered := make([]sourceFailureRecord, 0, len(items))
	changed := false
	for _, item := range items {
		if normalizeFailureKey(item.GroupKey) == groupKey && normalizeFailureKey(item.Action) == action && normalizeFailureKey(item.Source) == source {
			changed = true
			continue
		}
		filtered = append(filtered, item)
	}
	if changed {
		if err := writeSourceFailures(filtered); err != nil {
			log.Printf("failed to write source failures: %v", err)
		}
	}
}
