package domain

import "time"

type MusicAccount struct {
	ID             int64     `json:"id"`
	DeviceID       int64     `json:"device_id"`
	Provider       string    `json:"provider"`
	ProviderUserID string    `json:"provider_user_id"`
	DisplayName    string    `json:"display_name"`
	AccessToken    string    `json:"-"`
	RefreshToken   string    `json:"-"`
	TokenExpiresAt time.Time `json:"-"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
