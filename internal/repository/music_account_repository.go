package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"musicon-back/internal/domain"
)

// MusicAccountRepository manages OAuth account persistence.
type MusicAccountRepository interface {
	Upsert(ctx context.Context, account *domain.MusicAccount) (*domain.MusicAccount, error)
	FindByDeviceAndProvider(ctx context.Context, deviceID int64, provider string) (*domain.MusicAccount, error)
	FindByDeviceID(ctx context.Context, deviceID int64) ([]domain.MusicAccount, error)
	DeleteByDeviceAndProvider(ctx context.Context, deviceID int64, provider string) error
	UpdateTokens(ctx context.Context, id int64, accessToken, refreshToken string, expiresAt time.Time) error
}

type SQLiteMusicAccountRepository struct {
	db *sql.DB
}

func NewSQLiteMusicAccountRepository(db *sql.DB) *SQLiteMusicAccountRepository {
	return &SQLiteMusicAccountRepository{db: db}
}

func (r *SQLiteMusicAccountRepository) Upsert(ctx context.Context, account *domain.MusicAccount) (*domain.MusicAccount, error) {
	q := `
		INSERT INTO music_accounts (device_id, provider, provider_user_id, display_name,
		                            access_token, refresh_token, token_expires_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (device_id, provider) DO UPDATE SET
			provider_user_id = excluded.provider_user_id,
			display_name     = excluded.display_name,
			access_token     = excluded.access_token,
			refresh_token    = excluded.refresh_token,
			token_expires_at = excluded.token_expires_at,
			updated_at       = excluded.updated_at
	`

	now := time.Now()
	if _, err := r.db.ExecContext(ctx, q,
		account.DeviceID, account.Provider, account.ProviderUserID, account.DisplayName,
		account.AccessToken, account.RefreshToken, account.TokenExpiresAt, now,
	); err != nil {
		return nil, fmt.Errorf("failed to upsert music account: %w", err)
	}

	return r.FindByDeviceAndProvider(ctx, account.DeviceID, account.Provider)
}

func (r *SQLiteMusicAccountRepository) FindByDeviceAndProvider(ctx context.Context, deviceID int64, provider string) (*domain.MusicAccount, error) {
	q := `
		SELECT id, device_id, provider, provider_user_id, display_name,
		       access_token, refresh_token, token_expires_at, created_at, updated_at
		FROM music_accounts
		WHERE device_id = ? AND provider = ?
	`

	var a domain.MusicAccount
	err := r.db.QueryRowContext(ctx, q, deviceID, provider).Scan(
		&a.ID, &a.DeviceID, &a.Provider, &a.ProviderUserID, &a.DisplayName,
		&a.AccessToken, &a.RefreshToken, &a.TokenExpiresAt, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find music account: %w", err)
	}

	return &a, nil
}

func (r *SQLiteMusicAccountRepository) FindByDeviceID(ctx context.Context, deviceID int64) ([]domain.MusicAccount, error) {
	q := `
		SELECT id, device_id, provider, provider_user_id, display_name,
		       access_token, refresh_token, token_expires_at, created_at, updated_at
		FROM music_accounts
		WHERE device_id = ?
		ORDER BY provider ASC
	`

	rows, err := r.db.QueryContext(ctx, q, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to find music accounts: %w", err)
	}
	defer rows.Close()

	var accounts []domain.MusicAccount
	for rows.Next() {
		var a domain.MusicAccount
		if err := rows.Scan(
			&a.ID, &a.DeviceID, &a.Provider, &a.ProviderUserID, &a.DisplayName,
			&a.AccessToken, &a.RefreshToken, &a.TokenExpiresAt, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan music account: %w", err)
		}
		accounts = append(accounts, a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating music accounts: %w", err)
	}

	return accounts, nil
}

func (r *SQLiteMusicAccountRepository) DeleteByDeviceAndProvider(ctx context.Context, deviceID int64, provider string) error {
	q := `DELETE FROM music_accounts WHERE device_id = ? AND provider = ?`

	if _, err := r.db.ExecContext(ctx, q, deviceID, provider); err != nil {
		return fmt.Errorf("failed to delete music account: %w", err)
	}

	return nil
}

func (r *SQLiteMusicAccountRepository) UpdateTokens(ctx context.Context, id int64, accessToken, refreshToken string, expiresAt time.Time) error {
	q := `
		UPDATE music_accounts
		SET access_token = ?, refresh_token = ?, token_expires_at = ?, updated_at = ?
		WHERE id = ?
	`

	now := time.Now()
	if _, err := r.db.ExecContext(ctx, q, accessToken, refreshToken, expiresAt, now, id); err != nil {
		return fmt.Errorf("failed to update music account tokens: %w", err)
	}

	return nil
}
