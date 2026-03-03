package repository

import (
	"context"
	"database/sql"
	"fmt"

	"musicon-back/internal/domain"
)

type DeviceRepository interface {
	Upsert(ctx context.Context, token, platform string) (*domain.Device, error)
	FindByToken(ctx context.Context, token string) (*domain.Device, error)
	Delete(ctx context.Context, id int64) error
}

type SQLiteDeviceRepository struct {
	db *sql.DB
}

func NewSQLiteDeviceRepository(db *sql.DB) *SQLiteDeviceRepository {
	return &SQLiteDeviceRepository{db: db}
}

func (r *SQLiteDeviceRepository) Upsert(ctx context.Context, token, platform string) (*domain.Device, error) {
	q := `
		INSERT INTO devices (expo_push_token, platform)
		VALUES (?, ?)
		ON CONFLICT (expo_push_token) DO UPDATE SET
			platform = excluded.platform
	`

	if _, err := r.db.ExecContext(ctx, q, token, platform); err != nil {
		return nil, fmt.Errorf("failed to upsert device: %w", err)
	}

	return r.FindByToken(ctx, token)
}

func (r *SQLiteDeviceRepository) FindByToken(ctx context.Context, token string) (*domain.Device, error) {
	q := `
		SELECT id, expo_push_token, platform, created_at
		FROM devices
		WHERE expo_push_token = ?
	`

	var d domain.Device
	err := r.db.QueryRowContext(ctx, q, token).Scan(
		&d.ID, &d.ExpoPushToken, &d.Platform, &d.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find device by token: %w", err)
	}

	return &d, nil
}

func (r *SQLiteDeviceRepository) Delete(ctx context.Context, id int64) error {
	q := `DELETE FROM devices WHERE id = ?`

	if _, err := r.db.ExecContext(ctx, q, id); err != nil {
		return fmt.Errorf("failed to delete device: %w", err)
	}

	return nil
}
