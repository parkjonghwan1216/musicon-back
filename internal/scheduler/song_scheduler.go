package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"

	"musicon-back/internal/domain"
	"musicon-back/internal/fetcher"
	"musicon-back/internal/repository"
	"musicon-back/internal/service"
)

// SongScheduler periodically fetches the latest TJ songs and matches them against reservations.
type SongScheduler struct {
	fetcher         *fetcher.TJFetcher
	songRepo        repository.SongRepository
	matchingService *service.MatchingService
	interval        time.Duration
	location        *time.Location
	stopCh          chan struct{}
}

// NewSongScheduler creates a scheduler that runs at the given interval.
func NewSongScheduler(
	fetcher *fetcher.TJFetcher,
	songRepo repository.SongRepository,
	matchingService *service.MatchingService,
	interval time.Duration,
) *SongScheduler {
	loc, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		loc = time.UTC
	}

	return &SongScheduler{
		fetcher:         fetcher,
		songRepo:        songRepo,
		matchingService: matchingService,
		interval:        interval,
		location:        loc,
		stopCh:          make(chan struct{}),
	}
}

// Start begins the scheduling loop in a goroutine.
// It runs an initial fetch immediately, then repeats at the configured interval.
func (s *SongScheduler) Start() {
	go func() {
		log.Printf("[Scheduler] Started — fetching every %s", s.interval)

		// Run immediately on startup
		s.run()

		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.run()
			case <-s.stopCh:
				log.Println("[Scheduler] Stopped")
				return
			}
		}
	}()
}

// Stop signals the scheduler to stop.
func (s *SongScheduler) Stop() {
	close(s.stopCh)
}

func (s *SongScheduler) run() {
	now := time.Now().In(s.location)
	log.Printf("[Scheduler] Fetch started at %s", now.Format("2006-01-02 15:04:05 KST"))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	songs, err := s.fetchCurrentMonth(ctx, now)
	if err != nil {
		log.Printf("[Scheduler] Fetch failed: %v", err)
		return
	}

	if len(songs) == 0 {
		log.Println("[Scheduler] No songs found for current month")
		return
	}

	inserted, err := s.songRepo.UpsertMany(ctx, songs)
	if err != nil {
		log.Printf("[Scheduler] Upsert failed: %v", err)
		return
	}

	log.Printf("[Scheduler] Upserted %d songs", inserted)

	matched, err := s.matchingService.MatchNewSongs(ctx, songs)
	if err != nil {
		log.Printf("[Scheduler] Matching failed: %v", err)
		return
	}

	log.Printf("[Scheduler] Fetch complete — %d songs upserted, %d reservations matched", inserted, matched)
}

func (s *SongScheduler) fetchCurrentMonth(ctx context.Context, now time.Time) ([]domain.Song, error) {
	year := now.Year()
	month := int(now.Month())

	label := fmt.Sprintf("%04d-%02d", year, month)
	log.Printf("[Scheduler] Fetching %s", label)

	songs, err := s.fetcher.FetchByMonth(year, month)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %w", label, err)
	}

	return songs, nil
}
