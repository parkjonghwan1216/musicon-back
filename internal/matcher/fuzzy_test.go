package matcher

import (
	"math"
	"testing"
)

func TestJaroWinkler(t *testing.T) {
	tests := []struct {
		name string
		s1   string
		s2   string
		min  float64
		max  float64
	}{
		{
			name: "identical strings",
			s1:   "hello", s2: "hello",
			min: 1.0, max: 1.0,
		},
		{
			name: "completely different",
			s1:   "abc", s2: "xyz",
			min: 0.0, max: 0.01,
		},
		{
			name: "similar strings",
			s1:   "martha", s2: "marhta",
			min: 0.96, max: 1.0,
		},
		{
			name: "empty strings",
			s1:   "", s2: "",
			min: 1.0, max: 1.0,
		},
		{
			name: "one empty",
			s1:   "hello", s2: "",
			min: 0.0, max: 0.01,
		},
		{
			name: "korean similar",
			s1:   "사랑하는 그대에게",
			s2:   "사랑하는 그대",
			min: 0.85, max: 1.0,
		},
		{
			name: "korean exact",
			s1:   "밤편지",
			s2:   "밤편지",
			min: 1.0, max: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JaroWinkler(tt.s1, tt.s2)
			if got < tt.min || got > tt.max {
				t.Errorf("JaroWinkler(%q, %q) = %f, want [%f, %f]", tt.s1, tt.s2, got, tt.min, tt.max)
			}
		})
	}
}

func TestJaroWinklerSymmetry(t *testing.T) {
	pairs := [][2]string{
		{"hello", "helo"},
		{"사랑", "사량"},
		{"abc", "def"},
	}

	for _, pair := range pairs {
		a := JaroWinkler(pair[0], pair[1])
		b := JaroWinkler(pair[1], pair[0])
		if math.Abs(a-b) > 0.001 {
			t.Errorf("JaroWinkler not symmetric: (%q,%q)=%f vs (%q,%q)=%f", pair[0], pair[1], a, pair[1], pair[0], b)
		}
	}
}
