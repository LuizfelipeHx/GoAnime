package util

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/muesli/termenv"
)

var Logger *log.Logger

// getColoredPrefix returns a styled prefix with colors
func getColoredPrefix() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#6366F1")).
		Bold(true).
		Padding(0, 1).
		MarginRight(1)
	return style.Render("GoAnime")
}

// InitLogger initializes the beautiful charmbracelet logger
func InitLogger() {
	Logger = log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    IsDebug,
		ReportTimestamp: IsDebug,
		TimeFormat:      "15:04:05",
		Prefix:          getColoredPrefix(),
	})

	// Set the appropriate log level based on debug mode
	if IsDebug {
		Logger.SetLevel(log.DebugLevel)
		Logger.SetColorProfile(termenv.TrueColor)
		Logger.Debug("Debug logging enabled with charmbracelet/log")
	} else {
		Logger.SetLevel(log.InfoLevel)
		Logger.SetColorProfile(termenv.TrueColor)
	}
}

// Debug logs a debug message (only when debug mode is enabled)
func Debug(msg interface{}, keyvals ...interface{}) {
	if IsDebug && Logger != nil {
		Logger.Debug(fmt.Sprintf("%v", msg), keyvals...)
	}
}

// Info logs an info message
func Info(msg interface{}, keyvals ...interface{}) {
	if Logger != nil {
		Logger.Info(fmt.Sprintf("%v", msg), keyvals...)
	}
}

// Warn logs a warning message
func Warn(msg interface{}, keyvals ...interface{}) {
	if Logger != nil {
		Logger.Warn(fmt.Sprintf("%v", msg), keyvals...)
	}
}

// Error logs an error message
func Error(msg interface{}, keyvals ...interface{}) {
	if Logger != nil {
		Logger.Error(fmt.Sprintf("%v", msg), keyvals...)
	}
}

// Fatal logs a fatal message and exits
func Fatal(msg interface{}, keyvals ...interface{}) {
	if Logger != nil {
		Logger.Fatal(fmt.Sprintf("%v", msg), keyvals...)
	}
}

// Debugf logs a formatted debug message (only when debug mode is enabled)
func Debugf(format string, args ...interface{}) {
	if IsDebug && Logger != nil {
		Logger.Debug(fmt.Sprintf(format, args...))
	}
}

// Infof logs a formatted info message
func Infof(format string, args ...interface{}) {
	if Logger != nil {
		Logger.Info(fmt.Sprintf(format, args...))
	}
}

// Warnf logs a formatted warning message
func Warnf(format string, args ...interface{}) {
	if Logger != nil {
		Logger.Warn(fmt.Sprintf(format, args...))
	}
}

// Errorf logs a formatted error message
func Errorf(format string, args ...interface{}) {
	if Logger != nil {
		Logger.Error(fmt.Sprintf(format, args...))
	}
}

// PrintErrorBox exibe um painel de erro estilizado no terminal.
// Deve ser chamado antes de menus interativos para garantir visibilidade.
func PrintErrorBox(title, message string) {
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true)

	msgStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFD7D7"))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF4444")).
		Padding(0, 2).
		MarginTop(1).
		MarginBottom(1)

	content := titleStyle.Render("✖ "+title) + "\n" + msgStyle.Render(message)
	fmt.Fprintln(os.Stderr, boxStyle.Render(content))
}

// PrintWarningBox exibe um painel de aviso estilizado no terminal.
func PrintWarningBox(title, message string) {
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1A1A00")).
		Bold(true)

	msgStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3D3000"))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FFB300")).
		Background(lipgloss.Color("#FFF8E1")).
		Padding(0, 2).
		MarginTop(1).
		MarginBottom(1)

	content := titleStyle.Render("⚠  "+title) + "\n" + msgStyle.Render(message)
	fmt.Fprintln(os.Stderr, boxStyle.Render(content))
}

// FriendlyPlaybackError traduz erros técnicos para mensagens amigáveis.
// Retorna (título, detalhe) para passar ao PrintErrorBox.
func FriendlyPlaybackError(err error) (string, string) {
	if err == nil {
		return "", ""
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "mpv not found"):
		return "Player não encontrado",
			"O MPV não foi encontrado no sistema.\nReinstale em: https://mpv.io/installation/"
	case strings.Contains(msg, "failed to start mpv"):
		return "Player falhou ao iniciar",
			"O MPV não conseguiu abrir. Tente reinstalá-lo:\nhttps://mpv.io/installation/"
	case strings.Contains(msg, "timeout waiting for mpv"):
		return "Timeout de reprodução",
			"O player demorou demais para responder.\nA fonte pode estar lenta. Tente outro servidor."
	case strings.Contains(msg, "no valid video URL"),
		strings.Contains(msg, "failed to extract video URL"):
		return "Vídeo não encontrado",
			"Não foi possível obter o link do episódio.\nA fonte pode estar fora do ar. Tente outro servidor."
	case strings.Contains(msg, "go-ytdlp download failed"),
		strings.Contains(msg, "exit code 1"):
		return "Download falhou",
			"O yt-dlp encontrou um erro.\nVerifique sua conexão ou tente atualizar o yt-dlp."
	case strings.Contains(msg, "bad status"):
		return "Servidor indisponível",
			"O servidor de vídeo retornou um erro.\nVerifique sua conexão e tente novamente."
	default:
		return "Erro de reprodução",
			"Algo deu errado. Execute com --debug para mais detalhes.\n" + msg
	}
}
