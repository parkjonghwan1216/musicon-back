package service

import (
	"context"
	"fmt"

	"musicon-back/internal/domain"
	"musicon-back/internal/repository"
)

type DeviceService struct {
	repo repository.DeviceRepository
}

func NewDeviceService(repo repository.DeviceRepository) *DeviceService {
	return &DeviceService{repo: repo}
}

func (s *DeviceService) Register(ctx context.Context, token, platform string) (*domain.Device, error) {
	if token == "" {
		return nil, fmt.Errorf("expo push token is required")
	}

	device, err := s.repo.Upsert(ctx, token, platform)
	if err != nil {
		return nil, fmt.Errorf("failed to register device: %w", err)
	}

	return device, nil
}
