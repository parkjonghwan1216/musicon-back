package matcher

import "testing"

func TestScore(t *testing.T) {
	tests := []struct {
		name        string
		trackTitle  string
		trackArtist string
		songTitle   string
		songArtist  string
		minScore    float64
	}{
		{
			name:        "exact match",
			trackTitle:  "밤편지",
			trackArtist: "아이유",
			songTitle:   "밤편지",
			songArtist:  "아이유",
			minScore:    1.0,
		},
		{
			name:        "similar match",
			trackTitle:  "blueming",
			trackArtist: "iu",
			songTitle:   "blueming",
			songArtist:  "iu",
			minScore:    1.0,
		},
		{
			name:        "completely different",
			trackTitle:  "abc",
			trackArtist: "xyz",
			songTitle:   "가나다",
			songArtist:  "라마바",
			minScore:    0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Score(tt.trackTitle, tt.trackArtist, tt.songTitle, tt.songArtist)
			if got < tt.minScore {
				t.Errorf("Score() = %f, want >= %f", got, tt.minScore)
			}
		})
	}
}

func TestBestMatch(t *testing.T) {
	candidates := []CandidateSong{
		{ID: 1, Title: "밤편지", Artist: "아이유"},
		{ID: 2, Title: "좋은 날", Artist: "아이유"},
		{ID: 3, Title: "사랑이 빠져", Artist: "다른가수"},
	}

	t.Run("finds exact match", func(t *testing.T) {
		result := BestMatch("밤편지", "아이유", candidates)
		if result == nil {
			t.Fatal("expected match, got nil")
		}
		if result.SongID != 1 {
			t.Errorf("expected song ID 1, got %d", result.SongID)
		}
		if result.Score < MinMatchScore {
			t.Errorf("score %f below threshold %f", result.Score, MinMatchScore)
		}
	})

	t.Run("no match below threshold", func(t *testing.T) {
		result := BestMatch("completely different song", "unknown artist", candidates)
		if result != nil {
			t.Errorf("expected no match, got song ID %d with score %f", result.SongID, result.Score)
		}
	})

	t.Run("empty candidates", func(t *testing.T) {
		result := BestMatch("밤편지", "아이유", nil)
		if result != nil {
			t.Error("expected nil for empty candidates")
		}
	})
}
