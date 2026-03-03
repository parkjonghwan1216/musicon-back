package service

import (
	"context"
	"fmt"
	"log"
	"strings"

	"musicon-back/internal/domain"
	"musicon-back/internal/notification"
	"musicon-back/internal/repository"
)

type MatchingService struct {
	reservationRepo repository.ReservationRepository
	sender          notification.NotificationSender
}

func NewMatchingService(
	reservationRepo repository.ReservationRepository,
	sender notification.NotificationSender,
) *MatchingService {
	return &MatchingService{
		reservationRepo: reservationRepo,
		sender:          sender,
	}
}

// MatchNewSongs checks new songs against active reservations using partial matching.
// Returns the number of matches found.
func (s *MatchingService) MatchNewSongs(ctx context.Context, newSongs []domain.Song) (int, error) {
	activeReservations, err := s.reservationRepo.FindActiveWithTokens(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to find active reservations: %w", err)
	}

	if len(activeReservations) == 0 {
		log.Println("No active reservations to match")
		return 0, nil
	}

	log.Printf("Matching %d songs against %d active reservations", len(newSongs), len(activeReservations))

	var matches []domain.MatchResult

	for _, song := range newSongs {
		songArtist := strings.ToLower(song.Artist)
		songTitle := strings.ToLower(song.Title)

		for _, ar := range activeReservations {
			resArtist := strings.ToLower(ar.Reservation.Artist)
			resTitle := strings.ToLower(ar.Reservation.Title)

			if strings.Contains(songArtist, resArtist) && strings.Contains(songTitle, resTitle) {
				matches = append(matches, domain.MatchResult{
					Reservation:   ar.Reservation,
					Song:          song,
					ExpoPushToken: ar.ExpoPushToken,
				})
			}
		}
	}

	if len(matches) == 0 {
		log.Println("No matches found")
		return 0, nil
	}

	log.Printf("Found %d matches, sending notifications", len(matches))

	// Mark as matched
	for _, m := range matches {
		if err := s.reservationRepo.MarkAsMatched(ctx, m.Reservation.ID, m.Song.ID); err != nil {
			log.Printf("Warning: failed to mark reservation %d as matched: %v", m.Reservation.ID, err)
		}
	}

	// Send notifications
	if err := s.sender.SendBatch(ctx, matches); err != nil {
		return len(matches), fmt.Errorf("failed to send notifications: %w", err)
	}

	// Mark as notified
	for _, m := range matches {
		if err := s.reservationRepo.MarkAsNotified(ctx, m.Reservation.ID); err != nil {
			log.Printf("Warning: failed to mark reservation %d as notified: %v", m.Reservation.ID, err)
		}
	}

	return len(matches), nil
}
