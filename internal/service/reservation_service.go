package service

import (
	"context"
	"fmt"

	"musicon-back/internal/domain"
	"musicon-back/internal/repository"
)

type ReservationService struct {
	reservationRepo repository.ReservationRepository
	deviceRepo      repository.DeviceRepository
}

func NewReservationService(
	reservationRepo repository.ReservationRepository,
	deviceRepo repository.DeviceRepository,
) *ReservationService {
	return &ReservationService{
		reservationRepo: reservationRepo,
		deviceRepo:      deviceRepo,
	}
}

func (s *ReservationService) Create(ctx context.Context, token, artist, title string) (*domain.Reservation, error) {
	if artist == "" {
		return nil, fmt.Errorf("artist is required")
	}

	device, err := s.deviceRepo.FindByToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to find device: %w", err)
	}
	if device == nil {
		return nil, fmt.Errorf("device not found, register first")
	}

	reservation, err := s.reservationRepo.Create(ctx, device.ID, artist, title)
	if err != nil {
		return nil, fmt.Errorf("failed to create reservation: %w", err)
	}

	return reservation, nil
}

func (s *ReservationService) ListByDevice(ctx context.Context, token string) ([]domain.Reservation, error) {
	device, err := s.deviceRepo.FindByToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to find device: %w", err)
	}
	if device == nil {
		return nil, fmt.Errorf("device not found")
	}

	reservations, err := s.reservationRepo.FindByDeviceID(ctx, device.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list reservations: %w", err)
	}

	return reservations, nil
}

func (s *ReservationService) Update(ctx context.Context, token string, id int64, artist, title string) (*domain.Reservation, error) {
	if artist == "" {
		return nil, fmt.Errorf("artist is required")
	}

	device, err := s.deviceRepo.FindByToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to find device: %w", err)
	}
	if device == nil {
		return nil, fmt.Errorf("device not found")
	}

	existing, err := s.reservationRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find reservation: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("reservation not found")
	}
	if existing.DeviceID != device.ID {
		return nil, fmt.Errorf("unauthorized: reservation belongs to another device")
	}

	reservation, err := s.reservationRepo.Update(ctx, id, artist, title)
	if err != nil {
		return nil, fmt.Errorf("failed to update reservation: %w", err)
	}

	return reservation, nil
}

func (s *ReservationService) Delete(ctx context.Context, token string, id int64) error {
	device, err := s.deviceRepo.FindByToken(ctx, token)
	if err != nil {
		return fmt.Errorf("failed to find device: %w", err)
	}
	if device == nil {
		return fmt.Errorf("device not found")
	}

	existing, err := s.reservationRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find reservation: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("reservation not found")
	}
	if existing.DeviceID != device.ID {
		return fmt.Errorf("unauthorized: reservation belongs to another device")
	}

	if err := s.reservationRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete reservation: %w", err)
	}

	return nil
}
