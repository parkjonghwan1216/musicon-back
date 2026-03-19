package search

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/blevesearch/bleve/v2"

	"musicon-back/internal/domain"
	"musicon-back/internal/repository"
)

const (
	batchSize = 500
	pageSize  = 1000
)

// BleveSongIndexer 는 Bleve 인덱스에 곡 데이터를 색인하는 구현체입니다.
type BleveSongIndexer struct {
	index      bleve.Index
	songRepo   repository.SongRepository
	rebuilding atomic.Bool
}

// NewBleveSongIndexer 는 새 BleveSongIndexer 를 생성합니다.
func NewBleveSongIndexer(index bleve.Index, songRepo repository.SongRepository) *BleveSongIndexer {
	return &BleveSongIndexer{
		index:    index,
		songRepo: songRepo,
	}
}

// IndexSongs 는 곡 목록을 Bleve 인덱스에 배치로 색인합니다.
// RebuildFromDB 실행 중에는 증분 인덱싱을 건너뜁니다.
func (idx *BleveSongIndexer) IndexSongs(ctx context.Context, songs []domain.Song) error {
	if len(songs) == 0 {
		return nil
	}

	if idx.rebuilding.Load() {
		log.Println("[Search] Skipping incremental index during rebuild")
		return nil
	}

	return idx.indexBatch(ctx, songs)
}

// RebuildFromDB 는 SQLite에서 모든 곡 데이터를 읽어 Bleve 인덱스를 재구축합니다.
func (idx *BleveSongIndexer) RebuildFromDB(ctx context.Context) error {
	idx.rebuilding.Store(true)
	defer idx.rebuilding.Store(false)

	log.Println("[Search] Building Bleve index...")

	totalIndexed := 0
	offset := 0

	for {
		select {
		case <-ctx.Done():
			log.Printf("[Search] Bleve index rebuild cancelled at %d documents", totalIndexed)
			return ctx.Err()
		default:
		}

		songs, err := idx.songRepo.FindAll(ctx, pageSize, offset)
		if err != nil {
			return fmt.Errorf("failed to fetch songs at offset %d: %w", offset, err)
		}

		if len(songs) == 0 {
			break
		}

		if err := idx.indexBatch(ctx, songs); err != nil {
			return fmt.Errorf("failed to index songs at offset %d: %w", offset, err)
		}

		totalIndexed += len(songs)
		offset += len(songs)

		if len(songs) < pageSize {
			break
		}
	}

	log.Printf("[Search] Bleve index built: %d documents", totalIndexed)
	return nil
}

// IsRebuilding 은 현재 인덱스 재구축 중인지 반환합니다.
func (idx *BleveSongIndexer) IsRebuilding() bool {
	return idx.rebuilding.Load()
}

// indexBatch 는 곡 목록을 배치 단위로 Bleve에 색인합니다. context 취소를 확인합니다.
func (idx *BleveSongIndexer) indexBatch(ctx context.Context, songs []domain.Song) error {
	batch := idx.index.NewBatch()
	for i, song := range songs {
		doc := toSongDocument(song)
		docID := strconv.Itoa(song.TjNumber)
		if err := batch.Index(docID, doc); err != nil {
			return fmt.Errorf("failed to add song %d to batch: %w", song.TjNumber, err)
		}

		if (i+1)%batchSize == 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if err := idx.index.Batch(batch); err != nil {
				return fmt.Errorf("failed to execute batch at %d: %w", i+1, err)
			}
			batch = idx.index.NewBatch()
		}
	}

	if batch.Size() > 0 {
		if err := idx.index.Batch(batch); err != nil {
			return fmt.Errorf("failed to execute final batch: %w", err)
		}
	}

	return nil
}

// toSongDocument 는 domain.Song 을 Bleve 인덱싱용 SongDocument 로 변환합니다.
func toSongDocument(song domain.Song) SongDocument {
	title := strings.TrimSpace(song.Title)
	artist := strings.TrimSpace(song.Artist)
	tjStr := strconv.Itoa(song.TjNumber)
	titleInitials := ExtractEnglishInitials(title)
	artistInitials := ExtractEnglishInitials(artist)
	fullText := strings.Join([]string{title, artist, tjStr}, " ")

	return SongDocument{
		TjNumber:       tjStr,
		Title:          title,
		Artist:         artist,
		TitleChosung:   strings.TrimSpace(song.TitleChosung),
		ArtistChosung:  strings.TrimSpace(song.ArtistChosung),
		TitleInitials:  titleInitials,
		ArtistInitials: artistInitials,
		FullText:       fullText,
	}
}
