package service

import (
	"context"
	"fmt"
	"log"

	"musicon-back/internal/domain"
	"musicon-back/internal/repository"
	"musicon-back/internal/search"
)

type SongService struct {
	repo     repository.SongRepository
	searcher search.SongSearcher // nil 허용: Bleve 미사용 시 SQL fallback
}

func NewSongService(repo repository.SongRepository, searcher search.SongSearcher) *SongService {
	return &SongService{repo: repo, searcher: searcher}
}

func (s *SongService) Search(ctx context.Context, query string, limit, offset int) ([]domain.Song, error) {
	if query == "" {
		return nil, fmt.Errorf("search query is required")
	}

	// Bleve 우선 검색
	if s.searcher != nil {
		songs, err := s.searchWithBleve(ctx, query, limit, offset)
		if err != nil {
			log.Printf("[SongService] Bleve search failed, falling back to SQL: %v", err)
		} else {
			return songs, nil
		}
	}

	// SQL fallback
	return s.repo.Search(ctx, query, limit, offset)
}

func (s *SongService) searchWithBleve(ctx context.Context, query string, limit, offset int) ([]domain.Song, error) {
	tjNumbers, err := s.searcher.Search(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}

	if len(tjNumbers) == 0 {
		return []domain.Song{}, nil
	}

	songs, err := s.repo.FindByTjNumbers(ctx, tjNumbers)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch songs by TJ numbers: %w", err)
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
