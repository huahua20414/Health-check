// Package knowledge_index_pipeline 中的本文件负责暴露索引流程使用的向量模型初始化逻辑。
package knowledge_index_pipeline

import (
	"context"

	"oncall-agent/internal/ai/embedder"

	"github.com/cloudwego/eino/components/embedding"
)

func newEmbedding(ctx context.Context) (eb embedding.Embedder, err error) {
	return embedder.Embedding(ctx)
}
