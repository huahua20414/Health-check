package retriever

import (
	"context"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
)

type RetrievalConfig struct {
	FinalTopK              int
	CoarseRecallMultiplier int
	ScoreThreshold         float64
	MetadataFilter         string
}

func loadRetrievalConfig(ctx context.Context) RetrievalConfig {
	cfg := RetrievalConfig{
		FinalTopK:              defaultFinalTopK,
		CoarseRecallMultiplier: coarseRecallMultiplier,
	}
	if value, err := g.Cfg().Get(ctx, "retriever.final_top_k"); err == nil && value.Int() > 0 {
		cfg.FinalTopK = value.Int()
	}
	if value, err := g.Cfg().Get(ctx, "retriever.coarse_recall_multiplier"); err == nil && value.Int() > 0 {
		cfg.CoarseRecallMultiplier = value.Int()
	}
	if value, err := g.Cfg().Get(ctx, "retriever.score_threshold"); err == nil && value.Float64() > 0 {
		cfg.ScoreThreshold = value.Float64()
	}
	if value, err := g.Cfg().Get(ctx, "retriever.metadata_filter"); err == nil {
		cfg.MetadataFilter = strings.TrimSpace(value.String())
	}
	return cfg
}
