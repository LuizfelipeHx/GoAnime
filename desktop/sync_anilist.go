package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// anilistClientID must be set by the user with their own AniList API client ID.
// Register at https://anilist.co/settings/developer
const anilistClientID = ""

const (
	anilistAuthURL     = "https://anilist.co/api/v2/oauth/authorize"
	anilistTokenURL    = "https://anilist.co/api/v2/oauth/token"
	anilistGraphQLURL  = "https://graphql.anilist.co"
	anilistCallbackPort = "19842"
	anilistRedirectURI = "http://127.0.0.1:19842/callback"
)

type anilistTokenData struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
	ObtainedAt  string `json:"obtained_at"`
}

func anilistTokenPath() string {
	if runtime.GOOS == "windows" {
		if d := os.Getenv("LOCALAPPDATA"); d != "" {
			return filepath.Join(d, "GoAnime", "anilist_token.json")
		}
	}
	if h := os.Getenv("HOME"); h != "" {
		return filepath.Join(h, ".local", "goanime", "anilist_token.json")
	}
	return ""
}

func (a *App) loadAniListToken() *anilistTokenData {
	p := anilistTokenPath()
	if p == "" {
		return nil
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	var token anilistTokenData
	if err := json.Unmarshal(data, &token); err != nil {
		log.Printf("anilist: erro ao ler token: %v", err)
		return nil
	}
	return &token
}

func saveAniListToken(token *anilistTokenData) error {
	p := anilistTokenPath()
	if p == "" {
		return fmt.Errorf("caminho de armazenamento nao disponivel")
	}
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0644)
}

// GetAniListSyncStatus checks whether an AniList token is stored and valid.
func (a *App) GetAniListSyncStatus() AniListSyncStatus {
	token := a.loadAniListToken()
	if token == nil || token.AccessToken == "" {
		return AniListSyncStatus{
			Connected:   false,
			TokenStored: false,
		}
	}

	status := AniListSyncStatus{
		Connected:   true,
		TokenStored: true,
	}

	// Try to fetch the profile
	profile, err := fetchAniListProfile(a.httpClient, token.AccessToken)
	if err != nil {
		log.Printf("anilist: erro ao buscar perfil: %v", err)
		status.Connected = false
		return status
	}
	status.Profile = profile

	return status
}

// StartAniListAuth starts the OAuth2 Authorization Code flow.
// Returns the URL the user should open in their browser.
func (a *App) StartAniListAuth() (string, error) {
	if anilistClientID == "" {
		return "", fmt.Errorf("client ID do AniList nao configurado. Registre em https://anilist.co/settings/developer")
	}

	// Build the authorization URL
	authURL := fmt.Sprintf(
		"%s?client_id=%s&redirect_uri=%s&response_type=code",
		anilistAuthURL,
		anilistClientID,
		anilistRedirectURI,
	)

	// Start a temporary HTTP server for the OAuth callback
	go a.startOAuthCallbackServer()

	return authURL, nil
}

func (a *App) startOAuthCallbackServer() {
	listener, err := net.Listen("tcp", "127.0.0.1:"+anilistCallbackPort)
	if err != nil {
		log.Printf("anilist: erro ao iniciar servidor de callback: %v", err)
		return
	}

	mux := http.NewServeMux()
	server := &http.Server{Handler: mux}

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Codigo de autorizacao nao recebido", http.StatusBadRequest)
			return
		}

		// Exchange the code for a token
		token, err := exchangeAniListCode(a.httpClient, code)
		if err != nil {
			log.Printf("anilist: erro ao trocar codigo: %v", err)
			fmt.Fprintf(w, "<html><body><h2>Erro na autenticacao</h2><p>%s</p></body></html>", err.Error())
		} else {
			if err := saveAniListToken(token); err != nil {
				log.Printf("anilist: erro ao salvar token: %v", err)
			}
			fmt.Fprint(w, `<html><body style="font-family:sans-serif;text-align:center;padding:60px">
				<h2>Conectado com sucesso!</h2>
				<p>Voce pode fechar esta aba e voltar ao GoAnime.</p>
			</body></html>`)
		}

		// Shut down the callback server after a short delay
		go func() {
			time.Sleep(2 * time.Second)
			_ = server.Shutdown(context.Background())
		}()
	})

	// Auto-shutdown after 5 minutes if no callback is received
	go func() {
		time.Sleep(5 * time.Minute)
		_ = server.Shutdown(context.Background())
	}()

	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		log.Printf("anilist: erro no servidor de callback: %v", err)
	}
}

func exchangeAniListCode(client *http.Client, code string) (*anilistTokenData, error) {
	payload := map[string]string{
		"grant_type":    "authorization_code",
		"client_id":     anilistClientID,
		"redirect_uri":  anilistRedirectURI,
		"code":          code,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, anilistTokenURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requisicao de token falhou: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("AniList retornou status %d: %s", resp.StatusCode, string(respBody))
	}

	var token anilistTokenData
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("erro ao decodificar token: %w", err)
	}
	token.ObtainedAt = time.Now().Format(time.RFC3339)

	return &token, nil
}

// DisconnectAniList removes the stored AniList token.
func (a *App) DisconnectAniList() error {
	p := anilistTokenPath()
	if p == "" {
		return nil
	}
	err := os.Remove(p)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("erro ao remover token: %w", err)
	}
	return nil
}

// SyncToAniList pushes local watch progress to AniList using the SaveMediaListEntry mutation.
func (a *App) SyncToAniList() error {
	token := a.loadAniListToken()
	if token == nil || token.AccessToken == "" {
		return fmt.Errorf("nao conectado ao AniList")
	}

	progress := a.GetWatchProgress()
	if len(progress) == 0 {
		return nil
	}

	var syncErrors []string
	synced := 0

	for _, p := range progress {
		// We need an AniList media ID. Try to search by title.
		mediaID, err := searchAniListMediaID(a.httpClient, token.AccessToken, p.Title)
		if err != nil {
			log.Printf("anilist sync: erro ao buscar %q: %v", p.Title, err)
			syncErrors = append(syncErrors, fmt.Sprintf("%s: %v", p.Title, err))
			continue
		}
		if mediaID == 0 {
			continue
		}

		status := "CURRENT"
		if p.ProgressPercent >= 90.0 {
			status = "COMPLETED"
		}

		err = saveAniListEntry(a.httpClient, token.AccessToken, mediaID, status, p.EpisodeNumber)
		if err != nil {
			log.Printf("anilist sync: erro ao salvar %q: %v", p.Title, err)
			syncErrors = append(syncErrors, fmt.Sprintf("%s: %v", p.Title, err))
			continue
		}
		synced++
	}

	log.Printf("anilist sync: %d anime(s) sincronizado(s)", synced)

	if len(syncErrors) > 0 {
		return fmt.Errorf("sincronizado %d anime(s) com %d erro(s): %s",
			synced, len(syncErrors), strings.Join(syncErrors, "; "))
	}
	return nil
}

// ── AniList GraphQL helpers ──

const anilistViewerQuery = `
query {
	Viewer {
		id
		name
		avatar { large }
		siteUrl
	}
}
`

func fetchAniListProfile(client *http.Client, accessToken string) (*AniListProfile, error) {
	body, err := json.Marshal(map[string]string{
		"query": anilistViewerQuery,
	})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, anilistGraphQLURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AniList retornou status %d", resp.StatusCode)
	}

	var result struct {
		Data struct {
			Viewer struct {
				ID     int    `json:"id"`
				Name   string `json:"name"`
				Avatar struct {
					Large string `json:"large"`
				} `json:"avatar"`
				SiteURL string `json:"siteUrl"`
			} `json:"Viewer"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	v := result.Data.Viewer
	if v.ID == 0 {
		return nil, fmt.Errorf("perfil nao encontrado (token invalido?)")
	}

	return &AniListProfile{
		ID:      v.ID,
		Name:    v.Name,
		Avatar:  v.Avatar.Large,
		SiteURL: v.SiteURL,
	}, nil
}

const anilistSearchQuery = `
query ($search: String) {
	Media(search: $search, type: ANIME) {
		id
	}
}
`

func searchAniListMediaID(client *http.Client, accessToken string, title string) (int, error) {
	body, err := json.Marshal(map[string]interface{}{
		"query":     anilistSearchQuery,
		"variables": map[string]string{"search": title},
	})
	if err != nil {
		return 0, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, anilistGraphQLURL, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("AniList retornou status %d", resp.StatusCode)
	}

	var result struct {
		Data struct {
			Media struct {
				ID int `json:"id"`
			} `json:"Media"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	return result.Data.Media.ID, nil
}

const anilistSaveEntryMutation = `
mutation($mediaId: Int, $status: MediaListStatus, $progress: Int) {
	SaveMediaListEntry(mediaId: $mediaId, status: $status, progress: $progress) {
		id
		status
		progress
	}
}
`

func saveAniListEntry(client *http.Client, accessToken string, mediaID int, status string, progress int) error {
	body, err := json.Marshal(map[string]interface{}{
		"query": anilistSaveEntryMutation,
		"variables": map[string]interface{}{
			"mediaId":  mediaID,
			"status":   status,
			"progress": progress,
		},
	})
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, anilistGraphQLURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("AniList retornou status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
