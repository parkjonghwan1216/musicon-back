package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"musicon-back/internal/domain"
)

const (
	youtubeAPIBase    = "https://www.googleapis.com/youtube/v3"
	youtubeMaxTracks  = 500
	youtubeMaxPages   = 15
	youtubeMaxRespLen = 5 * 1024 * 1024 // 5MB
	youtubeFetchTimeout = 120 * time.Second
)

// YouTubeProvider implements MusicProvider for YouTube using the YouTube Data API.
type YouTubeProvider struct {
	oauthCfg   *oauth2.Config
	httpClient *http.Client
}

// NewYouTubeProvider creates a new YouTube provider with the given OAuth credentials.
func NewYouTubeProvider(clientID, clientSecret string) *YouTubeProvider {
	return &YouTubeProvider{
		httpClient: &http.Client{Timeout: 30 * time.Second},
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

// FetchUserTracks fetches the user's liked videos via YouTube Data API.
// Note: AlbumName is unavailable from the YouTube Data API videos.list endpoint.
func (p *YouTubeProvider) FetchUserTracks(ctx context.Context, accessToken, _ string) ([]ExternalTrack, error) {
	ctx, cancel := context.WithTimeout(ctx, youtubeFetchTimeout)
	defer cancel()

	var tracks []ExternalTrack
	pageToken := ""

	for page := 0; page < youtubeMaxPages && len(tracks) < youtubeMaxTracks; page++ {
		params := url.Values{
			"part":       {"snippet"},
			"myRating":   {"like"},
			"maxResults": {"50"},
		}
		if pageToken != "" {
			params.Set("pageToken", pageToken)
		}

		reqURL := youtubeAPIBase + "/videos?" + params.Encode()
		body, err := p.youtubeGet(ctx, accessToken, reqURL)
		if err != nil {
			return nil, fmt.Errorf("youtube fetch liked videos: %w", err)
		}

		var resp youtubeVideoListResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("youtube parse liked videos: %w", err)
		}

		if len(resp.Items) == 0 {
			break
		}

		for _, item := range resp.Items {
			tracks = append(tracks, ExternalTrack{
				ExternalID: item.ID,
				Title:      item.Snippet.Title,
				Artist:     cleanChannelTitle(item.Snippet.ChannelTitle),
				ImageURL:   item.Snippet.Thumbnails.bestURL(),
			})
		}

		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}

	if len(tracks) > youtubeMaxTracks {
		tracks = tracks[:youtubeMaxTracks]
	}

	return tracks, nil
}

// youtubeVideoListResponse maps the YouTube Data API videos.list response.
type youtubeVideoListResponse struct {
	NextPageToken string             `json:"nextPageToken"`
	Items         []youtubeVideoItem `json:"items"`
}

type youtubeVideoItem struct {
	ID      string              `json:"id"`
	Snippet youtubeVideoSnippet `json:"snippet"`
}

type youtubeVideoSnippet struct {
	Title        string             `json:"title"`
	ChannelTitle string             `json:"channelTitle"`
	Thumbnails   youtubeThumbnails  `json:"thumbnails"`
}

type youtubeThumbnails struct {
	High    *youtubeThumbnail `json:"high"`
	Medium  *youtubeThumbnail `json:"medium"`
	Default *youtubeThumbnail `json:"default"`
}

type youtubeThumbnail struct {
	URL string `json:"url"`
}

func (t youtubeThumbnails) bestURL() string {
	if t.High != nil && t.High.URL != "" {
		return t.High.URL
	}
	if t.Medium != nil && t.Medium.URL != "" {
		return t.Medium.URL
	}
	if t.Default != nil && t.Default.URL != "" {
		return t.Default.URL
	}
	return ""
}

// cleanChannelTitle strips " - Topic" suffix from YouTube Music auto-generated topic channels.
func cleanChannelTitle(title string) string {
	if strings.HasSuffix(title, " - Topic") {
		return strings.TrimSuffix(title, " - Topic")
	}
	return title
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
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("youtube API error %d: %s", resp.StatusCode, string(respBody))
	}

	return io.ReadAll(io.LimitReader(resp.Body, youtubeMaxRespLen))
}

var _ MusicProvider = (*YouTubeProvider)(nil)
