package provider

import (
	"context"
	"time"
)

// TokenResult holds the result of an OAuth token exchange or refresh.
type TokenResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	UserID       string
	DisplayName  string
}

// ExternalTrack represents a track fetched from an external music service.
type ExternalTrack struct {
	ExternalID string
	Title      string
	Artist     string
	AlbumName  string
	ImageURL   string
}

// MusicProvider defines the interface for external music service integrations.
type MusicProvider interface {
	// Name returns the provider identifier (e.g. "spotify", "youtube").
	Name() string
	// ExchangeCode exchanges an OAuth authorization code for tokens.
	ExchangeCode(ctx context.Context, code, redirectURI string) (*TokenResult, error)
	// RefreshAccessToken refreshes an expired access token.
	RefreshAccessToken(ctx context.Context, refreshToken string) (*TokenResult, error)
	// FetchUserTracks fetches the user's liked/top tracks.
	FetchUserTracks(ctx context.Context, accessToken string) ([]ExternalTrack, error)
}
