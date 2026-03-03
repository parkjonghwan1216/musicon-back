package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "modernc.org/sqlite"

	"musicon-back/internal/domain"
	"musicon-back/internal/fetcher"
	"musicon-back/internal/migration"
	"musicon-back/internal/notification"
	"musicon-back/internal/repository"
	"musicon-back/internal/service"
)

func main() {
	currentMonth := flag.Bool("current-month", false, "fetch only the current month (for daily cron)")
	flag.Parse()

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

	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		log.Fatalf("Failed to enable foreign keys: %v", err)
	}

	if err := migration.RunAll(db, "migrations"); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	songRepo := repository.NewSQLiteSongRepository(db)
	reservationRepo := repository.NewSQLiteReservationRepository(db)
	tj := fetcher.NewTJFetcher(tjBaseURL)
	pushService := notification.NewExpoPushService()
	matchingService := service.NewMatchingService(reservationRepo, pushService)

	ctx := context.Background()
	now := time.Now()

	var startYear, startMonth, endYear, endMonth int

	if *currentMonth {
		startYear = now.Year()
		startMonth = int(now.Month())
		endYear = startYear
		endMonth = startMonth
	} else {
		startYear = 2001
		startMonth = 1
		endYear = now.Year()
		endMonth = int(now.Month())
	}

	var totalInserted int64
	var allFetchedSongs []domain.Song

	for year := startYear; year <= endYear; year++ {
		monthStart := 1
		if year == startYear {
			monthStart = startMonth
		}
		monthEnd := 12
		if year == endYear {
			monthEnd = endMonth
		}

		for month := monthStart; month <= monthEnd; month++ {
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

			inserted, err := songRepo.UpsertMany(ctx, songs)
			if err != nil {
				log.Printf("Warning: failed to upsert %s: %v", label, err)
				continue
			}

			totalInserted += inserted
			allFetchedSongs = append(allFetchedSongs, songs...)
			log.Printf("  %s: %d songs upserted", label, inserted)

			time.Sleep(500 * time.Millisecond) // rate limit
		}
	}

	log.Printf("Done! Total songs upserted: %d", totalInserted)

	// Run matching against fetched songs
	if len(allFetchedSongs) > 0 {
		matched, err := matchingService.MatchNewSongs(ctx, allFetchedSongs)
		if err != nil {
			log.Printf("Warning: matching failed: %v", err)
		} else {
			log.Printf("Matching complete: %d matches found", matched)
		}
	}
}
