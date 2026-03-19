package search

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/blevesearch/bleve/v2"
	bleveSearch "github.com/blevesearch/bleve/v2/search"
	"github.com/blevesearch/bleve/v2/search/query"
)

// BleveSongSearcher 는 Bleve 검색 엔진 기반 곡 검색 구현체입니다.
type BleveSongSearcher struct {
	index bleve.Index
}

// NewBleveSongSearcher 는 새 BleveSongSearcher 를 생성합니다.
func NewBleveSongSearcher(index bleve.Index) *BleveSongSearcher {
	return &BleveSongSearcher{index: index}
}

// Search 는 쿼리에 매칭되는 곡의 TJ 번호 목록을 score 순으로 반환합니다.
func (s *BleveSongSearcher) Search(_ context.Context, q string, limit, offset int) ([]int, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil, fmt.Errorf("search query is required")
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	subQueries := buildSubQueries(q)
	if len(subQueries) == 0 {
		return nil, nil
	}

	disjunction := bleve.NewDisjunctionQuery(subQueries...)
	searchRequest := bleve.NewSearchRequestOptions(disjunction, limit, offset, false)
	searchRequest.Fields = []string{"tj_number"}
	searchRequest.SortBy([]string{"-_score"})

	result, err := s.index.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("bleve search failed: %w", err)
	}

	return extractTjNumbers(result.Hits), nil
}

// buildSubQueries 는 쿼리 문자열을 분석하여 DisjunctionQuery 하위 쿼리 목록을 구성합니다.
func buildSubQueries(q string) []query.Query {
	var queries []query.Query
	lowerQ := strings.ToLower(q)

	// 1. TJ 번호 정확 매칭 (숫자인 경우)
	if isNumeric(q) {
		tjQuery := bleve.NewTermQuery(q)
		tjQuery.SetField("tj_number")
		tjQuery.SetBoost(30)
		queries = append(queries, tjQuery)
	}

	// 2. 초성 검색 (한글 초성으로만 이루어진 경우)
	if IsChosungOnly(q) {
		titleChosungQ := bleve.NewTermQuery(q)
		titleChosungQ.SetField("title_chosung")
		titleChosungQ.SetBoost(20)
		queries = append(queries, titleChosungQ)

		artistChosungQ := bleve.NewTermQuery(q)
		artistChosungQ.SetField("artist_chosung")
		artistChosungQ.SetBoost(15)
		queries = append(queries, artistChosungQ)

		return queries
	}

	// 3. title MatchQuery
	titleMatchQ := bleve.NewMatchQuery(q)
	titleMatchQ.SetField("title")
	titleMatchQ.SetBoost(20)
	queries = append(queries, titleMatchQ)

	// 4. artist MatchQuery
	artistMatchQ := bleve.NewMatchQuery(q)
	artistMatchQ.SetField("artist")
	artistMatchQ.SetBoost(15)
	queries = append(queries, artistMatchQ)

	// 5. 한글이 포함된 경우 초성 추출 후 매칭
	if ContainsKorean(q) {
		chosung := ExtractChosung(q)
		if chosung != q {
			titleChosungQ := bleve.NewTermQuery(chosung)
			titleChosungQ.SetField("title_chosung")
			titleChosungQ.SetBoost(10)
			queries = append(queries, titleChosungQ)

			artistChosungQ := bleve.NewTermQuery(chosung)
			artistChosungQ.SetField("artist_chosung")
			artistChosungQ.SetBoost(8)
			queries = append(queries, artistChosungQ)
		}
	}

	// 6. 영문 이니셜 매칭 (영문만 있고 짧은 경우)
	if !ContainsKorean(q) && len(lowerQ) <= 10 {
		titleInitialsQ := bleve.NewTermQuery(lowerQ)
		titleInitialsQ.SetField("title_initials")
		titleInitialsQ.SetBoost(20)
		queries = append(queries, titleInitialsQ)

		artistInitialsQ := bleve.NewTermQuery(lowerQ)
		artistInitialsQ.SetField("artist_initials")
		artistInitialsQ.SetBoost(15)
		queries = append(queries, artistInitialsQ)
	}

	// 7. full_text MatchQuery
	fullTextQ := bleve.NewMatchQuery(q)
	fullTextQ.SetField("full_text")
	fullTextQ.SetBoost(10)
	queries = append(queries, fullTextQ)

	return queries
}

// extractTjNumbers 는 Bleve 검색 결과에서 TJ 번호 목록을 추출합니다.
func extractTjNumbers(hits bleveSearch.DocumentMatchCollection) []int {
	tjNumbers := make([]int, 0, len(hits))
	for _, hit := range hits {
		tjStr, ok := hit.Fields["tj_number"].(string)
		if !ok {
			continue
		}
		tjNum, err := strconv.Atoi(tjStr)
		if err != nil {
			continue
		}
		tjNumbers = append(tjNumbers, tjNum)
	}
	return tjNumbers
}

// isNumeric 은 문자열이 ASCII 숫자(0-9)로만 이루어져 있는지 확인합니다.
func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
