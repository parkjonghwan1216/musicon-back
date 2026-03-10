package matcher

const (
	titleWeight  = 0.6
	artistWeight = 0.4
	// MinMatchScore is the minimum combined score to consider a track matched.
	MinMatchScore = 0.70
)

// Score computes a weighted similarity score between an external track and a TJ song.
// Both title and artist strings should already be normalized.
func Score(trackTitle, trackArtist, songTitle, songArtist string) float64 {
	titleScore := JaroWinkler(trackTitle, songTitle)
	artistScore := JaroWinkler(trackArtist, songArtist)
	return titleScore*titleWeight + artistScore*artistWeight
}

// ScoreResult holds the matching result for a single candidate song.
type ScoreResult struct {
	SongID int64
	Score  float64
}

// BestMatch finds the best matching song from a list of candidates.
// Returns nil if no candidate exceeds MinMatchScore.
func BestMatch(trackTitle, trackArtist string, candidates []CandidateSong) *ScoreResult {
	normTitle := Normalize(trackTitle)
	normArtist := Normalize(trackArtist)

	var best *ScoreResult

	for _, c := range candidates {
		score := Score(normTitle, normArtist, Normalize(c.Title), Normalize(c.Artist))
		if score >= MinMatchScore && (best == nil || score > best.Score) {
			best = &ScoreResult{
				SongID: c.ID,
				Score:  score,
			}
		}
	}

	return best
}

// CandidateSong represents a TJ song candidate for matching.
type CandidateSong struct {
	ID     int64
	Title  string
	Artist string
}
