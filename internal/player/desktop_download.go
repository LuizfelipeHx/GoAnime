package player

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func DownloadEpisodeForDesktop(streamURL, mediaURL, mediaName string, episodeNum int) (string, error) {
	destPath, err := DesktopEpisodePath(mediaURL, mediaName, episodeNum)
	if err != nil {
		return "", err
	}

	if stat, err := os.Stat(destPath); err == nil && stat.Size() >= 1024 {
		return destPath, nil
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0o700); err != nil {
		return "", fmt.Errorf("failed to create download directory: %w", err)
	}

	if isDesktopYtDlpDownload(streamURL) {
		if err := downloadWithYtDlp(streamURL, destPath, nil); err != nil {
			return "", err
		}
	} else {
		if err := DownloadVideo(streamURL, destPath, 4, nil); err != nil {
			return "", err
		}
	}

	if stat, err := os.Stat(destPath); err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	} else if stat.Size() < 1024 {
		return "", fmt.Errorf("download failed: file too small (%d bytes)", stat.Size())
	}

	return destPath, nil
}

func DesktopEpisodePath(mediaURL, mediaName string, episodeNum int) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	folderName := DownloadFolderFormatter(mediaURL)
	if folderName == "" {
		folderName = sanitizeDesktopName(mediaName)
	}
	if folderName == "" {
		folderName = "goanime"
	}

	downloadDir := filepath.Join(homeDir, "Downloads", "GoAnime", folderName)
	if err := os.MkdirAll(downloadDir, 0o700); err != nil {
		return "", err
	}

	if episodeNum < 1 {
		episodeNum = 1
	}

	return filepath.Join(downloadDir, fmt.Sprintf("%02d.mp4", episodeNum)), nil
}

func isDesktopYtDlpDownload(streamURL string) bool {
	lower := strings.ToLower(streamURL)
	return strings.Contains(lower, "blogger.com") ||
		strings.Contains(lower, ".m3u8") ||
		strings.Contains(lower, "wixmp.com") ||
		strings.Contains(lower, "sharepoint.com")
}

func sanitizeDesktopName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	var b strings.Builder
	lastUnderscore := false
	for _, r := range value {
		isAllowed := (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '-' || r == '_' || r == ' '

		if !isAllowed {
			r = '_'
		}
		if r == ' ' {
			r = '_'
		}
		if r == '_' {
			if lastUnderscore {
				continue
			}
			lastUnderscore = true
		} else {
			lastUnderscore = false
		}
		b.WriteRune(r)
	}

	return strings.Trim(b.String(), "_ ")
}
