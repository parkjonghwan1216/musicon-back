package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL    string
	ServerPort     string
	TJAPIBaseURL   string
	BaseURL        string
	BleveIndexPath string

	SpotifyClientID     string
	SpotifyClientSecret string
	YouTubeClientID     string
	YouTubeClientSecret string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		ServerPort:   os.Getenv("SERVER_PORT"),
		TJAPIBaseURL: os.Getenv("TJ_API_BASE_URL"),

		SpotifyClientID:     os.Getenv("SPOTIFY_CLIENT_ID"),
		SpotifyClientSecret: os.Getenv("SPOTIFY_CLIENT_SECRET"),
		YouTubeClientID:     os.Getenv("YOUTUBE_CLIENT_ID"),
		YouTubeClientSecret: os.Getenv("YOUTUBE_CLIENT_SECRET"),
	}

	if cfg.DatabaseURL == "" {
		cfg.DatabaseURL = "musicon.db"
	}

	if cfg.ServerPort == "" {
		cfg.ServerPort = "3000"
	}

	if cfg.TJAPIBaseURL == "" {
		cfg.TJAPIBaseURL = "https://www.tjmedia.com/legacy/api/newSongOfMonth"
	}

	cfg.BleveIndexPath = os.Getenv("BLEVE_INDEX_PATH")
	if cfg.BleveIndexPath == "" {
		cfg.BleveIndexPath = "data/bleve_index"
	}

	cfg.BaseURL = os.Getenv("BASE_URL")
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:" + cfg.ServerPort
	}

	return cfg, nil
}
