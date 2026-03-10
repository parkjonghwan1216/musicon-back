package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"musicon-back/internal/domain"
	"musicon-back/internal/matcher"
)

// TrackMatchRepository manages TJ song match results.
type TrackMatchRepository interface {
	Upsert(ctx context.Context, match *domain.TrackMatch) error
	FindMatchedByDeviceID(ctx context.Context, deviceID int64, limit, offset int) ([]domain.MatchedTrackResult, error)
	FindCandidateSongs(ctx context.Context, titleKeyword string) ([]matcher.CandidateSong, error)
}

type SQLiteTrackMatchRepository struct {
	db *sql.DB
}

func NewSQLiteTrackMatchRepository(db *sql.DB) *SQLiteTrackMatchRepository {
	return &SQLiteTrackMatchRepository{db: db}
}

func (r *SQLiteTrackMatchRepository) Upsert(ctx context.Context, match *domain.TrackMatch) error {
	q := `
		INSERT INTO track_matches (music_track_id, song_id, match_score, status)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (music_track_id) DO UPDATE SET
			song_id     = excluded.song_id,
			match_score = excluded.match_score,
			status      = excluded.status
	`

	if _, err := r.db.ExecContext(ctx, q,
		match.MusicTrackID, match.SongID, match.MatchScore, match.Status,
	); err != nil {
		return fmt.Errorf("failed to upsert track match: %w", err)
	}

	return nil
}

func (r *SQLiteTrackMatchRepository) FindMatchedByDeviceID(ctx context.Context, deviceID int64, limit, offset int) ([]domain.MatchedTrackResult, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	q := `
		SELECT mt.id, mt.device_id, mt.provider, mt.external_id, mt.title, mt.artist,
		       mt.album_name, mt.image_url, mt.synced_at,
		       tm.match_score, tm.status,
		       s.id, s.tj_number, s.title, s.artist, s.lyricist, s.composer,
		       s.title_chosung, s.artist_chosung, s.has_mv, s.published_at, s.created_at
		FROM music_tracks mt
		JOIN track_matches tm ON mt.id = tm.music_track_id
		LEFT JOIN songs s ON tm.song_id = s.id
		WHERE mt.device_id = ?
		ORDER BY tm.match_score DESC, mt.synced_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.QueryContext(ctx, q, deviceID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to find matched tracks: %w", err)
	}
	defer rows.Close()

	var results []domain.MatchedTrackResult
	for rows.Next() {
		var r domain.MatchedTrackResult
		var songID sql.NullInt64
		var songTjNumber sql.NullInt64
		var songTitle, songArtist, songLyricist, songComposer sql.NullString
		var songTitleChosung, songArtistChosung sql.NullString
		var songHasMV sql.NullBool
		var songPublishedAt, songCreatedAt sql.NullTime

		if err := rows.Scan(
			&r.Track.ID, &r.Track.DeviceID, &r.Track.Provider, &r.Track.ExternalID,
			&r.Track.Title, &r.Track.Artist, &r.Track.AlbumName, &r.Track.ImageURL, &r.Track.SyncedAt,
			&r.MatchScore, &r.Status,
			&songID, &songTjNumber, &songTitle, &songArtist, &songLyricist, &songComposer,
			&songTitleChosung, &songArtistChosung, &songHasMV, &songPublishedAt, &songCreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan matched track: %w", err)
		}

		if songID.Valid {
			r.Song = &domain.Song{
				ID:            songID.Int64,
				TjNumber:      int(songTjNumber.Int64),
				Title:         songTitle.String,
				Artist:        songArtist.String,
				Lyricist:      songLyricist.String,
				Composer:      songComposer.String,
				TitleChosung:  songTitleChosung.String,
				ArtistChosung: songArtistChosung.String,
				HasMV:         songHasMV.Bool,
				PublishedAt:   songPublishedAt.Time,
				CreatedAt:     songCreatedAt.Time,
			}
		}

		results = append(results, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating matched tracks: %w", err)
	}

	return results, nil
}

func (r *SQLiteTrackMatchRepository) FindCandidateSongs(ctx context.Context, titleKeyword string) ([]matcher.CandidateSong, error) {
	q := `
		SELECT id, title, artist
		FROM songs
		WHERE title LIKE ? ESCAPE '\' OR artist LIKE ? ESCAPE '\'
		LIMIT 100
	`

	escaped := escapeLikePattern(titleKeyword)
	pattern := "%" + escaped + "%"
	rows, err := r.db.QueryContext(ctx, q, pattern, pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to find candidate songs: %w", err)
	}
	defer rows.Close()

	var candidates []matcher.CandidateSong
	for rows.Next() {
		var c matcher.CandidateSong
		if err := rows.Scan(&c.ID, &c.Title, &c.Artist); err != nil {
			return nil, fmt.Errorf("failed to scan candidate song: %w", err)
		}
		candidates = append(candidates, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating candidate songs: %w", err)
	}

	return candidates, nil
}

func escapeLikePattern(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}
