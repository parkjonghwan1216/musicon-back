package domain

import "time"

type Song struct {
	ID            int64     `json:"id"`
	TjNumber      int       `json:"tj_number"`
	Title         string    `json:"title"`
	Artist        string    `json:"artist"`
	Lyricist      string    `json:"lyricist,omitempty"`
	Composer      string    `json:"composer,omitempty"`
	TitleChosung  string    `json:"title_chosung,omitempty"`
	ArtistChosung string    `json:"artist_chosung,omitempty"`
	HasMV         bool      `json:"has_mv"`
	PublishedAt   time.Time `json:"published_at"`
	CreatedAt     time.Time `json:"created_at"`
}
