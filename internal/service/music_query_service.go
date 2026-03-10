package service

import (
	"context"
	"fmt"

	"musicon-back/internal/domain"
	"musicon-back/internal/repository"
)

// MusicQueryService provides read-only access to match results.
type MusicQueryService struct {
	matchRepo  repository.TrackMatchRepository
	deviceRepo repository.DeviceRepository
}

func NewMusicQueryService(
	matchRepo repository.TrackMatchRepository,
	deviceRepo repository.DeviceRepository,
) *MusicQueryService {
	return &MusicQueryService{
		matchRepo:  matchRepo,
		deviceRepo: deviceRepo,
	}
}

// GetMatches returns matched track results for a device with pagination.
func (s *MusicQueryService) GetMatches(ctx context.Context, expoPushToken string, limit, offset int) ([]domain.MatchedTrackResult, error) {
	device, err := s.deviceRepo.FindByToken(ctx, expoPushToken)
	if err != nil {
		return nil, fmt.Errorf("failed to find device: %w", err)
	}
	if device == nil {
		return nil, fmt.Errorf("device not found")
	}

	results, err := s.matchRepo.FindMatchedByDeviceID(ctx, device.ID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get matches: %w", err)
	}

	return results, nil
}
