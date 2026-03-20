package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"musicon-back/internal/domain"
)

type ReservationRepository interface {
	Create(ctx context.Context, deviceID int64, artist, title string) (*domain.Reservation, error)
	FindByDeviceID(ctx context.Context, deviceID int64) ([]domain.Reservation, error)
	FindByID(ctx context.Context, id int64) (*domain.Reservation, error)
	Update(ctx context.Context, id int64, artist, title string) (*domain.Reservation, error)
	Delete(ctx context.Context, id int64) error
	FindActiveWithTokens(ctx context.Context) ([]activeReservation, error)
	MarkAsMatched(ctx context.Context, reservationID, songID int64) error
	MarkAsNotified(ctx context.Context, reservationID int64) error
	HasNotified(ctx context.Context, reservationID, songID int64) (bool, error)
	RecordNotification(ctx context.Context, reservationID, songID int64) error
}

// activeReservation holds reservation data joined with the device's push token.
type activeReservation struct {
	Reservation   domain.Reservation
	ExpoPushToken string
}

type SQLiteReservationRepository struct {
	db *sql.DB
}

func NewSQLiteReservationRepository(db *sql.DB) *SQLiteReservationRepository {
	return &SQLiteReservationRepository{db: db}
}

func (r *SQLiteReservationRepository) Create(ctx context.Context, deviceID int64, artist, title string) (*domain.Reservation, error) {
	q := `
		INSERT INTO reservations (device_id, artist, title)
		VALUES (?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, q, deviceID, artist, title)
	if err != nil {
		return nil, fmt.Errorf("failed to create reservation: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return r.FindByID(ctx, id)
}

func (r *SQLiteReservationRepository) FindByID(ctx context.Context, id int64) (*domain.Reservation, error) {
	q := `
		SELECT id, device_id, artist, title, status, matched_song_id, notified_at, created_at, updated_at
		FROM reservations
		WHERE id = ?
	`

	var res domain.Reservation
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&res.ID, &res.DeviceID, &res.Artist, &res.Title, &res.Status,
		&res.MatchedSongID, &res.NotifiedAt, &res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find reservation: %w", err)
	}

	return &res, nil
}

func (r *SQLiteReservationRepository) FindByDeviceID(ctx context.Context, deviceID int64) ([]domain.Reservation, error) {
	q := `
		SELECT id, device_id, artist, title, status, matched_song_id, notified_at, created_at, updated_at
		FROM reservations
		WHERE device_id = ?
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, q, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to find reservations: %w", err)
	}
	defer rows.Close()

	var reservations []domain.Reservation
	for rows.Next() {
		var res domain.Reservation
		if err := rows.Scan(
			&res.ID, &res.DeviceID, &res.Artist, &res.Title, &res.Status,
			&res.MatchedSongID, &res.NotifiedAt, &res.CreatedAt, &res.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan reservation: %w", err)
		}
		reservations = append(reservations, res)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating reservations: %w", err)
	}

	return reservations, nil
}

func (r *SQLiteReservationRepository) Update(ctx context.Context, id int64, artist, title string) (*domain.Reservation, error) {
	q := `
		UPDATE reservations
		SET artist = ?, title = ?, updated_at = ?
		WHERE id = ?
	`

	now := time.Now()
	if _, err := r.db.ExecContext(ctx, q, artist, title, now, id); err != nil {
		return nil, fmt.Errorf("failed to update reservation: %w", err)
	}

	return r.FindByID(ctx, id)
}

func (r *SQLiteReservationRepository) Delete(ctx context.Context, id int64) error {
	q := `DELETE FROM reservations WHERE id = ?`

	if _, err := r.db.ExecContext(ctx, q, id); err != nil {
		return fmt.Errorf("failed to delete reservation: %w", err)
	}

	return nil
}

// FindActiveWithTokens returns active reservations that still need matching:
// - Artist+title reservations: status='active' AND notified_at IS NULL (one-time match)
// - Artist-only reservations: status='active' AND notified_at is always NULL
//   (uses reservation_notifications table for per-song dedup instead)
func (r *SQLiteReservationRepository) FindActiveWithTokens(ctx context.Context) ([]activeReservation, error) {
	q := `
		SELECT r.id, r.device_id, r.artist, r.title, r.status,
		       r.matched_song_id, r.notified_at, r.created_at, r.updated_at,
		       d.expo_push_token
		FROM reservations r
		JOIN devices d ON r.device_id = d.id
		WHERE r.status = ? AND r.notified_at IS NULL
	`

	rows, err := r.db.QueryContext(ctx, q, domain.ReservationStatusActive)
	if err != nil {
		return nil, fmt.Errorf("failed to find active reservations: %w", err)
	}
	defer rows.Close()

	var results []activeReservation
	for rows.Next() {
		var ar activeReservation
		if err := rows.Scan(
			&ar.Reservation.ID, &ar.Reservation.DeviceID,
			&ar.Reservation.Artist, &ar.Reservation.Title, &ar.Reservation.Status,
			&ar.Reservation.MatchedSongID, &ar.Reservation.NotifiedAt,
			&ar.Reservation.CreatedAt, &ar.Reservation.UpdatedAt,
			&ar.ExpoPushToken,
		); err != nil {
			return nil, fmt.Errorf("failed to scan active reservation: %w", err)
		}
		results = append(results, ar)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating active reservations: %w", err)
	}

	return results, nil
}

func (r *SQLiteReservationRepository) MarkAsMatched(ctx context.Context, reservationID, songID int64) error {
	q := `
		UPDATE reservations
		SET status = ?, matched_song_id = ?, updated_at = ?
		WHERE id = ?
	`

	now := time.Now()
	if _, err := r.db.ExecContext(ctx, q, domain.ReservationStatusMatched, songID, now, reservationID); err != nil {
		return fmt.Errorf("failed to mark reservation as matched: %w", err)
	}

	return nil
}

func (r *SQLiteReservationRepository) MarkAsNotified(ctx context.Context, reservationID int64) error {
	q := `
		UPDATE reservations
		SET notified_at = ?, updated_at = ?
		WHERE id = ?
	`

	now := time.Now()
	if _, err := r.db.ExecContext(ctx, q, now, now, reservationID); err != nil {
		return fmt.Errorf("failed to mark reservation as notified: %w", err)
	}

	return nil
}

func (r *SQLiteReservationRepository) HasNotified(ctx context.Context, reservationID, songID int64) (bool, error) {
	q := `SELECT 1 FROM reservation_notifications WHERE reservation_id = ? AND song_id = ?`

	var exists int
	err := r.db.QueryRowContext(ctx, q, reservationID, songID).Scan(&exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check notification history: %w", err)
	}

	return true, nil
}

func (r *SQLiteReservationRepository) RecordNotification(ctx context.Context, reservationID, songID int64) error {
	q := `INSERT OR IGNORE INTO reservation_notifications (reservation_id, song_id) VALUES (?, ?)`

	if _, err := r.db.ExecContext(ctx, q, reservationID, songID); err != nil {
		return fmt.Errorf("failed to record notification: %w", err)
	}

	return nil
}
