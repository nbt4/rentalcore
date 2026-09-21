package pdf

import (
	"os"
	"strconv"
	"strings"
)

const (
	semanticNoMatch        = "no_match"
	semanticCandidateLimit = 60
)

func semanticMinimumConfidence() float64 {
	const fallback = 0.70
	value, err := strconv.ParseFloat(strings.TrimSpace(os.Getenv("JEV_MIN_CONFIDENCE")), 64)
	if err != nil || value < 0 || value > 1 {
		return fallback
	}
	return value
}

func truncateDecisionText(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	const maxRunes = 300
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes])
}

func semanticRecallScore(needle string, values ...string) float64 {
	needle = normalizeProductText(needle)
	if needle == "" {
		return 0
	}
	best := 0.0
	candidateTokens := make(map[string]bool)
	for _, value := range values {
		normalized := normalizeProductText(value)
		if score := calculateSimilarity(needle, normalized); score > best {
			best = score
		}
		for _, token := range strings.Fields(normalized) {
			candidateTokens[token] = true
		}
	}
	needleTokens := strings.Fields(needle)
	matched := 0
	for _, token := range needleTokens {
		if candidateTokens[token] {
			matched++
		}
	}
	if len(needleTokens) > 0 {
		coverage := float64(matched) / float64(len(needleTokens)) * 100
		if coverage > best {
			best = coverage
		}
	}
	return best
}
