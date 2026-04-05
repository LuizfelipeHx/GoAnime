package main

import "time"

// NyaaRelease represents a torrent release from Nyaa.
type NyaaRelease struct {
	Title    string    `json:"title"`
	Link     string    `json:"link"`
	InfoHash string    `json:"infoHash,omitempty"`
	Size     string    `json:"size"`
	Date     time.Time `json:"date"`
	Seeders  int       `json:"seeders"`
	Category string    `json:"category,omitempty"`
	IsNew    bool      `json:"isNew"`
}

// AIRecommendation is a single anime recommendation from the AI.
type AIRecommendation struct {
	Title  string `json:"title"`
	Reason string `json:"reason"`
	Genres string `json:"genres,omitempty"`
	Score  string `json:"score,omitempty"`
}

// CuratedRelease is a Nyaa release with AI quality assessment.
type CuratedRelease struct {
	Release NyaaRelease `json:"release"`
	Quality string      `json:"quality"` // "Excelente", "Bom", "Regular"
	Summary string      `json:"summary"`
}

// BotStatus represents the current state of all bots.
type BotStatus struct {
	AIOnline      bool   `json:"aiOnline"`
	AIModel       string `json:"aiModel,omitempty"`
	ReleasesCount int    `json:"releasesCount"`
	NewReleases   int    `json:"newReleases"`
	LastCheck     string `json:"lastCheck,omitempty"`
	RecsAvailable bool   `json:"recsAvailable"`
	RecsCount     int    `json:"recsCount"`
	CuratedCount  int    `json:"curatedCount"`
}
