package util

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const ytdlpMaxAgeDays = 30

// CheckYtDlpAge verifica se o binário yt-dlp está desatualizado (>30 dias).
// Exibe um aviso visível no terminal se for o caso; caso contrário, não faz nada.
func CheckYtDlpAge() {
	binaryPath := FindYtDlpBinary()
	if binaryPath == "" {
		// Binário não encontrado: go-ytdlp fará o download automaticamente quando necessário
		return
	}

	info, err := os.Stat(binaryPath)
	if err != nil {
		return
	}

	age := time.Since(info.ModTime())
	if age <= ytdlpMaxAgeDays*24*time.Hour {
		return
	}

	days := int(age.Hours() / 24)
	PrintWarningBox(
		"yt-dlp desatualizado",
		fmt.Sprintf(
			"O binário yt-dlp tem %d dias (recomendado: <%d dias).\n"+
				"Downloads podem falhar com fontes atuais.\n"+
				"Execute 'goanime --update' para atualizar.",
			days, ytdlpMaxAgeDays,
		),
	)
}

// FindYtDlpBinary retorna o caminho do binário yt-dlp instalado pelo go-ytdlp,
// ou "" se não encontrado.
func FindYtDlpBinary() string {
	var candidates []string

	if runtime.GOOS == "windows" {
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData != "" {
			candidates = append(candidates,
				filepath.Join(localAppData, "go-ytdlp", "yt-dlp.exe"),
			)
		}
		// Fallback: diretório temp
		candidates = append(candidates,
			filepath.Join(os.TempDir(), "go-ytdlp", "yt-dlp.exe"),
		)
	} else {
		home := os.Getenv("HOME")
		if home != "" {
			candidates = append(candidates,
				filepath.Join(home, ".cache", "go-ytdlp", "yt-dlp"),
				filepath.Join(home, ".local", "bin", "yt-dlp"),
			)
		}
		candidates = append(candidates,
			filepath.Join(os.TempDir(), "go-ytdlp", "yt-dlp"),
			"/usr/local/bin/yt-dlp",
			"/usr/bin/yt-dlp",
		)
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
