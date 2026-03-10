package matcher

import "testing"

func TestNormalize(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "lowercase",
			input: "Hello World",
			want:  "hello world",
		},
		{
			name:  "remove parenthesized content",
			input: "사랑 (feat. 아이유)",
			want:  "사랑",
		},
		{
			name:  "remove bracketed content",
			input: "밤편지 [MV]",
			want:  "밤편지",
		},
		{
			name:  "remove noise words",
			input: "좋은 날 Official MV",
			want:  "좋은 날",
		},
		{
			name:  "collapse whitespace",
			input: "  hello   world  ",
			want:  "hello world",
		},
		{
			name:  "combined",
			input: "Love Dive (Official Music Video) [4K]",
			want:  "love dive",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Normalize(tt.input)
			if got != tt.want {
				t.Errorf("Normalize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseYouTubeTitle(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantArtist string
		wantTitle  string
	}{
		{
			name:       "standard format",
			input:      "IU - Blueming",
			wantArtist: "IU",
			wantTitle:  "Blueming",
		},
		{
			name:       "with MV suffix",
			input:      "아이유 - 밤편지 (Official MV)",
			wantArtist: "아이유",
			wantTitle:  "밤편지",
		},
		{
			name:       "no separator",
			input:      "Love Dive",
			wantArtist: "",
			wantTitle:  "Love Dive",
		},
		{
			name:       "multiple dashes uses first split",
			input:      "BTS - Boy With Luv - Official",
			wantArtist: "BTS",
			wantTitle:  "Boy With Luv",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotArtist, gotTitle := ParseYouTubeTitle(tt.input)
			if gotArtist != tt.wantArtist {
				t.Errorf("ParseYouTubeTitle(%q) artist = %q, want %q", tt.input, gotArtist, tt.wantArtist)
			}
			if gotTitle != tt.wantTitle {
				t.Errorf("ParseYouTubeTitle(%q) title = %q, want %q", tt.input, gotTitle, tt.wantTitle)
			}
		})
	}
}
