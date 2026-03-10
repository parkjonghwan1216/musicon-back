package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"musicon-back/internal/domain"
)

// MusicTrackRepository manages external music track persistence.
type MusicTrackRepository interface {
	UpsertMany(ctx context.Context, deviceID int64, provider string, tracks []domain.MusicTrack) (int64, error)
	FindByDeviceID(ctx context.Context, deviceID int64) ([]domain.MusicTrack, error)
	DeleteByDeviceAndProvider(ctx context.Context, deviceID int64, provider string) error
}

type SQLiteMusicTrackRepository struct {
	db *sql.DB
}

func NewSQLiteMusicTrackRepository(db *sql.DB) *SQLiteMusicTrackRepository {
	return &SQLiteMusicTrackRepository{db: db}
}

func (r *SQLiteMusicTrackRepository) UpsertMany(ctx context.Context, deviceID int64, provider string, tracks []domain.MusicTrack) (int64, error) {
	if len(tracks) == 0 {
		return 0, nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	q := `
		INSERT INTO music_tracks (device_id, provider, external_id, title, artist, album_name, image_url, synced_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (device_id, provider, external_id) DO UPDATE SET
			title      = excluded.title,
			artist     = excluded.artist,
			album_name = excluded.album_name,
			image_url  = excluded.image_url,
			synced_at  = excluded.synced_at
	`

	stmt, err := tx.PrepareContext(ctx, q)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare upsert statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	var total int64
	for _, t := range tracks {
		result, err := stmt.ExecContext(ctx,
			deviceID, provider, t.ExternalID, t.Title, t.Artist, t.AlbumName, t.ImageURL, now,
		)
		if err != nil {
			return total, fmt.Errorf("failed to upsert music track: %w", err)
		}
		affected, _ := result.RowsAffected()
		total += affected
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return total, nil
}

func (r *SQLiteMusicTrackRepository) FindByDeviceID(ctx context.Context, deviceID int64) ([]domain.MusicTrack, error) {
	q := `
		SELECT id, device_id, provider, external_id, title, artist, album_name, image_url, synced_at
		FROM music_tracks
		WHERE device_id = ?
		ORDER BY synced_at DESC
	`

	rows, err := r.db.QueryContext(ctx, q, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to find music tracks: %w", err)
	}
	defer rows.Close()

	var tracks []domain.MusicTrack
	for rows.Next() {
		var t domain.MusicTrack
		if err := rows.Scan(
			&t.ID, &t.DeviceID, &t.Provider, &t.ExternalID, &t.Title, &t.Artist,
			&t.AlbumName, &t.ImageURL, &t.SyncedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan music track: %w", err)
		}
		tracks = append(tracks, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating music tracks: %w", err)
	}

	return tracks, nil
}

func (r *SQLiteMusicTrackRepository) DeleteByDeviceAndProvider(ctx context.Context, deviceID int64, provider string) error {
	q := `DELETE FROM music_tracks WHERE device_id = ? AND provider = ?`

	if _, err := r.db.ExecContext(ctx, q, deviceID, provider); err != nil {
		return fmt.Errorf("failed to delete music tracks: %w", err)
	}

	return nil
}
