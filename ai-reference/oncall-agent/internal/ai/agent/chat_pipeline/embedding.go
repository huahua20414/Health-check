// Package chat_pipeline 中的本文件负责暴露对话流水线可复用的向量模型初始化逻辑。
package chat_pipeline

import (
	"context"

	"oncall-agent/internal/ai/embedder"

	"github.com/cloudwego/eino/components/embedding"
)

// 获取向量检索工具
func newEmbedding(ctx context.Context) (eb embedding.Embedder, err error) {
	return embedder.Embedding(ctx)
}
