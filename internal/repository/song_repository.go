package repository

import (
	"context"
	"database/sql"
	"fmt"

	"musicon-back/internal/domain"
)

type SongRepository interface {
	Search(ctx context.Context, query string, limit, offset int) ([]domain.Song, error)
	FindByTjNumber(ctx context.Context, tjNumber int) (*domain.Song, error)
	UpsertMany(ctx context.Context, songs []domain.Song) (int64, error)
}

type SQLiteSongRepository struct {
	db *sql.DB
}

func NewSQLiteSongRepository(db *sql.DB) *SQLiteSongRepository {
	return &SQLiteSongRepository{db: db}
}

func (r *SQLiteSongRepository) Search(ctx context.Context, query string, limit, offset int) ([]domain.Song, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	likePattern := "%" + query + "%"
	prefixPattern := query + "%"

	q := `
		SELECT id, tj_number, title, artist, lyricist, composer,
		       title_chosung, artist_chosung, has_mv, published_at, created_at
		FROM songs
		WHERE title LIKE ?
		   OR artist LIKE ?
		   OR title_chosung LIKE ?
		   OR artist_chosung LIKE ?
		   OR CAST(tj_number AS TEXT) LIKE ?
		ORDER BY
		  CASE WHEN CAST(tj_number AS TEXT) = ? THEN 0
		       WHEN title LIKE ? THEN 1
		       ELSE 2 END,
		  title ASC
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.QueryContext(ctx, q,
		likePattern, likePattern, likePattern, likePattern,
		likePattern,
		query, prefixPattern,
		limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search songs: %w", err)
	}
	defer rows.Close()

	return scanSongs(rows)
}

func (r *SQLiteSongRepository) FindByTjNumber(ctx context.Context, tjNumber int) (*domain.Song, error) {
	q := `
		SELECT id, tj_number, title, artist, lyricist, composer,
		       title_chosung, artist_chosung, has_mv, published_at, created_at
		FROM songs
		WHERE tj_number = ?
	`

	var s domain.Song
	err := r.db.QueryRowContext(ctx, q, tjNumber).Scan(
		&s.ID, &s.TjNumber, &s.Title, &s.Artist,
		&s.Lyricist, &s.Composer, &s.TitleChosung, &s.ArtistChosung,
		&s.HasMV, &s.PublishedAt, &s.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find song by TJ number: %w", err)
	}

	return &s, nil
}

func (r *SQLiteSongRepository) UpsertMany(ctx context.Context, songs []domain.Song) (int64, error) {
	if len(songs) == 0 {
		return 0, nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	q := `
		INSERT INTO songs (tj_number, title, artist, lyricist, composer,
		                    title_chosung, artist_chosung, has_mv, published_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (tj_number) DO UPDATE SET
			title = excluded.title,
			artist = excluded.artist,
			lyricist = excluded.lyricist,
			composer = excluded.composer,
			title_chosung = excluded.title_chosung,
			artist_chosung = excluded.artist_chosung,
			has_mv = excluded.has_mv,
			published_at = excluded.published_at
	`

	stmt, err := tx.PrepareContext(ctx, q)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare upsert statement: %w", err)
	}
	defer stmt.Close()

	var total int64
	for _, s := range songs {
		result, err := stmt.ExecContext(ctx,
			s.TjNumber, s.Title, s.Artist, s.Lyricist, s.Composer,
			s.TitleChosung, s.ArtistChosung, s.HasMV, s.PublishedAt,
		)
		if err != nil {
			return total, fmt.Errorf("failed to upsert song: %w", err)
		}
		affected, _ := result.RowsAffected()
		total += affected
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return total, nil
}

func scanSongs(rows *sql.Rows) ([]domain.Song, error) {
	var songs []domain.Song
	for rows.Next() {
		var s domain.Song
		err := rows.Scan(
			&s.ID, &s.TjNumber, &s.Title, &s.Artist,
			&s.Lyricist, &s.Composer, &s.TitleChosung, &s.ArtistChosung,
			&s.HasMV, &s.PublishedAt, &s.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan song: %w", err)
		}
		songs = append(songs, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating song rows: %w", err)
	}

	return songs, nil
}
