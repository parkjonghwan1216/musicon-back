package service

import (
	"context"
	"fmt"
	"time"

	"musicon-back/internal/domain"
	"musicon-back/internal/provider"
	"musicon-back/internal/repository"
)

// MusicAuthService handles OAuth connection/disconnection and token refresh.
type MusicAuthService struct {
	accountRepo repository.MusicAccountRepository
	trackRepo   repository.MusicTrackRepository
	deviceRepo  repository.DeviceRepository
	registry    *provider.Registry
}

func NewMusicAuthService(
	accountRepo repository.MusicAccountRepository,
	trackRepo repository.MusicTrackRepository,
	deviceRepo repository.DeviceRepository,
	registry *provider.Registry,
) *MusicAuthService {
	return &MusicAuthService{
		accountRepo: accountRepo,
		trackRepo:   trackRepo,
		deviceRepo:  deviceRepo,
		registry:    registry,
	}
}

// Connect exchanges an OAuth code for tokens and saves the account.
func (s *MusicAuthService) Connect(ctx context.Context, providerName, code, redirectURI, expoPushToken string) (*domain.MusicAccount, error) {
	device, err := s.deviceRepo.FindByToken(ctx, expoPushToken)
	if err != nil {
		return nil, fmt.Errorf("failed to find device: %w", err)
	}
	if device == nil {
		return nil, fmt.Errorf("device not found, register first")
	}

	p, err := s.registry.Get(providerName)
	if err != nil {
		return nil, err
	}

	tokenResult, err := p.ExchangeCode(ctx, code, redirectURI)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	account := &domain.MusicAccount{
		DeviceID:       device.ID,
		Provider:       providerName,
		ProviderUserID: tokenResult.UserID,
		DisplayName:    tokenResult.DisplayName,
		AccessToken:    tokenResult.AccessToken,
		RefreshToken:   tokenResult.RefreshToken,
		TokenExpiresAt: tokenResult.ExpiresAt,
	}

	saved, err := s.accountRepo.Upsert(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("failed to save music account: %w", err)
	}

	return saved, nil
}

// Disconnect removes the account and its associated tracks.
func (s *MusicAuthService) Disconnect(ctx context.Context, expoPushToken, providerName string) error {
	device, err := s.deviceRepo.FindByToken(ctx, expoPushToken)
	if err != nil {
		return fmt.Errorf("failed to find device: %w", err)
	}
	if device == nil {
		return fmt.Errorf("device not found")
	}

	// Delete tracks first (cascade would handle this, but explicit is clearer)
	if err := s.trackRepo.DeleteByDeviceAndProvider(ctx, device.ID, providerName); err != nil {
		return fmt.Errorf("failed to delete tracks: %w", err)
	}

	if err := s.accountRepo.DeleteByDeviceAndProvider(ctx, device.ID, providerName); err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}

	return nil
}

// ListAccounts returns all connected music accounts for a device.
func (s *MusicAuthService) ListAccounts(ctx context.Context, expoPushToken string) ([]domain.MusicAccount, error) {
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

	return accounts, nil
}

// EnsureValidToken refreshes the access token if it has expired.
// Returns the account with a valid token.
func (s *MusicAuthService) EnsureValidToken(ctx context.Context, account *domain.MusicAccount) (*domain.MusicAccount, error) {
	if time.Now().Before(account.TokenExpiresAt.Add(-1 * time.Minute)) {
		return account, nil
	}

	p, err := s.registry.Get(account.Provider)
	if err != nil {
		return nil, err
	}

	tokenResult, err := p.RefreshAccessToken(ctx, account.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	if err := s.accountRepo.UpdateTokens(ctx,
		account.ID, tokenResult.AccessToken, tokenResult.RefreshToken, tokenResult.ExpiresAt,
	); err != nil {
		return nil, fmt.Errorf("failed to update tokens: %w", err)
	}

	return &domain.MusicAccount{
		ID:             account.ID,
		DeviceID:       account.DeviceID,
		Provider:       account.Provider,
		ProviderUserID: account.ProviderUserID,
		DisplayName:    account.DisplayName,
		AccessToken:    tokenResult.AccessToken,
		RefreshToken:   tokenResult.RefreshToken,
		TokenExpiresAt: tokenResult.ExpiresAt,
		CreatedAt:      account.CreatedAt,
		UpdatedAt:      account.UpdatedAt,
	}, nil
}
