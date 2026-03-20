package domain

import "time"

const (
	ReservationStatusActive    = "active"
	ReservationStatusMatched   = "matched"
	ReservationStatusCancelled = "cancelled"
)

type Reservation struct {
	ID            int64      `json:"id"`
	DeviceID      int64      `json:"device_id"`
	Artist        string     `json:"artist"`
	Title         string     `json:"title"`
	Status        string     `json:"status"`
	MatchedSongID *int64     `json:"matched_song_id,omitempty"`
	NotifiedAt    *time.Time `json:"notified_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// IsArtistOnly returns true if this reservation subscribes to all songs by the artist
// (title is empty), rather than a specific song.
func (r Reservation) IsArtistOnly() bool {
	return r.Title == ""
}

// MatchResult holds a matched reservation along with the song and push token
// for sending notifications.
type MatchResult struct {
	Reservation   Reservation `json:"reservation"`
	Song          Song        `json:"song"`
	ExpoPushToken string      `json:"expo_push_token"`
}
