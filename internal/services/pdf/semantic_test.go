package pdf

import (
	"strings"
	"testing"
)

func TestSemanticMinimumConfidence(t *testing.T) {
	t.Setenv("JEV_MIN_CONFIDENCE", "0.88")
	if got := semanticMinimumConfidence(); got != 0.88 {
		t.Fatalf("semanticMinimumConfidence() = %v, want 0.88", got)
	}

	t.Setenv("JEV_MIN_CONFIDENCE", "2")
	if got := semanticMinimumConfidence(); got != 0.70 {
		t.Fatalf("semanticMinimumConfidence() = %v, want fallback 0.70", got)
	}
}

func TestTruncateDecisionTextNormalizesAndLimits(t *testing.T) {
	got := truncateDecisionText("  alpha\n\tbeta  ")
	if got != "alpha beta" {
		t.Fatalf("truncateDecisionText() = %q", got)
	}

	got = truncateDecisionText(strings.Repeat("ä", 301))
	if len([]rune(got)) != 300 {
		t.Fatalf("truncateDecisionText() rune count = %d, want 300", len([]rune(got)))
	}
}

func TestSemanticRecallScoreUsesAllCandidateFields(t *testing.T) {
	got := semanticRecallScore("Shure QLXD receiver", "Wireless receiver", "Shure", "QLXD")
	if got < 99 {
		t.Fatalf("semanticRecallScore() = %v, want full token recall", got)
	}
}
