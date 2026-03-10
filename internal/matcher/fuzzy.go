package matcher

// JaroWinkler computes the Jaro-Winkler similarity between two strings.
// Returns a value between 0.0 (no similarity) and 1.0 (exact match).
func JaroWinkler(s1, s2 string) float64 {
	jaro := jaroSimilarity(s1, s2)
	if jaro == 0 {
		return 0
	}

	// Winkler modification: boost for common prefix (up to 4 chars)
	r1 := []rune(s1)
	r2 := []rune(s2)

	prefixLen := 0
	maxPrefix := 4
	if len(r1) < maxPrefix {
		maxPrefix = len(r1)
	}
	if len(r2) < maxPrefix {
		maxPrefix = len(r2)
	}
	for i := 0; i < maxPrefix; i++ {
		if r1[i] == r2[i] {
			prefixLen++
		} else {
			break
		}
	}

	const scalingFactor = 0.1
	return jaro + float64(prefixLen)*scalingFactor*(1.0-jaro)
}

func jaroSimilarity(s1, s2 string) float64 {
	r1 := []rune(s1)
	r2 := []rune(s2)

	if len(r1) == 0 && len(r2) == 0 {
		return 1.0
	}
	if len(r1) == 0 || len(r2) == 0 {
		return 0.0
	}

	maxDist := len(r1)
	if len(r2) > maxDist {
		maxDist = len(r2)
	}
	matchWindow := maxDist/2 - 1
	if matchWindow < 0 {
		matchWindow = 0
	}

	matched1 := make([]bool, len(r1))
	matched2 := make([]bool, len(r2))

	matches := 0
	transpositions := 0

	for i := 0; i < len(r1); i++ {
		lo := i - matchWindow
		if lo < 0 {
			lo = 0
		}
		hi := i + matchWindow + 1
		if hi > len(r2) {
			hi = len(r2)
		}

		for j := lo; j < hi; j++ {
			if matched2[j] || r1[i] != r2[j] {
				continue
			}
			matched1[i] = true
			matched2[j] = true
			matches++
			break
		}
	}

	if matches == 0 {
		return 0.0
	}

	k := 0
	for i := 0; i < len(r1); i++ {
		if !matched1[i] {
			continue
		}
		for !matched2[k] {
			k++
		}
		if r1[i] != r2[k] {
			transpositions++
		}
		k++
	}

	m := float64(matches)
	return (m/float64(len(r1)) + m/float64(len(r2)) + (m-float64(transpositions)/2.0)/m) / 3.0
}
