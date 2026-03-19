package search

import (
	"context"

	"musicon-back/internal/domain"
)

// SongSearcher 는 전문 검색 엔진을 통한 곡 검색 인터페이스입니다.
// 인덱스 lifecycle(Open/Close)은 main.go에서 관리합니다.
type SongSearcher interface {
	// Search 는 쿼리에 매칭되는 곡의 TJ 번호 목록을 score 순으로 반환합니다.
	Search(ctx context.Context, query string, limit, offset int) ([]int, error)
}

// SongIndexer 는 검색 인덱스에 곡 데이터를 색인하는 인터페이스입니다.
type SongIndexer interface {
	IndexSongs(ctx context.Context, songs []domain.Song) error
	RebuildFromDB(ctx context.Context) error
}
