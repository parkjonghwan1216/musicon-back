package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"musicon-back/internal/domain"
)

const (
	spotifyAuthURL  = "https://accounts.spotify.com/authorize"
	spotifyTokenURL = "https://accounts.spotify.com/api/token"
	spotifyAPIBase  = "https://api.spotify.com/v1"
	spotifyMaxTracks = 500
)

// SpotifyProvider implements MusicProvider for Spotify.
type SpotifyProvider struct {
	oauthCfg   *oauth2.Config
	httpClient *http.Client
}

// NewSpotifyProvider creates a new Spotify provider with the given OAuth credentials.
func NewSpotifyProvider(clientID, clientSecret string) *SpotifyProvider {
	return &SpotifyProvider{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		oauthCfg: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  spotifyAuthURL,
				TokenURL: spotifyTokenURL,
			},
			Scopes: []string{"user-top-read", "user-library-read"},
		},
	}
}

func (p *SpotifyProvider) Name() string {
	return domain.ProviderSpotify
}

func (p *SpotifyProvider) ExchangeCode(ctx context.Context, code, redirectURI string) (*TokenResult, error) {
	cfg := *p.oauthCfg
	cfg.RedirectURL = redirectURI

	token, err := cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("spotify token exchange failed: %w", err)
	}

	userID, displayName, err := p.fetchUserProfile(ctx, token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("spotify fetch profile failed: %w", err)
	}

	return &TokenResult{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    token.Expiry,
		UserID:       userID,
		DisplayName:  displayName,
	}, nil
}

func (p *SpotifyProvider) RefreshAccessToken(ctx context.Context, refreshToken string) (*TokenResult, error) {
	src := p.oauthCfg.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken})
	token, err := src.Token()
	if err != nil {
		return nil, fmt.Errorf("spotify token refresh failed: %w", err)
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

func (p *SpotifyProvider) FetchUserTracks(ctx context.Context, accessToken, _ string) ([]ExternalTrack, error) {
	var tracks []ExternalTrack

	// Fetch top tracks (medium term)
	topTracks, err := p.fetchTopTracks(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	tracks = append(tracks, topTracks...)

	// Fetch saved/liked tracks
	savedTracks, err := p.fetchSavedTracks(ctx, accessToken, spotifyMaxTracks-len(tracks))
	if err != nil {
		return nil, err
	}
	tracks = append(tracks, savedTracks...)

	// Deduplicate by external ID
	return deduplicateTracks(tracks), nil
}

func (p *SpotifyProvider) fetchTopTracks(ctx context.Context, accessToken string) ([]ExternalTrack, error) {
	url := spotifyAPIBase + "/me/top/tracks?time_range=medium_term&limit=50"
	return p.fetchSpotifyTracks(ctx, accessToken, url)
}

func (p *SpotifyProvider) fetchSavedTracks(ctx context.Context, accessToken string, maxCount int) ([]ExternalTrack, error) {
	var all []ExternalTrack
	url := fmt.Sprintf("%s/me/tracks?limit=50", spotifyAPIBase)

	for url != "" && len(all) < maxCount {
		tracks, nextURL, err := p.fetchSavedTracksPage(ctx, accessToken, url)
		if err != nil {
			return nil, err
		}
		all = append(all, tracks...)
		url = nextURL
	}

	if len(all) > maxCount {
		all = all[:maxCount]
	}

	return all, nil
}

type spotifyTrackResponse struct {
	Items []struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Artists []struct {
			Name string `json:"name"`
		} `json:"artists"`
		Album struct {
			Name   string `json:"name"`
			Images []struct {
				URL string `json:"url"`
			} `json:"images"`
		} `json:"album"`
	} `json:"items"`
	Next string `json:"next"`
}

type spotifySavedTracksResponse struct {
	Items []struct {
		Track struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Artists []struct {
				Name string `json:"name"`
			} `json:"artists"`
			Album struct {
				Name   string `json:"name"`
				Images []struct {
					URL string `json:"url"`
				} `json:"images"`
			} `json:"album"`
		} `json:"track"`
	} `json:"items"`
	Next string `json:"next"`
}

func (p *SpotifyProvider) fetchSpotifyTracks(ctx context.Context, accessToken, url string) ([]ExternalTrack, error) {
	body, err := p.spotifyGet(ctx, accessToken, url)
	if err != nil {
		return nil, err
	}

	var resp spotifyTrackResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("spotify parse tracks failed: %w", err)
	}

	tracks := make([]ExternalTrack, 0, len(resp.Items))
	for _, item := range resp.Items {
		tracks = append(tracks, ExternalTrack{
			ExternalID: item.ID,
			Title:      item.Name,
			Artist:     joinArtists(item.Artists),
			AlbumName:  item.Album.Name,
			ImageURL:   firstImageURL(item.Album.Images),
		})
	}

	return tracks, nil
}

func (p *SpotifyProvider) fetchSavedTracksPage(ctx context.Context, accessToken, url string) ([]ExternalTrack, string, error) {
	body, err := p.spotifyGet(ctx, accessToken, url)
	if err != nil {
		return nil, "", err
	}

	var resp spotifySavedTracksResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, "", fmt.Errorf("spotify parse saved tracks failed: %w", err)
	}

	tracks := make([]ExternalTrack, 0, len(resp.Items))
	for _, item := range resp.Items {
		tracks = append(tracks, ExternalTrack{
			ExternalID: item.Track.ID,
			Title:      item.Track.Name,
			Artist:     joinArtists(item.Track.Artists),
			AlbumName:  item.Track.Album.Name,
			ImageURL:   firstImageURL(item.Track.Album.Images),
		})
	}

	return tracks, resp.Next, nil
}

func (p *SpotifyProvider) spotifyGet(ctx context.Context, accessToken, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("spotify create request failed: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("spotify request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("spotify API error %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

func (p *SpotifyProvider) fetchUserProfile(ctx context.Context, accessToken string) (string, string, error) {
	body, err := p.spotifyGet(ctx, accessToken, spotifyAPIBase+"/me")
	if err != nil {
		return "", "", err
	}

	var profile struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
	}
	if err := json.Unmarshal(body, &profile); err != nil {
		return "", "", fmt.Errorf("spotify parse profile failed: %w", err)
	}

	return profile.ID, profile.DisplayName, nil
}

func joinArtists(artists []struct {
	Name string `json:"name"`
}) string {
	names := make([]string, 0, len(artists))
	for _, a := range artists {
		names = append(names, a.Name)
	}
	return strings.Join(names, ", ")
}

func firstImageURL(images []struct {
	URL string `json:"url"`
}) string {
	if len(images) > 0 {
		return images[0].URL
	}
	return ""
}

func deduplicateTracks(tracks []ExternalTrack) []ExternalTrack {
	seen := make(map[string]struct{}, len(tracks))
	result := make([]ExternalTrack, 0, len(tracks))
	for _, t := range tracks {
		if _, ok := seen[t.ExternalID]; ok {
			continue
		}
		seen[t.ExternalID] = struct{}{}
		result = append(result, t)
	}
	return result
}

// ensure SpotifyProvider implements MusicProvider at compile time.
var _ MusicProvider = (*SpotifyProvider)(nil)

