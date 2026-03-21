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
	"musicon-back/internal/matcher"
)

const (
	youtubeAPIBase       = "https://www.googleapis.com/youtube/v3"
	youtubeMaxResults    = 50
	youtubeMaxTracks     = 500
	youtubeMusicCategory = "10" // YouTube video category ID for Music
)

// YouTubeProvider implements MusicProvider for YouTube Music via the YouTube Data API.
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
			Scopes:       []string{"https://www.googleapis.com/auth/youtube.readonly"},
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

func (p *YouTubeProvider) FetchUserTracks(ctx context.Context, accessToken string) ([]ExternalTrack, error) {
	// Fetch liked videos (playlist ID = "LL")
	return p.fetchPlaylistItems(ctx, accessToken, "LL", youtubeMaxTracks)
}

type youtubePlaylistResponse struct {
	Items []struct {
		Snippet struct {
			ResourceID struct {
				VideoID string `json:"videoId"`
			} `json:"resourceId"`
			Title      string `json:"title"`
			Thumbnails struct {
				Default struct {
					URL string `json:"url"`
				} `json:"default"`
			} `json:"thumbnails"`
		} `json:"snippet"`
	} `json:"items"`
	NextPageToken string `json:"nextPageToken"`
}

// playlistItem holds raw data fetched from YouTube playlistItems API before category filtering.
type playlistItem struct {
	videoID  string
	title    string
	imageURL string
}

func (p *YouTubeProvider) fetchPlaylistItems(ctx context.Context, accessToken, playlistID string, maxCount int) ([]ExternalTrack, error) {
	// Step 1: Collect raw playlist items with video IDs.
	var items []playlistItem
	pageToken := ""

	for len(items) < maxCount {
		params := url.Values{
			"part":       {"snippet"},
			"playlistId": {playlistID},
			"maxResults": {fmt.Sprintf("%d", youtubeMaxResults)},
		}
		if pageToken != "" {
			params.Set("pageToken", pageToken)
		}

		reqURL := youtubeAPIBase + "/playlistItems?" + params.Encode()
		body, err := p.youtubeGet(ctx, accessToken, reqURL)
		if err != nil {
			return nil, err
		}

		var resp youtubePlaylistResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("youtube parse playlist failed: %w", err)
		}

		for _, item := range resp.Items {
			items = append(items, playlistItem{
				videoID:  item.Snippet.ResourceID.VideoID,
				title:    item.Snippet.Title,
				imageURL: item.Snippet.Thumbnails.Default.URL,
			})
		}

		if resp.NextPageToken == "" || len(resp.Items) == 0 {
			break
		}
		pageToken = resp.NextPageToken
	}

	if len(items) > maxCount {
		items = items[:maxCount]
	}

	// Step 2: Filter by Music category (categoryId=10) in batches.
	musicIDs, err := p.filterMusicVideoIDs(ctx, accessToken, items)
	if err != nil {
		return nil, err
	}

	// Step 3: Build ExternalTrack list from filtered items.
	var all []ExternalTrack
	for _, item := range items {
		if !musicIDs[item.videoID] {
			continue
		}
		artist, title := matcher.ParseYouTubeTitle(item.title)
		all = append(all, ExternalTrack{
			ExternalID: item.videoID,
			Title:      title,
			Artist:     artist,
			ImageURL:   item.imageURL,
		})
	}

	return all, nil
}

// youtubeVideosResponse represents the response from YouTube /videos API.
type youtubeVideosResponse struct {
	Items []struct {
		ID      string `json:"id"`
		Snippet struct {
			CategoryID string `json:"categoryId"`
		} `json:"snippet"`
	} `json:"items"`
}

// filterMusicVideoIDs calls the YouTube /videos API in batches of 50 to check each video's
// category. Returns a set of video IDs that belong to the Music category (categoryId=10).
func (p *YouTubeProvider) filterMusicVideoIDs(ctx context.Context, accessToken string, items []playlistItem) (map[string]bool, error) {
	musicIDs := make(map[string]bool, len(items))

	for i := 0; i < len(items); i += youtubeMaxResults {
		end := i + youtubeMaxResults
		if end > len(items) {
			end = len(items)
		}

		var ids []string
		for _, item := range items[i:end] {
			ids = append(ids, item.videoID)
		}

		params := url.Values{
			"part": {"snippet"},
			"id":   {strings.Join(ids, ",")},
		}
		reqURL := youtubeAPIBase + "/videos?" + params.Encode()

		body, err := p.youtubeGet(ctx, accessToken, reqURL)
		if err != nil {
			return nil, fmt.Errorf("youtube filter videos failed: %w", err)
		}

		var resp youtubeVideosResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("youtube parse videos failed: %w", err)
		}

		for _, v := range resp.Items {
			if v.Snippet.CategoryID == youtubeMusicCategory {
				musicIDs[v.ID] = true
			}
		}
	}

	return musicIDs, nil
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
