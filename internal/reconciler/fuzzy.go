package reconciler

import (
	"math"
	"strings"
	"unicode/utf8"
)

// DamerauLevenshtein computes the edit distance between two strings,
// counting insertions, deletions, substitutions, and adjacent transpositions.
func DamerauLevenshtein(s1, s2 string) int {
	r1, r2 := []rune(s1), []rune(s2)
	len1, len2 := len(r1), len(r2)

	if len1 == 0 {
		return len2
	}
	if len2 == 0 {
		return len1
	}

	// 2D matrix for dynamic programming
	d := make([][]int, len1+1)
	for i := range d {
		d[i] = make([]int, len2+1)
	}

	for i := 0; i <= len1; i++ {
		d[i][0] = i
	}
	for j := 0; j <= len2; j++ {
		d[0][j] = j
	}

	for i := 1; i <= len1; i++ {
		for j := 1; j <= len2; j++ {
			cost := 1
			if r1[i-1] == r2[j-1] {
				cost = 0
			}

			d[i][j] = min(
				d[i-1][j]+1,      // deletion
				min(
					d[i][j-1]+1,  // insertion
					d[i-1][j-1]+cost, // substitution
				),
			)

			// Adjacent transposition
			if i > 1 && j > 1 && r1[i-1] == r2[j-2] && r1[i-2] == r2[j-1] {
				d[i][j] = min(d[i][j], d[i-2][j-2]+cost)
			}
		}
	}

	return d[len1][len2]
}

// LevenshteinSimilarity returns a normalized score between 0.0 (completely different) and 1.0 (identical).
func LevenshteinSimilarity(s1, s2 string) float64 {
	r1Len := utf8.RuneCountInString(s1)
	r2Len := utf8.RuneCountInString(s2)
	maxLen := max(r1Len, r2Len)
	if maxLen == 0 {
		return 1.0
	}

	dist := DamerauLevenshtein(s1, s2)
	return 1.0 - (float64(dist) / float64(maxLen))
}

// JaroWinkler calculates the Jaro-Winkler similarity score (0.0 to 1.0) between two strings.
// It applies a boost for matching prefixes up to 4 characters.
func JaroWinkler(s1, s2 string) float64 {
	r1, r2 := []rune(s1), []rune(s2)
	len1, len2 := len(r1), len(r2)

	if len1 == 0 && len2 == 0 {
		return 1.0
	}
	if len1 == 0 || len2 == 0 {
		return 0.0
	}

	matchWindow := max(len1, len2)/2 - 1
	if matchWindow < 0 {
		matchWindow = 0
	}

	matched1 := make([]bool, len1)
	matched2 := make([]bool, len2)
	matches := 0

	for i := 0; i < len1; i++ {
		start := max(0, i-matchWindow)
		end := min(len2-1, i+matchWindow)

		for j := start; j <= end; j++ {
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

	// Calculate transpositions
	transpositions := 0
	k := 0
	for i := 0; i < len1; i++ {
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
	jaro := (m/float64(len1) + m/float64(len2) + (m-float64(transpositions/2))/m) / 3.0

	// Common prefix boost (Winkler modification, up to 4 characters)
	prefixLen := 0
	maxPrefix := min(min(len1, len2), 4)
	for i := 0; i < maxPrefix; i++ {
		if r1[i] == r2[i] {
			prefixLen++
		} else {
			break
		}
	}

	return jaro + float64(prefixLen)*0.1*(1.0-jaro)
}

// MatchResult captures the best fuzzy match candidate.
type MatchResult struct {
	CanonicalKey string
	Score        float64
	Distance     int
	Matched      bool
}

// FindBestFuzzyMatch finds the closest candidate from a list of schema keys.
func FindBestFuzzyMatch(input string, candidates []string, minSimilarity float64, maxDistance int) MatchResult {
	if len(candidates) == 0 {
		return MatchResult{}
	}

	best := MatchResult{Score: -1.0}
	inputLower := strings.ToLower(input)
	inputFolded := FoldKey(input)

	for _, cand := range candidates {
		candLower := strings.ToLower(cand)
		candFolded := FoldKey(cand)

		// 1. Exact folded match (e.g. user_id -> userid -> userId)
		if inputFolded == candFolded {
			return MatchResult{
				CanonicalKey: cand,
				Score:        1.0,
				Distance:     0,
				Matched:      true,
			}
		}

		// 2. Compute Levenshtein distance on folded and lowercase strings
		dist := DamerauLevenshtein(inputLower, candLower)
		foldedDist := DamerauLevenshtein(inputFolded, candFolded)
		effectiveDist := min(dist, foldedDist)

		simLev := LevenshteinSimilarity(inputLower, candLower)
		simJW := JaroWinkler(inputLower, candLower)
		// Combined blended score
		score := (simLev * 0.5) + (simJW * 0.5)

		if effectiveDist <= maxDistance || score >= minSimilarity {
			if score > best.Score {
				best = MatchResult{
					CanonicalKey: cand,
					Score:        math.Round(score*1000) / 1000,
					Distance:     effectiveDist,
					Matched:      true,
				}
			}
		}
	}

	return best
}
