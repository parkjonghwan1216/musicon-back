package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL  string
	ServerPort   string
	TJAPIBaseURL string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		ServerPort:   os.Getenv("SERVER_PORT"),
		TJAPIBaseURL: os.Getenv("TJ_API_BASE_URL"),
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

	return cfg, nil
}
