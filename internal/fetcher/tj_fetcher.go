package fetcher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"musicon-back/internal/domain"
)

type TJFetcher struct {
	baseURL    string
	httpClient *http.Client
}

func NewTJFetcher(baseURL string) *TJFetcher {
	return &TJFetcher{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type tjRequest struct {
	SearchYm string `json:"searchYm"`
}

type tjResponse struct {
	ResultCode string `json:"resultCode"`
	ResultData struct {
		ItemsTotalCount int      `json:"itemsTotalCount"`
		Items           []tjItem `json:"items"`
	} `json:"resultData"`
}

type tjItem struct {
	Pro         int    `json:"pro"`
	IndexTitle  string `json:"indexTitle"`
	IndexSong   string `json:"indexSong"`
	Word        string `json:"word"`
	Com         string `json:"com"`
	MvYn        string `json:"mv_yn"`
	PublishDate string `json:"publishdate"`
}

// FetchByMonth fetches songs published in the given year/month from the TJ API.
func (f *TJFetcher) FetchByMonth(year, month int) ([]domain.Song, error) {
	searchYm := fmt.Sprintf("%04d%02d", year, month)

	reqBody, err := json.Marshal(tjRequest{SearchYm: searchYm})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := f.httpClient.Post(f.baseURL, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var tjResp tjResponse
	if err := json.Unmarshal(body, &tjResp); err != nil {
		return nil, fmt.Errorf("failed to parse response JSON: %w", err)
	}

	items := tjResp.ResultData.Items
	songs := make([]domain.Song, 0, len(items))

	for _, item := range items {
		publishedAt, err := time.Parse("2006-01-02", item.PublishDate)
		if err != nil {
			publishedAt = time.Now()
		}

		song := domain.Song{
			TjNumber:      item.Pro,
			Title:         strings.TrimSpace(item.IndexTitle),
			Artist:        strings.TrimSpace(item.IndexSong),
			Lyricist:      strings.TrimSpace(item.Word),
			Composer:      strings.TrimSpace(item.Com),
			TitleChosung:  extractChosung(strings.TrimSpace(item.IndexTitle)),
			ArtistChosung: extractChosung(strings.TrimSpace(item.IndexSong)),
			HasMV:         item.MvYn == "Y",
			PublishedAt:   publishedAt,
		}

		songs = append(songs, song)
	}

	log.Printf("Fetched %d songs for %s", len(songs), searchYm)
	return songs, nil
}

// extractChosung extracts Korean initial consonants (초성) from a string.
func extractChosung(s string) string {
	chosungs := []rune{
		'ㄱ', 'ㄲ', 'ㄴ', 'ㄷ', 'ㄸ', 'ㄹ', 'ㅁ', 'ㅂ', 'ㅃ',
		'ㅅ', 'ㅆ', 'ㅇ', 'ㅈ', 'ㅉ', 'ㅊ', 'ㅋ', 'ㅌ', 'ㅍ', 'ㅎ',
	}

	var result strings.Builder
	for _, r := range s {
		if r >= 0xAC00 && r <= 0xD7A3 {
			idx := (r - 0xAC00) / 28 / 21
			result.WriteRune(chosungs[idx])
		} else if utf8.ValidRune(r) {
			result.WriteRune(r)
		}
	}
	return result.String()
}
