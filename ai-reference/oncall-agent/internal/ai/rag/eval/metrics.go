package eval

func Calculate(cases []QueryCase) Metrics {
	metrics := Metrics{Total: len(cases)}
	if len(cases) == 0 {
		return metrics
	}

	var hitCount int
	var recallSum float64
	var citationSum float64
	var noAnswerCount int
	var lowSimilarityCount int
	var lowSimilarityRejected int
	var topScoreSum float64
	var topScoreCount int

	for _, item := range cases {
		expectedDocs := toSet(item.ExpectedDocIDs)
		retrievedDocs := toSet(item.RetrievedDocIDs)
		if intersects(expectedDocs, retrievedDocs) {
			hitCount++
		}
		recallSum += coverage(expectedDocs, retrievedDocs)

		expectedCites := toSet(item.ExpectedCites)
		retrievedCites := toSet(item.RetrievedCites)
		citationCoverage := coverage(expectedCites, retrievedCites)
		citationSum += citationCoverage
		if item.Answered && len(expectedCites) > 0 && citationCoverage < 1 {
			metrics.AnsweredWithMissingCitation++
		}

		if !item.Answered {
			noAnswerCount++
		}
		if item.LowSimilarityHit {
			lowSimilarityCount++
			if item.Rejected {
				lowSimilarityRejected++
			}
		}
		if len(item.TopKScores) > 0 {
			topScoreSum += item.TopKScores[0]
			topScoreCount++
		}
	}

	metrics.HitRate = ratio(hitCount, len(cases))
	metrics.TopKRecallRate = recallSum / float64(len(cases))
	metrics.CitationCoverageRate = citationSum / float64(len(cases))
	metrics.NoAnswerRate = ratio(noAnswerCount, len(cases))
	metrics.LowSimilarityRejectionRate = ratio(lowSimilarityRejected, lowSimilarityCount)
	if topScoreCount > 0 {
		metrics.AverageTopScore = topScoreSum / float64(topScoreCount)
	}
	return metrics
}

func toSet(values []string) map[string]bool {
	result := make(map[string]bool, len(values))
	for _, value := range values {
		if value != "" {
			result[value] = true
		}
	}
	return result
}

func intersects(left, right map[string]bool) bool {
	for key := range left {
		if right[key] {
			return true
		}
	}
	return false
}

func coverage(expected, actual map[string]bool) float64 {
	if len(expected) == 0 {
		return 1
	}
	var matched int
	for key := range expected {
		if actual[key] {
			matched++
		}
	}
	return float64(matched) / float64(len(expected))
}

func ratio(numerator, denominator int) float64 {
	if denominator <= 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}
