package service

import (
	"context"
	"fmt"
	"log"

	"musicon-back/internal/domain"
	"musicon-back/internal/matcher"
	"musicon-back/internal/provider"
	"musicon-back/internal/repository"
)

// MusicSyncService orchestrates fetching external tracks and matching them to TJ songs.
type MusicSyncService struct {
	accountRepo    repository.MusicAccountRepository
	trackRepo      repository.MusicTrackRepository
	matchRepo      repository.TrackMatchRepository
	deviceRepo     repository.DeviceRepository
	registry       *provider.Registry
	musicAuthSvc   *MusicAuthService
}

func NewMusicSyncService(
	accountRepo repository.MusicAccountRepository,
	trackRepo repository.MusicTrackRepository,
	matchRepo repository.TrackMatchRepository,
	deviceRepo repository.DeviceRepository,
	registry *provider.Registry,
	musicAuthSvc *MusicAuthService,
) *MusicSyncService {
	return &MusicSyncService{
		accountRepo:  accountRepo,
		trackRepo:    trackRepo,
		matchRepo:    matchRepo,
		deviceRepo:   deviceRepo,
		registry:     registry,
		musicAuthSvc: musicAuthSvc,
	}
}

// SyncResult holds the summary of a sync operation.
type SyncResult struct {
	Provider      string `json:"provider"`
	TracksFound   int    `json:"tracks_found"`
	TracksMatched int    `json:"tracks_matched"`
}

// Sync fetches tracks from all connected providers and matches them against the TJ DB.
func (s *MusicSyncService) Sync(ctx context.Context, expoPushToken string) ([]SyncResult, error) {
	device, err := s.deviceRepo.FindByToken(ctx, expoPushToken)
	if err != nil {
		return nil, fmt.Errorf("failed to find device: %w", err)
	}
	if device == nil {
		return nil, fmt.Errorf("device not found")
	}

	accounts, err := s.accountRepo.FindByDeviceID(ctx, device.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}

	if len(accounts) == 0 {
		return nil, fmt.Errorf("no music accounts connected")
	}

	var results []SyncResult
	for i := range accounts {
		result, err := s.syncProvider(ctx, &accounts[i])
		if err != nil {
			log.Printf("sync failed for provider %s: %v", accounts[i].Provider, err)
			continue
		}
		results = append(results, *result)
	}

	return results, nil
}

func (s *MusicSyncService) syncProvider(ctx context.Context, account *domain.MusicAccount) (*SyncResult, error) {
	// Ensure token is valid
	account, err := s.musicAuthSvc.EnsureValidToken(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	// Get the provider implementation
	p, err := s.registry.Get(account.Provider)
	if err != nil {
		return nil, err
	}

	// Fetch external tracks
	externalTracks, err := p.FetchUserTracks(ctx, account.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tracks: %w", err)
	}

	// Convert to domain tracks and persist
	domainTracks := make([]domain.MusicTrack, 0, len(externalTracks))
	for _, et := range externalTracks {
		domainTracks = append(domainTracks, domain.MusicTrack{
			ExternalID: et.ExternalID,
			Title:      et.Title,
			Artist:     et.Artist,
			AlbumName:  et.AlbumName,
			ImageURL:   et.ImageURL,
		})
	}

	if _, err := s.trackRepo.UpsertMany(ctx, account.DeviceID, account.Provider, domainTracks); err != nil {
		return nil, fmt.Errorf("failed to save tracks: %w", err)
	}

	// Reload tracks to get IDs
	allTracks, err := s.trackRepo.FindByDeviceID(ctx, account.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to reload tracks: %w", err)
	}

	// Match each track against TJ DB
	matchedCount := 0
	for _, track := range allTracks {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if track.Provider != account.Provider {
			continue
		}

		matched, err := s.matchTrack(ctx, &track)
		if err != nil {
			log.Printf("match failed for track %q: %v", track.Title, err)
			continue
		}
		if matched {
			matchedCount++
		}
	}

	return &SyncResult{
		Provider:      account.Provider,
		TracksFound:   len(externalTracks),
		TracksMatched: matchedCount,
	}, nil
}

func (s *MusicSyncService) matchTrack(ctx context.Context, track *domain.MusicTrack) (bool, error) {
	// Use the first few characters of the normalized title as search keyword
	keyword := extractKeyword(track.Title)
	if keyword == "" {
		// Save as unmatched
		return false, s.matchRepo.Upsert(ctx, &domain.TrackMatch{
			MusicTrackID: track.ID,
			Status:       domain.MatchStatusUnmatched,
		})
	}

	// Find candidate songs using SQL LIKE filter
	candidates, err := s.matchRepo.FindCandidateSongs(ctx, keyword)
	if err != nil {
		return false, fmt.Errorf("failed to find candidates: %w", err)
	}

	// Run fuzzy matching
	best := matcher.BestMatch(track.Title, track.Artist, candidates)

	if best == nil {
		return false, s.matchRepo.Upsert(ctx, &domain.TrackMatch{
			MusicTrackID: track.ID,
			Status:       domain.MatchStatusUnmatched,
		})
	}

	songID := best.SongID
	return true, s.matchRepo.Upsert(ctx, &domain.TrackMatch{
		MusicTrackID: track.ID,
		SongID:       &songID,
		MatchScore:   best.Score,
		Status:       domain.MatchStatusMatched,
	})
}

// extractKeyword returns the first meaningful portion of a title for SQL LIKE filtering.
func extractKeyword(title string) string {
	normalized := matcher.Normalize(title)
	if len(normalized) == 0 {
		return ""
	}

	// Use up to first 10 runes as keyword for LIKE search
	runes := []rune(normalized)
	if len(runes) > 10 {
		runes = runes[:10]
	}
	return string(runes)
}
