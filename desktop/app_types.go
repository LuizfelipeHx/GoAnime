package main

type MediaAlternative struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	Source    string `json:"source"`
	MediaType string `json:"mediaType"`
}

type MediaRequest struct {
	Name         string             `json:"name"`
	URL          string             `json:"url"`
	Source       string             `json:"source"`
	MediaType    string             `json:"mediaType"`
	GroupKey     string             `json:"groupKey,omitempty"`
	Alternatives []MediaAlternative `json:"alternatives,omitempty"`
}

type MediaResult struct {
	Name               string             `json:"name"`
	URL                string             `json:"url"`
	ImageURL           string             `json:"imageUrl"`
	Source             string             `json:"source"`
	MediaType          string             `json:"mediaType"`
	Year               string             `json:"year"`
	Score              float64            `json:"score,omitempty"`
	Description        string             `json:"description,omitempty"`
	Genres             []string           `json:"genres,omitempty"`
	CanonicalTitle     string             `json:"canonicalTitle,omitempty"`
	GroupKey           string             `json:"groupKey,omitempty"`
	SeasonNumber       int                `json:"seasonNumber,omitempty"`
	AvailableSources   []string           `json:"availableSources,omitempty"`
	WatchSource        string             `json:"watchSource,omitempty"`
	DownloadSource     string             `json:"downloadSource,omitempty"`
	DubSource          string             `json:"dubSource,omitempty"`
	SubSource          string             `json:"subSource,omitempty"`
	Alternatives       []MediaAlternative `json:"alternatives,omitempty"`
	HasPortuguese      bool               `json:"hasPortuguese,omitempty"`
	HasEnglish         bool               `json:"hasEnglish,omitempty"`
	HasDub             bool               `json:"hasDub,omitempty"`
	HasSub             bool               `json:"hasSub,omitempty"`
	WatchHasPortuguese bool               `json:"watchHasPortuguese,omitempty"`
	WatchHasEnglish    bool               `json:"watchHasEnglish,omitempty"`
	WatchHasDub        bool               `json:"watchHasDub,omitempty"`
	WatchHasSub        bool               `json:"watchHasSub,omitempty"`
}

type EpisodeResult struct {
	Number string `json:"number"`
	Num    int    `json:"num"`
	Title  string `json:"title"`
	URL    string `json:"url"`
}

type EpisodesResponse struct {
	Name           string          `json:"name"`
	Source         string          `json:"source"`
	MediaType      string          `json:"mediaType"`
	Count          int             `json:"count"`
	Episodes       []EpisodeResult `json:"episodes"`
	ResolvedSource string          `json:"resolvedSource,omitempty"`
	ResolvedURL    string          `json:"resolvedUrl,omitempty"`
	Note           string          `json:"note,omitempty"`
}

type SubtitleResult struct {
	URL      string `json:"url"`
	ProxyURL string `json:"proxyUrl"`
	Language string `json:"language"`
	Label    string `json:"label"`
}

type StreamRequest struct {
	Media         MediaRequest `json:"media"`
	EpisodeURL    string       `json:"episodeUrl"`
	EpisodeNumber string       `json:"episodeNumber"`
	Mode          string       `json:"mode"`
	Quality       string       `json:"quality"`
}

type StreamResponse struct {
	StreamURL          string           `json:"streamUrl"`
	ProxyURL           string           `json:"proxyUrl"`
	ContentType        string           `json:"contentType"`
	Subtitles          []SubtitleResult `json:"subtitles,omitempty"`
	Note               string           `json:"note,omitempty"`
	ResolvedSource     string           `json:"resolvedSource,omitempty"`
	ResolvedURL        string           `json:"resolvedUrl,omitempty"`
	ResolvedEpisodeURL string           `json:"resolvedEpisodeUrl,omitempty"`
}

type HistoryEntry struct {
	Name string `json:"name"`
}

type WatchProgressEntry struct {
	AllanimeID        string  `json:"allanimeId"`
	Title             string  `json:"title"`
	EpisodeNumber     int     `json:"episodeNumber"`
	PlaybackTime      int     `json:"playbackTime"`
	Duration          int     `json:"duration"`
	ProgressPercent   float64 `json:"progressPercent"`
	TotalEpisodes     int     `json:"totalEpisodes"`
	RemainingEpisodes int     `json:"remainingEpisodes"`
	MediaType         string  `json:"mediaType"`
	LastUpdated       string  `json:"lastUpdated"`
}

type CatalogItem struct {
	ID          int      `json:"id"`
	Title       string   `json:"title"`
	CoverImage  string   `json:"coverImage"`
	BannerImage string   `json:"bannerImage"`
	Score       float64  `json:"score"`
	Genres      []string `json:"genres"`
	Episodes    int      `json:"episodes"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
}

type CatalogSection struct {
	Label string        `json:"label"`
	Items []CatalogItem `json:"items"`
}

type FavoriteEntry struct {
	Title     string `json:"title"`
	ImageURL  string `json:"imageUrl"`
	URL       string `json:"url"`
	Source    string `json:"source"`
	MediaType string `json:"mediaType"`
	AddedAt   string `json:"addedAt"`
}

type UpdateWatchProgressRequest struct {
	AllanimeID    string `json:"allanimeId"`
	Title         string `json:"title"`
	EpisodeNumber int    `json:"episodeNumber"`
	PlaybackTime  int    `json:"playbackTime"`
	Duration      int    `json:"duration"`
	MediaType     string `json:"mediaType"`
}

type DownloadEpisodeRequest struct {
	Media         MediaRequest `json:"media"`
	EpisodeURL    string       `json:"episodeUrl"`
	EpisodeNumber string       `json:"episodeNumber"`
	Mode          string       `json:"mode"`
	Quality       string       `json:"quality"`
}

type DownloadEpisodeResponse struct {
	FilePath string `json:"filePath"`
	Message  string `json:"message"`
}

type SearchCoversEvent struct {
	Query     string        `json:"query"`
	Source    string        `json:"source"`
	MediaType string        `json:"mediaType"`
	Results   []MediaResult `json:"results"`
}

type RelatedAnime struct {
	MalID    int    `json:"malId"`
	Name     string `json:"name"`
	Relation string `json:"relation"`
	ImageURL string `json:"imageUrl"`
}
