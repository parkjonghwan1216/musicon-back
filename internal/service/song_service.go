package service

import (
	"context"
	"fmt"

	"musicon-back/internal/domain"
	"musicon-back/internal/repository"
)

type SongService struct {
	repo repository.SongRepository
}

func NewSongService(repo repository.SongRepository) *SongService {
	return &SongService{repo: repo}
}

func (s *SongService) Search(ctx context.Context, query string, limit, offset int) ([]domain.Song, error) {
	if query == "" {
		return nil, fmt.Errorf("search query is required")
	}

	songs, err := s.repo.Search(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to search songs: %w", err)
	}

	return songs, nil
}

func (s *SongService) FindByTjNumber(ctx context.Context, tjNumber int) (*domain.Song, error) {
	if tjNumber <= 0 {
		return nil, fmt.Errorf("invalid TJ number: %d", tjNumber)
	}

	song, err := s.repo.FindByTjNumber(ctx, tjNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to find song: %w", err)
	}

	return song, nil
}
