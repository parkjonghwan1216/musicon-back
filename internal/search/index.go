package search

import (
	"fmt"
	"log"
	"os"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"

	// Bleve 분석기 패키지 등록 (각 패키지의 init()에서 registry에 등록됨)
	_ "github.com/blevesearch/bleve/v2/analysis/analyzer/custom"
	_ "github.com/blevesearch/bleve/v2/analysis/token/edgengram"
	_ "github.com/blevesearch/bleve/v2/analysis/token/lowercase"
	_ "github.com/blevesearch/bleve/v2/analysis/token/unicodenorm"
	_ "github.com/blevesearch/bleve/v2/analysis/tokenizer/unicode"
)

// SongDocument 는 Bleve 인덱스에 저장되는 곡 문서 구조입니다.
type SongDocument struct {
	TjNumber       string `json:"tj_number"`
	Title          string `json:"title"`
	Artist         string `json:"artist"`
	TitleChosung   string `json:"title_chosung"`
	ArtistChosung  string `json:"artist_chosung"`
	TitleInitials  string `json:"title_initials"`
	ArtistInitials string `json:"artist_initials"`
	FullText       string `json:"full_text"`
}

// buildIndexMapping 은 곡 검색을 위한 Bleve 인덱스 매핑을 생성합니다.
func buildIndexMapping() (mapping.IndexMapping, error) {
	indexMapping := bleve.NewIndexMapping()

	// 커스텀 토큰 필터: NFKC 유니코드 정규화
	err := indexMapping.AddCustomTokenFilter(tokenFilterNFKC, map[string]interface{}{
		"type": "normalize_unicode",
		"form": "nfkc",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to add NFKC token filter: %w", err)
	}

	// 커스텀 토큰 필터: Edge N-gram (1~4)
	err = indexMapping.AddCustomTokenFilter(tokenFilterEdgeNgram14, map[string]interface{}{
		"type": "edge_ngram",
		"min":  1.0,
		"max":  4.0,
		"side": "front",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to add edge ngram token filter: %w", err)
	}

	// 커스텀 분석기: koCJKEdgeNgram (한국어/CJK용)
	// unicode tokenizer → NFKC 정규화 → 소문자 변환 → edge ngram
	err = indexMapping.AddCustomAnalyzer(AnalyzerKoCJKEdgeNgram, map[string]interface{}{
		"type":          "custom",
		"tokenizer":     "unicode",
		"token_filters": []interface{}{tokenFilterNFKC, "to_lower", tokenFilterEdgeNgram14},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to add koCJKEdgeNgram analyzer: %w", err)
	}

	indexMapping.DefaultAnalyzer = AnalyzerKoCJKEdgeNgram

	songMapping := bleve.NewDocumentMapping()
	songMapping.Dynamic = false

	// tj_number: keyword, 저장 (결과에서 TJ 번호로 DB 조회에 사용)
	tjField := bleve.NewTextFieldMapping()
	tjField.Analyzer = "keyword"
	tjField.Store = true
	songMapping.AddFieldMappingsAt("tj_number", tjField)

	// title: koCJKEdgeNgram 분석, 저장하지 않음
	titleField := bleve.NewTextFieldMapping()
	titleField.Analyzer = AnalyzerKoCJKEdgeNgram
	titleField.Store = false
	songMapping.AddFieldMappingsAt("title", titleField)

	// artist: koCJKEdgeNgram 분석, 저장하지 않음
	artistField := bleve.NewTextFieldMapping()
	artistField.Analyzer = AnalyzerKoCJKEdgeNgram
	artistField.Store = false
	songMapping.AddFieldMappingsAt("artist", artistField)

	// title_chosung: keyword, 저장하지 않음
	titleChosungField := bleve.NewTextFieldMapping()
	titleChosungField.Analyzer = "keyword"
	titleChosungField.Store = false
	songMapping.AddFieldMappingsAt("title_chosung", titleChosungField)

	// artist_chosung: keyword, 저장하지 않음
	artistChosungField := bleve.NewTextFieldMapping()
	artistChosungField.Analyzer = "keyword"
	artistChosungField.Store = false
	songMapping.AddFieldMappingsAt("artist_chosung", artistChosungField)

	// title_initials: keyword, 저장하지 않음
	titleInitialsField := bleve.NewTextFieldMapping()
	titleInitialsField.Analyzer = "keyword"
	titleInitialsField.Store = false
	songMapping.AddFieldMappingsAt("title_initials", titleInitialsField)

	// artist_initials: keyword, 저장하지 않음
	artistInitialsField := bleve.NewTextFieldMapping()
	artistInitialsField.Analyzer = "keyword"
	artistInitialsField.Store = false
	songMapping.AddFieldMappingsAt("artist_initials", artistInitialsField)

	// full_text: koCJKEdgeNgram 분석, 저장하지 않음
	fullTextField := bleve.NewTextFieldMapping()
	fullTextField.Analyzer = AnalyzerKoCJKEdgeNgram
	fullTextField.Store = false
	songMapping.AddFieldMappingsAt("full_text", fullTextField)

	indexMapping.AddDocumentMapping("song", songMapping)
	indexMapping.DefaultMapping = songMapping

	return indexMapping, nil
}

// OpenOrCreateIndex 는 기존 Bleve 인덱스를 열거나 새로 생성합니다.
func OpenOrCreateIndex(path string) (bleve.Index, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		indexMapping, err := buildIndexMapping()
		if err != nil {
			return nil, fmt.Errorf("failed to build index mapping: %w", err)
		}
		idx, err := bleve.New(path, indexMapping)
		if err != nil {
			return nil, fmt.Errorf("failed to create bleve index: %w", err)
		}
		return idx, nil
	}

	idx, err := bleve.Open(path)
	if err != nil {
		log.Printf("[Search] Failed to open existing index (%v), recreating...", err)
		if removeErr := os.RemoveAll(path); removeErr != nil {
			return nil, fmt.Errorf("failed to remove corrupted index (original: %v): %w", err, removeErr)
		}
		indexMapping, mapErr := buildIndexMapping()
		if mapErr != nil {
			return nil, fmt.Errorf("failed to build index mapping: %w", mapErr)
		}
		idx, err = bleve.New(path, indexMapping)
		if err != nil {
			return nil, fmt.Errorf("failed to recreate bleve index: %w", err)
		}
	}

	return idx, nil
}
