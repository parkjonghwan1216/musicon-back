package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "modernc.org/sqlite"

	"musicon-back/internal/fetcher"
	"musicon-back/internal/repository"
)

func main() {
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "musicon.db"
	}

	tjBaseURL := os.Getenv("TJ_API_BASE_URL")
	if tjBaseURL == "" {
		tjBaseURL = "https://www.tjmedia.com/legacy/api/newSongOfMonth"
	}

	db, err := sql.Open("sqlite", dbURL)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		log.Fatalf("Failed to enable WAL mode: %v", err)
	}

	// Run migrations
	migrationSQL, err := os.ReadFile("migrations/001_create_songs.sql")
	if err != nil {
		log.Fatalf("Failed to read migration file: %v", err)
	}
	if _, err := db.Exec(string(migrationSQL)); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	repo := repository.NewSQLiteSongRepository(db)
	tj := fetcher.NewTJFetcher(tjBaseURL)
	ctx := context.Background()

	now := time.Now()
	startYear := 2001
	endYear := now.Year()
	endMonth := int(now.Month())

	var totalInserted int64

	for year := startYear; year <= endYear; year++ {
		maxMonth := 12
		if year == endYear {
			maxMonth = endMonth
		}

		for month := 1; month <= maxMonth; month++ {
			label := fmt.Sprintf("%04d-%02d", year, month)
			log.Printf("Fetching %s ...", label)

			songs, err := tj.FetchByMonth(year, month)
			if err != nil {
				log.Printf("Warning: failed to fetch %s: %v", label, err)
				continue
			}

			if len(songs) == 0 {
				log.Printf("  %s: no songs found", label)
				continue
			}

			inserted, err := repo.UpsertMany(ctx, songs)
			if err != nil {
				log.Printf("Warning: failed to upsert %s: %v", label, err)
				continue
			}

			totalInserted += inserted
			log.Printf("  %s: %d songs upserted", label, inserted)

			time.Sleep(500 * time.Millisecond) // rate limit
		}
	}

	log.Printf("Done! Total songs upserted: %d", totalInserted)
}
