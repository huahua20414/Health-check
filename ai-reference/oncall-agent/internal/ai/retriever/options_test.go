package retriever

import (
	"testing"

	einoRetriever "github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
)

func TestResolveConfiguredTopK(t *testing.T) {
	if got := resolveFinalTopKWithDefault(7); got != 7 {
		t.Fatalf("default top k mismatch: got=%d want=7", got)
	}
	if got := resolveFinalTopKWithDefault(7, einoRetriever.WithTopK(3)); got != 3 {
		t.Fatalf("option top k mismatch: got=%d want=3", got)
	}
}

func TestResolveScoreThreshold(t *testing.T) {
	if got := resolveScoreThreshold(0.6); got != 0.6 {
		t.Fatalf("default threshold mismatch: got=%f", got)
	}
	if got := resolveScoreThreshold(0.6, einoRetriever.WithScoreThreshold(0.8)); got != 0.8 {
		t.Fatalf("option threshold mismatch: got=%f", got)
	}
}

func TestFilterByScoreThreshold(t *testing.T) {
	docs := []*schema.Document{
		{ID: "low", Content: "low"},
		{ID: "high", Content: "high"},
	}
	docs[0].WithScore(0.2)
	docs[1].WithScore(0.9)

	filtered := filterByScoreThreshold(docs, 0.5)
	if len(filtered) != 1 || filtered[0].ID != "high" {
		t.Fatalf("filtered docs mismatch: got=%v", filtered)
	}
}
