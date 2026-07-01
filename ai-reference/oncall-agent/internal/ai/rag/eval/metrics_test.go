package eval

import "testing"

func TestCalculate(t *testing.T) {
	metrics := Calculate([]QueryCase{
		{
			ID:              "hit",
			ExpectedDocIDs:  []string{"doc-a", "doc-b"},
			RetrievedDocIDs: []string{"doc-a"},
			ExpectedCites:   []string{"doc-a"},
			RetrievedCites:  []string{"doc-a"},
			TopKScores:      []float64{0.9},
			Answered:        true,
		},
		{
			ID:               "reject",
			ExpectedDocIDs:   []string{"doc-c"},
			RetrievedDocIDs:  []string{"doc-x"},
			ExpectedCites:    []string{"doc-c"},
			TopKScores:       []float64{0.2},
			Answered:         false,
			Rejected:         true,
			LowSimilarityHit: true,
		},
	})

	if metrics.Total != 2 {
		t.Fatalf("total mismatch: got=%d", metrics.Total)
	}
	if metrics.HitRate != 0.5 {
		t.Fatalf("hit rate mismatch: got=%f", metrics.HitRate)
	}
	if metrics.TopKRecallRate != 0.25 {
		t.Fatalf("topk recall mismatch: got=%f", metrics.TopKRecallRate)
	}
	if metrics.CitationCoverageRate != 0.5 {
		t.Fatalf("citation coverage mismatch: got=%f", metrics.CitationCoverageRate)
	}
	if metrics.NoAnswerRate != 0.5 {
		t.Fatalf("no answer rate mismatch: got=%f", metrics.NoAnswerRate)
	}
	if metrics.LowSimilarityRejectionRate != 1 {
		t.Fatalf("low similarity rejection mismatch: got=%f", metrics.LowSimilarityRejectionRate)
	}
}
