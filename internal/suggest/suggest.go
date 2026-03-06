package suggest

import (
	"fmt"
	"strings"
)

// DidYouMeanSuffix returns a user-facing suggestion suffix like:
// `. Did you mean "title"?`
// It returns an empty string when no close candidate exists.
func DidYouMeanSuffix(value string, candidates []string) string {
	suggestion, ok := Closest(value, candidates)
	if !ok {
		return ""
	}
	return fmt.Sprintf(". Did you mean %q?", suggestion)
}

// Closest finds the closest candidate by Levenshtein distance.
// It returns false when no candidate is close enough to be useful.
func Closest(value string, candidates []string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" || len(candidates) == 0 {
		return "", false
	}

	valueNorm := strings.ToLower(value)
	best := ""
	bestNorm := ""
	bestDist := -1

	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		candidateNorm := strings.ToLower(candidate)
		if candidateNorm == valueNorm {
			return "", false
		}
		dist := levenshtein(valueNorm, candidateNorm)
		if bestDist == -1 || dist < bestDist || (dist == bestDist && candidateNorm < bestNorm) {
			best = candidate
			bestNorm = candidateNorm
			bestDist = dist
		}
	}

	if best == "" {
		return "", false
	}
	if bestDist <= maxSuggestionDistance(valueNorm, bestNorm) {
		return best, true
	}
	return "", false
}

func maxSuggestionDistance(a, b string) int {
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	switch {
	case maxLen <= 4:
		return 1
	case maxLen <= 8:
		return 2
	case maxLen <= 12:
		return 3
	default:
		return 4
	}
}

func levenshtein(a, b string) int {
	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Dynamic programming with O(min(len(a), len(b))) memory.
	if len(a) < len(b) {
		return levenshteinInternal(a, b)
	}
	return levenshteinInternal(b, a)
}

func levenshteinInternal(shorter, longer string) int {
	prev := make([]int, len(shorter)+1)
	curr := make([]int, len(shorter)+1)

	for i := 0; i <= len(shorter); i++ {
		prev[i] = i
	}

	for j := 1; j <= len(longer); j++ {
		curr[0] = j
		for i := 1; i <= len(shorter); i++ {
			cost := 0
			if shorter[i-1] != longer[j-1] {
				cost = 1
			}
			deletion := prev[i] + 1
			insertion := curr[i-1] + 1
			substitution := prev[i-1] + cost
			curr[i] = min3(deletion, insertion, substitution)
		}
		prev, curr = curr, prev
	}

	return prev[len(shorter)]
}

func min3(a, b, c int) int {
	if a <= b && a <= c {
		return a
	}
	if b <= c {
		return b
	}
	return c
}
