package main

import (
	"fmt"
	"strings"

	"github.com/alvarorichard/Goanime/internal/api"
	"github.com/alvarorichard/Goanime/internal/models"
	"github.com/alvarorichard/Goanime/internal/player"
	"github.com/alvarorichard/Goanime/internal/scraper"
)

func buildAnimeMedia(name string, alt MediaAlternative, fallbackType string) *models.Anime {
	mediaType := strings.TrimSpace(alt.MediaType)
	if mediaType == "" {
		mediaType = fallbackType
	}
	return &models.Anime{
		Name:      name,
		URL:       strings.TrimSpace(alt.URL),
		Source:    normalizeSource(alt.Source),
		MediaType: parseMediaType(mediaType),
	}
}

func buildEpisodeItems(episodes []models.Episode) []EpisodeResult {
	items := make([]EpisodeResult, 0, min(len(episodes), maxEpisodeItems))
	for _, ep := range episodes {
		if len(items) >= maxEpisodeItems {
			break
		}
		title := strings.TrimSpace(ep.Title.English)
		if title == "" {
			title = strings.TrimSpace(ep.Title.Romaji)
		}
		if title == "" {
			title = ep.Number
		}
		num := ep.Num
		if num <= 0 {
			num = parseEpisodeNum(ep.Number)
		}
		items = append(items, EpisodeResult{
			Number: ep.Number,
			Num:    num,
			Title:  title,
			URL:    ep.URL,
		})
	}
	return items
}

func (a *App) tryGetEpisodes(req MediaRequest) (*EpisodesResponse, error) {
	if strings.TrimSpace(req.URL) == "" {
		return nil, fmt.Errorf("media URL is required")
	}

	normalizedType := strings.ToLower(strings.TrimSpace(req.MediaType))
	if normalizedType == "" {
		normalizedType = "anime"
	}

	candidates := buildOrderedMediaAlternatives(req, "watch")
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no source candidates available")
	}
	groupKey := req.GroupKey
	if strings.TrimSpace(groupKey) == "" {
		groupKey = normalizeSearchText(cleanDisplayName(req.Name))
	}

	var errs []string
	for _, candidate := range candidates {
		media := buildAnimeMedia(req.Name, candidate, normalizedType)
		if normalizedType == "tv" && strings.Contains(strings.ToLower(media.Source), "flixhq") {
			err := fmt.Errorf("TV episodes from FlixHQ are not yet available in desktop mode")
			recordSourceFailure(groupKey, "watch", candidate.Source, err)
			errs = append(errs, err.Error())
			continue
		}

		episodes, err := api.GetAnimeEpisodesEnhanced(media)
		if err != nil {
			recordSourceFailure(groupKey, "watch", candidate.Source, err)
			errs = append(errs, fmt.Sprintf("%s: %v", candidate.Source, err))
			continue
		}
		clearSourceFailure(groupKey, "watch", candidate.Source)
		items := buildEpisodeItems(episodes)
		note := ""
		if !strings.EqualFold(candidate.Source, req.Source) || strings.TrimSpace(candidate.URL) != strings.TrimSpace(req.URL) {
			note = fmt.Sprintf("Fonte ajustada automaticamente para %s.", candidate.Source)
		}
		return &EpisodesResponse{
			Name:           req.Name,
			Source:         normalizeSource(candidate.Source),
			MediaType:      normalizedType,
			Count:          len(items),
			Episodes:       items,
			ResolvedSource: normalizeSource(candidate.Source),
			ResolvedURL:    strings.TrimSpace(candidate.URL),
			Note:           note,
		}, nil
	}

	if len(errs) == 0 {
		return nil, fmt.Errorf("no episodes found")
	}
	return nil, fmt.Errorf("failed to load episodes: %s", strings.Join(errs, " | "))
}

func (a *App) tryGetStream(req StreamRequest) (*StreamResponse, error) {
	if strings.TrimSpace(req.Media.URL) == "" {
		return nil, fmt.Errorf("media URL is required")
	}
	if strings.TrimSpace(req.EpisodeNumber) == "" {
		req.EpisodeNumber = "1"
	}

	candidates := buildOrderedMediaAlternatives(req.Media, "watch")
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no source candidates available")
	}
	groupKey := req.Media.GroupKey
	if strings.TrimSpace(groupKey) == "" {
		groupKey = normalizeSearchText(cleanDisplayName(req.Media.Name))
	}

	var errs []string
	for _, candidate := range candidates {
		response, err := a.streamForCandidate(req, candidate)
		if err != nil {
			recordSourceFailure(groupKey, "watch", candidate.Source, err)
			errs = append(errs, fmt.Sprintf("%s: %v", candidate.Source, err))
			continue
		}
		clearSourceFailure(groupKey, "watch", candidate.Source)
		if !strings.EqualFold(candidate.Source, req.Media.Source) || strings.TrimSpace(candidate.URL) != strings.TrimSpace(req.Media.URL) {
			response.Note = fmt.Sprintf("Stream ajustado automaticamente para %s.", candidate.Source)
		}
		response.ResolvedSource = normalizeSource(candidate.Source)
		response.ResolvedURL = strings.TrimSpace(candidate.URL)
		return response, nil
	}

	if len(errs) == 0 {
		return nil, fmt.Errorf("failed to load stream")
	}
	return nil, fmt.Errorf("failed to load stream: %s", strings.Join(errs, " | "))
}

func (a *App) streamForCandidate(req StreamRequest, candidate MediaAlternative) (*StreamResponse, error) {
	quality := strings.TrimSpace(req.Quality)
	if quality == "" {
		quality = "best"
	}
	mode := strings.TrimSpace(req.Mode)
	if mode == "" {
		mode = "sub"
	}
	episodeNumber := strings.TrimSpace(req.EpisodeNumber)
	if episodeNumber == "" {
		episodeNumber = "1"
	}

	anime := buildAnimeMedia(req.Media.Name, candidate, req.Media.MediaType)
	var episode *models.Episode
	if strings.EqualFold(strings.TrimSpace(candidate.Source), strings.TrimSpace(req.Media.Source)) && strings.TrimSpace(candidate.URL) == strings.TrimSpace(req.Media.URL) {
		episode = &models.Episode{URL: strings.TrimSpace(req.EpisodeURL), Number: episodeNumber, Num: parseEpisodeNum(episodeNumber)}
	} else {
		episodes, err := api.GetAnimeEpisodesEnhanced(anime)
		if err != nil {
			return nil, err
		}
		episode = findMatchingEpisode(episodes, episodeNumber)
		if episode == nil {
			return nil, fmt.Errorf("episode %s not found in %s", episodeNumber, candidate.Source)
		}
	}

	if anime.MediaType == models.MediaTypeMovie && strings.Contains(strings.ToLower(anime.Source), "flixhq") {
		streamURL, subtitles, err := api.GetFlixHQStreamURL(anime, episode, quality)
		if err != nil {
			return nil, err
		}
		responseSubs := make([]SubtitleResult, 0, len(subtitles))
		for _, sub := range subtitles {
			if strings.TrimSpace(sub.URL) == "" {
				continue
			}
			responseSubs = append(responseSubs, SubtitleResult{
				URL:      sub.URL,
				ProxyURL: a.toProxyURL(sub.URL),
				Language: sub.Language,
				Label:    sub.Label,
			})
		}
		return &StreamResponse{
			StreamURL:          streamURL,
			ProxyURL:           a.toProxyURL(streamURL),
			ContentType:        detectContentType(streamURL),
			Subtitles:          responseSubs,
			ResolvedEpisodeURL: episode.URL,
		}, nil
	}

	if isAllAnimeMedia(anime) {
		animeID := extractAllAnimeID(anime.URL)
		if animeID == "" {
			return nil, fmt.Errorf("could not resolve AllAnime ID")
		}
		client := scraper.NewAllAnimeClient()
		streamURL, _, err := client.GetEpisodeURL(animeID, episodeNumber, mode, quality)
		if err != nil {
			return nil, err
		}
		return &StreamResponse{
			StreamURL:          streamURL,
			ProxyURL:           a.toProxyURL(streamURL),
			ContentType:        detectContentType(streamURL),
			ResolvedEpisodeURL: episode.URL,
		}, nil
	}

	streamURL, err := player.GetVideoURLForEpisodeEnhanced(episode, anime)
	if err != nil {
		return nil, err
	}
	return &StreamResponse{
		StreamURL:          streamURL,
		ProxyURL:           a.toProxyURL(streamURL),
		ContentType:        detectContentType(streamURL),
		ResolvedEpisodeURL: episode.URL,
	}, nil
}

func findMatchingEpisode(items []models.Episode, episodeNumber string) *models.Episode {
	targetNum := parseEpisodeNum(episodeNumber)
	targetText := strings.TrimSpace(episodeNumber)
	for i := range items {
		if targetNum > 0 && items[i].Num == targetNum {
			return &items[i]
		}
		if targetNum > 0 && parseEpisodeNum(items[i].Number) == targetNum {
			return &items[i]
		}
		if targetText != "" && strings.TrimSpace(items[i].Number) == targetText {
			return &items[i]
		}
	}
	return nil
}
