// Package eval calculates retrieval quality metrics for RAG experiments.
package eval

type QueryCase struct {
	ID               string
	Query            string
	ExpectedDocIDs   []string
	ExpectedCites    []string
	Answer           string
	RetrievedDocIDs  []string
	RetrievedCites   []string
	TopKScores       []float64
	Answered         bool
	Rejected         bool
	LowSimilarityHit bool
}

type Metrics struct {
	Total                       int
	HitRate                     float64
	TopKRecallRate              float64
	CitationCoverageRate        float64
	NoAnswerRate                float64
	LowSimilarityRejectionRate  float64
	AverageTopScore             float64
	AnsweredWithMissingCitation int
}
