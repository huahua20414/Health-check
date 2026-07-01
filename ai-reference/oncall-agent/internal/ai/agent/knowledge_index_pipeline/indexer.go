// Package knowledge_index_pipeline 中的本文件负责初始化向量索引写入节点。
package knowledge_index_pipeline

import (
	"context"
	indexer2 "oncall-agent/internal/ai/indexer"

	"github.com/cloudwego/eino/components/indexer"
)

// newIndexer component initialization function of node 'RedisIndexer' in graph 'KnowledgeIndexing'
func newIndexer(ctx context.Context) (idr indexer.Indexer, err error) {
	return indexer2.NewMilvusIndexer(ctx)
}
