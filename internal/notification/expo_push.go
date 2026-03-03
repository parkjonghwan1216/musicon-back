package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"musicon-back/internal/domain"
)

const (
	expoAPIURL       = "https://exp.host/--/api/v2/push/send"
	maxBatchSize     = 100
	requestTimeout   = 30 * time.Second
)

// NotificationSender sends push notifications for matched reservations.
type NotificationSender interface {
	SendBatch(ctx context.Context, matches []domain.MatchResult) error
}

type expoPushMessage struct {
	To    string `json:"to"`
	Title string `json:"title"`
	Body  string `json:"body"`
	Sound string `json:"sound,omitempty"`
}

// ExpoPushService sends push notifications via the Expo Push API.
type ExpoPushService struct {
	httpClient *http.Client
	apiURL     string
}

func NewExpoPushService() *ExpoPushService {
	return &ExpoPushService{
		httpClient: &http.Client{Timeout: requestTimeout},
		apiURL:     expoAPIURL,
	}
}

func (s *ExpoPushService) SendBatch(ctx context.Context, matches []domain.MatchResult) error {
	if len(matches) == 0 {
		return nil
	}

	messages := make([]expoPushMessage, 0, len(matches))
	for _, m := range matches {
		messages = append(messages, expoPushMessage{
			To:    m.ExpoPushToken,
			Title: "노래방 신곡 알림",
			Body:  fmt.Sprintf("'%s' - %s가 TJ 노래방에 등록되었습니다!", m.Song.Title, m.Song.Artist),
			Sound: "default",
		})
	}

	// Send in chunks of maxBatchSize
	for i := 0; i < len(messages); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(messages) {
			end = len(messages)
		}

		chunk := messages[i:end]
		if err := s.sendChunk(ctx, chunk); err != nil {
			return fmt.Errorf("failed to send push notification chunk: %w", err)
		}

		log.Printf("Sent %d push notifications", len(chunk))
	}

	return nil
}

func (s *ExpoPushService) sendChunk(ctx context.Context, messages []expoPushMessage) error {
	body, err := json.Marshal(messages)
	if err != nil {
		return fmt.Errorf("failed to marshal push messages: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.apiURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create push request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("push request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("expo push API returned %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
