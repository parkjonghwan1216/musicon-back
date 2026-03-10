package domain

import "time"

type MusicTrack struct {
	ID         int64     `json:"id"`
	DeviceID   int64     `json:"device_id"`
	Provider   string    `json:"provider"`
	ExternalID string    `json:"external_id"`
	Title      string    `json:"title"`
	Artist     string    `json:"artist"`
	AlbumName  string    `json:"album_name,omitempty"`
	ImageURL   string    `json:"image_url,omitempty"`
	SyncedAt   time.Time `json:"synced_at"`
}
