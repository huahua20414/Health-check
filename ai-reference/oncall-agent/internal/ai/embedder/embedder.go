// Package embedder 负责初始化项目统一使用的文本向量模型。
package embedder

import (
	"context"
	"log"

	"github.com/cloudwego/eino-ext/components/embedding/dashscope"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/gogf/gf/v2/frame/g"
)

func Embedding(ctx context.Context) (eb embedding.Embedder, err error) {
	model, err := g.Cfg().Get(ctx, "embedding_model.model")
	if err != nil {
		return nil, err
	}
	api_key, err := g.Cfg().Get(ctx, "embedding_model.api_key")
	if err != nil {
		return nil, err
	}
	dim := 1024
	embedder, err := dashscope.NewEmbedder(ctx, &dashscope.EmbeddingConfig{
		Model:      model.String(),
		APIKey:     api_key.String(),
		Dimensions: &dim,
	})
	if err != nil {
		log.Printf("new embedder error: %v\n", err)
		return nil, err
	}
	return embedder, nil
}
