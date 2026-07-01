package retriever

import (
	"math"
	"testing"

	einoRetriever "github.com/cloudwego/eino/components/retriever"
)

func TestResolveTopK(t *testing.T) {
	finalTopK := resolveFinalTopK(einoRetriever.WithTopK(5))
	if finalTopK != 5 {
		t.Fatalf("resolveFinalTopK mismatch: got=%d want=5", finalTopK)
	}
	coarseTopK := resolveCoarseTopK(finalTopK)
	if coarseTopK < finalTopK {
		t.Fatalf("resolveCoarseTopK should be >= finalTopK, got=%d final=%d", coarseTopK, finalTopK)
	}
}

func TestNormalizeRewriteQuery(t *testing.T) {
	original := "mysql 告警怎么处理"
	rewritten := normalizeRewriteQuery(original, "改写后的查询：MySQL 连接告警处理步骤")
	if rewritten != "MySQL 连接告警处理步骤" {
		t.Fatalf("normalizeRewriteQuery mismatch: got=%q", rewritten)
	}
}

func TestCosineSimilarity(t *testing.T) {
	score := cosineSimilarity([]float64{1, 0}, []float64{1, 0})
	if math.Abs(score-1) > 1e-9 {
		t.Fatalf("cosineSimilarity mismatch: got=%f want=1", score)
	}
}

func TestEncodeDecodeRewriteInput(t *testing.T) {
	encoded := EncodeRewriteInput("他是怎么工作的", "user: 给我讲一下llm")
	query, history := decodeRewriteInput(encoded)
	if query != "他是怎么工作的" {
		t.Fatalf("decode query mismatch: got=%q", query)
	}
	if history != "user: 给我讲一下llm" {
		t.Fatalf("decode history mismatch: got=%q", history)
	}
}
