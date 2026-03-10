package domain

const (
	MatchStatusMatched   = "matched"
	MatchStatusUnmatched = "unmatched"
)

type TrackMatch struct {
	ID           int64   `json:"id"`
	MusicTrackID int64   `json:"music_track_id"`
	SongID       *int64  `json:"song_id,omitempty"`
	MatchScore   float64 `json:"match_score"`
	Status       string  `json:"status"`
}

// MatchedTrackResult combines a music track with its TJ match result for API responses.
type MatchedTrackResult struct {
	Track      MusicTrack `json:"track"`
	Song       *Song      `json:"song,omitempty"`
	MatchScore float64    `json:"match_score"`
	Status     string     `json:"status"`
}
