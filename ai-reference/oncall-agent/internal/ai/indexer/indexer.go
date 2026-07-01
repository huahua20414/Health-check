// Package indexer 负责创建写入 Milvus 的索引器，用于知识库向量入库。
package indexer

import (
	"context"

	embedder2 "oncall-agent/internal/ai/embedder"
	"oncall-agent/utility/client"
	"oncall-agent/utility/common"

	"github.com/cloudwego/eino-ext/components/indexer/milvus"
)

func NewMilvusIndexer(ctx context.Context) (*milvus.Indexer, error) {
	cli, err := client.NewMilvusClient(ctx)
	if err != nil {
		return nil, err
	}
	eb, err := embedder2.Embedding(ctx)
	if err != nil {
		return nil, err
	}
	config := &milvus.IndexerConfig{
		Client:     cli,
		Collection: common.MilvusCollectionName,
		Fields:     common.MilvusFields(),
		Embedding:  eb,
	}
	indexer, err := milvus.NewIndexer(ctx, config)
	if err != nil {
		return nil, err
	}
	return indexer, nil
}
