package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"musicon-back/internal/domain"
)

const (
	youtubeAPIBase     = "https://www.googleapis.com/youtube/v3"
	scriptTimeout      = 60 * time.Second
	maxStderrLen       = 200
)

// YouTubeProvider implements MusicProvider for YouTube Music via ytmusicapi Python sidecar.
type YouTubeProvider struct {
	oauthCfg   *oauth2.Config
	httpClient *http.Client
	scriptPath string
}

// NewYouTubeProvider creates a new YouTube provider with the given OAuth credentials.
// It validates that scriptPath exists and is a .py file.
func NewYouTubeProvider(clientID, clientSecret, scriptPath string) *YouTubeProvider {
	absPath, err := filepath.Abs(filepath.Clean(scriptPath))
	if err != nil {
		panic(fmt.Sprintf("youtube: invalid script path %q: %v", scriptPath, err))
	}
	if strings.Contains(filepath.Clean(scriptPath), "..") {
		panic(fmt.Sprintf("youtube: path traversal not allowed: %q", scriptPath))
	}
	if !strings.HasSuffix(absPath, ".py") {
		panic(fmt.Sprintf("youtube: script must be a .py file: %q", absPath))
	}
	if info, err := os.Stat(absPath); err != nil || info.IsDir() {
		panic(fmt.Sprintf("youtube: script not found or is a directory: %q", absPath))
	}

	return &YouTubeProvider{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		scriptPath: absPath,
		oauthCfg: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint:     google.Endpoint,
			Scopes:       []string{"https://www.googleapis.com/auth/youtube"},
		},
	}
}

func (p *YouTubeProvider) Name() string {
	return domain.ProviderYouTube
}

func (p *YouTubeProvider) ExchangeCode(ctx context.Context, code, redirectURI string) (*TokenResult, error) {
	cfg := *p.oauthCfg
	cfg.RedirectURL = redirectURI

	token, err := cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("youtube token exchange failed: %w", err)
	}

	userID, displayName, err := p.fetchChannelInfo(ctx, token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("youtube fetch channel failed: %w", err)
	}

	return &TokenResult{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    token.Expiry,
		UserID:       userID,
		DisplayName:  displayName,
	}, nil
}

func (p *YouTubeProvider) RefreshAccessToken(ctx context.Context, refreshToken string) (*TokenResult, error) {
	src := p.oauthCfg.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken})
	token, err := src.Token()
	if err != nil {
		return nil, fmt.Errorf("youtube token refresh failed: %w", err)
	}

	newRefresh := token.RefreshToken
	if newRefresh == "" {
		newRefresh = refreshToken
	}

	return &TokenResult{
		AccessToken:  token.AccessToken,
		RefreshToken: newRefresh,
		ExpiresAt:    token.Expiry,
	}, nil
}

// scriptInput is the JSON payload sent to the Python sidecar via stdin.
type scriptInput struct {
	AccessToken  string `json:"access_token"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// scriptTrack maps to the JSON output from the Python sidecar.
type scriptTrack struct {
	ExternalID string `json:"external_id"`
	Title      string `json:"title"`
	Artist     string `json:"artist"`
	AlbumName  string `json:"album_name"`
	ImageURL   string `json:"image_url"`
}

func (p *YouTubeProvider) FetchUserTracks(ctx context.Context, accessToken string) ([]ExternalTrack, error) {
	ctx, cancel := context.WithTimeout(ctx, scriptTimeout)
	defer cancel()

	input := scriptInput{
		AccessToken:  accessToken,
		ClientID:     p.oauthCfg.ClientID,
		ClientSecret: p.oauthCfg.ClientSecret,
	}

	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("ytmusic script: failed to marshal input: %w", err)
	}

	cmd := exec.CommandContext(ctx, "python3", p.scriptPath)
	cmd.Stdin = bytes.NewReader(inputJSON)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrMsg := stderr.String()
		if len(stderrMsg) > maxStderrLen {
			stderrMsg = stderrMsg[:maxStderrLen] + "..."
		}
		return nil, fmt.Errorf("ytmusic script failed: %w, stderr: %s", err, stderrMsg)
	}

	var tracks []scriptTrack
	if err := json.Unmarshal(stdout.Bytes(), &tracks); err != nil {
		return nil, fmt.Errorf("ytmusic script: failed to parse output: %w", err)
	}

	result := make([]ExternalTrack, len(tracks))
	for i, t := range tracks {
		result[i] = ExternalTrack{
			ExternalID: t.ExternalID,
			Title:      t.Title,
			Artist:     t.Artist,
			AlbumName:  t.AlbumName,
			ImageURL:   t.ImageURL,
		}
	}

	return result, nil
}

type youtubeChannelResponse struct {
	Items []struct {
		ID      string `json:"id"`
		Snippet struct {
			Title string `json:"title"`
		} `json:"snippet"`
	} `json:"items"`
}

func (p *YouTubeProvider) fetchChannelInfo(ctx context.Context, accessToken string) (string, string, error) {
	params := url.Values{
		"part": {"snippet"},
		"mine": {"true"},
	}
	reqURL := youtubeAPIBase + "/channels?" + params.Encode()

	body, err := p.youtubeGet(ctx, accessToken, reqURL)
	if err != nil {
		return "", "", err
	}

	var resp youtubeChannelResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", "", fmt.Errorf("youtube parse channel failed: %w", err)
	}

	if len(resp.Items) == 0 {
		return "", "", fmt.Errorf("youtube channel not found")
	}

	return resp.Items[0].ID, resp.Items[0].Snippet.Title, nil
}

func (p *YouTubeProvider) youtubeGet(ctx context.Context, accessToken, reqURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("youtube create request failed: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("youtube request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("youtube API error %d: %s", resp.StatusCode, string(respBody))
	}

	return io.ReadAll(resp.Body)
}

var _ MusicProvider = (*YouTubeProvider)(nil)
