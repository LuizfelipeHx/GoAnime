// Package download provides high-level download workflow management
package download

import (
	"fmt"
	"log"

	"github.com/alvarorichard/Goanime/internal/api"
	"github.com/alvarorichard/Goanime/internal/appflow"
	"github.com/alvarorichard/Goanime/internal/downloader"
	"github.com/alvarorichard/Goanime/internal/player"
	"github.com/alvarorichard/Goanime/internal/util"
)

// HandleDownloadRequest processes a download request from command line
func HandleDownloadRequest(request *util.DownloadRequest) error {
	util.Info("Starting enhanced download mode...")

	// Use source preference if specified
	source := request.Source
	quality := request.Quality
	if quality == "" {
		quality = "best"
	}

	util.Infof("Using source: %s, quality: %s", source, quality)

	// Try enhanced search with retry logic
	anime, err := appflow.SearchAnimeWithRetry(request.AnimeName)
	if err != nil {
		util.Errorf("Failed to search for anime: %v", err)
		return err
	}

	if request.IsRange {
		util.Infof("Downloading episodes %d-%d of %s",
			request.StartEpisode, request.EndEpisode, anime.Name)

		// Exclusive AllAnime Smart Range
		if request.AllAnimeSmart && (anime.Source == "AllAnime" || source == "allanime" || source == "AllAnime") {
			util.Info("AllAnime Smart Range enabled: mirror priority + AniSkip integration + progress UI")
			// Use player batch downloader with provided range to get consistent progress UI
			eps, err := api.GetAnimeEpisodesEnhanced(anime)
			if err == nil && len(eps) > 0 {
				if err := player.HandleBatchDownloadRange(eps, anime.URL, request.StartEpisode, request.EndEpisode); err == nil {
					return nil
				}
				// Fall through to API-based smart range if UI path fails
				util.Infof("Progress UI path failed, falling back to API smart range: %v", err)
			} else if err != nil {
				util.Infof("Enhanced episodes fetch failed for progress path: %v", err)
			}
			if err := api.DownloadAllAnimeSmartRange(anime, request.StartEpisode, request.EndEpisode, quality); err != nil {
				util.Errorf("AllAnime Smart Range failed: %v", err)
				// Fallback to normal enhanced
				if err := api.DownloadEpisodeRangeEnhanced(anime, request.StartEpisode, request.EndEpisode, quality); err != nil {
					util.Infof("Enhanced download failed, falling back to legacy: %v", err)
					// Fallback to legacy downloader
					episodes, epErr := appflow.GetAnimeEpisodesLegacy(anime.URL)
					if epErr != nil {
						return fmt.Errorf("falha ao obter episódios: %w", epErr)
					}
					dl := downloader.NewEpisodeDownloader(episodes, anime.URL)
					return dl.DownloadEpisodeRange(request.StartEpisode, request.EndEpisode)
				}
				return nil
			}
			return nil
		}

		// Try enhanced download first
		if err := api.DownloadEpisodeRangeEnhanced(anime, request.StartEpisode, request.EndEpisode, quality); err != nil {
			util.Infof("Enhanced download failed, falling back to legacy: %v", err)
			// Fallback to legacy downloader
			episodes, epErr := appflow.GetAnimeEpisodesLegacy(anime.URL)
			if epErr != nil {
				return fmt.Errorf("falha ao obter episódios: %w", epErr)
			}
			dl := downloader.NewEpisodeDownloader(episodes, anime.URL)
			return dl.DownloadEpisodeRange(request.StartEpisode, request.EndEpisode)
		}
		return nil
	} else {
		util.Infof("Downloading episode %d of %s",
			request.EpisodeNum, anime.Name)

		// Enhanced download is a placeholder - use legacy downloader
		util.Infof("Using legacy downloader for episode %d", request.EpisodeNum)
		episodes, epErr := appflow.GetAnimeEpisodesLegacy(anime.URL)
		if epErr != nil {
			return fmt.Errorf("falha ao obter episódios: %w", epErr)
		}
		dl := downloader.NewEpisodeDownloader(episodes, anime.URL)
		return dl.DownloadSingleEpisode(request.EpisodeNum)
	}
}

// Example usage functions for documentation

// ExampleSingleDownload demonstrates single episode download
func ExampleSingleDownload() {
	// Command: goanime -d "My Hero Academia" 15
	// This would create a DownloadRequest like:
	_ = &util.DownloadRequest{
		AnimeName:  "My Hero Academia",
		EpisodeNum: 15,
		IsRange:    false,
	}
	// Then call: HandleDownloadRequest(request)
	log.Println("Single download example - see DownloadRequest struct for configuration")
}
