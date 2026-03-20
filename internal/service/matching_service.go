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
// Artist-only reservations (title == "") match all songs by that artist and stay active.
// Artist+title reservations match once and transition to matched status.
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

	for _, ar := range activeReservations {
		resArtist := strings.ToLower(ar.Reservation.Artist)
		resTitle := strings.ToLower(ar.Reservation.Title)

		for _, song := range newSongs {
			songArtist := strings.ToLower(song.Artist)
			songTitle := strings.ToLower(song.Title)

			artistMatch := strings.Contains(songArtist, resArtist)
			titleMatch := ar.Reservation.IsArtistOnly() || strings.Contains(songTitle, resTitle)

			if artistMatch && titleMatch {
				if ar.Reservation.IsArtistOnly() {
					notified, err := s.reservationRepo.HasNotified(ctx, ar.Reservation.ID, song.ID)
					if err != nil {
						log.Printf("Warning: failed to check notification history: %v", err)
						continue
					}
					if notified {
						continue
					}
				}

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

	// Send notifications first to avoid losing notifications on record failure
	if err := s.sender.SendBatch(ctx, matches); err != nil {
		return 0, fmt.Errorf("failed to send notifications: %w", err)
	}

	// Record/mark after successful send
	for _, m := range matches {
		if m.Reservation.IsArtistOnly() {
			if err := s.reservationRepo.RecordNotification(ctx, m.Reservation.ID, m.Song.ID); err != nil {
				log.Printf("Warning: failed to record notification for reservation %d: %v", m.Reservation.ID, err)
			}
		} else {
			if err := s.reservationRepo.MarkAsMatched(ctx, m.Reservation.ID, m.Song.ID); err != nil {
				log.Printf("Warning: failed to mark reservation %d as matched: %v", m.Reservation.ID, err)
			}
			if err := s.reservationRepo.MarkAsNotified(ctx, m.Reservation.ID); err != nil {
				log.Printf("Warning: failed to mark reservation %d as notified: %v", m.Reservation.ID, err)
			}
		}
	}

	return len(matches), nil
}
