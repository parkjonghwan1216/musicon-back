package domain

import "time"

type Device struct {
	ID            int64     `json:"id"`
	ExpoPushToken string    `json:"expo_push_token"`
	Platform      string    `json:"platform"`
	CreatedAt     time.Time `json:"created_at"`
}
