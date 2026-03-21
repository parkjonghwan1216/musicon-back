package matcher

import (
	"regexp"
	"strings"

	"golang.org/x/text/unicode/norm"
)

var (
	// parenRe removes parenthesized and bracketed content: (feat. X), [MV], etc.
	parenRe = regexp.MustCompile(`[\(\[\{][^\)\]\}]*[\)\]\}]`)

	// noiseRe removes common noise words from music titles.
	noiseRe = regexp.MustCompile(`(?i)\b(official|m/?v|music\s*video|lyric\s*video|audio|hd|4k|lyrics?)\b`)

	// trailingPuncRe removes trailing punctuation like dashes left after noise removal.
	trailingPuncRe = regexp.MustCompile(`[\s\-]+$`)

	// multiSpaceRe collapses multiple whitespace characters into a single space.
	multiSpaceRe = regexp.MustCompile(`\s+`)
)

// Normalize prepares a string for fuzzy comparison:
// 1. Unicode NFC normalization
// 2. Lowercase
// 3. Remove parenthesized/bracketed content
// 4. Remove noise words (MV, official, etc.)
// 5. Collapse whitespace and trim
func Normalize(s string) string {
	s = norm.NFC.String(s)
	s = strings.ToLower(s)
	s = parenRe.ReplaceAllString(s, "")
	s = noiseRe.ReplaceAllString(s, "")
	s = multiSpaceRe.ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)
	return s
}

// separators lists common artist-title separators found in YouTube titles.
var separators = []string{" - ", " – ", " — ", " | "}

// ParseYouTubeTitle attempts to split a YouTube video title into artist and title.
// YouTube titles often follow the pattern "Artist - Title" or "Artist - Title (MV)".
func ParseYouTubeTitle(raw string) (artist, title string) {
	raw = strings.TrimSpace(raw)

	for _, sep := range separators {
		parts := strings.SplitN(raw, sep, 2)
		if len(parts) == 2 {
			artist = strings.TrimSpace(parts[0])
			title = strings.TrimSpace(parts[1])
			// Remove noise from title
			title = parenRe.ReplaceAllString(title, "")
			title = noiseRe.ReplaceAllString(title, "")
			title = trailingPuncRe.ReplaceAllString(title, "")
			title = strings.TrimSpace(title)
			return artist, title
		}
	}

	// Fallback: use whole string as title
	return "", raw
}
