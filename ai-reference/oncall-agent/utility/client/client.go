// Package client 负责创建并初始化 Milvus 客户端及相关数据库资源。
package client

import (
	"context"
	"fmt"
	"oncall-agent/utility/common"

	cli "github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

func NewMilvusClient(ctx context.Context) (cli.Client, error) {
	// 1. 先连接default数据库
	defaultClient, err := cli.NewClient(ctx, cli.Config{
		Address: "localhost:19530",
		DBName:  "default",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to default database: %w", err)
	}
	// 2. 检查agent数据库是否存在，不存在则创建
	databases, err := defaultClient.ListDatabases(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list databases: %w", err)
	}
	agentDBExists := false
	for _, db := range databases {
		if db.Name == common.MilvusDBName {
			agentDBExists = true
			break
		}
	}
	if !agentDBExists {
		err = defaultClient.CreateDatabase(ctx, common.MilvusDBName)
		if err != nil {
			return nil, fmt.Errorf("failed to create agent database: %w", err)
		}
	}

	// 3. 创建连接到agent数据库的客户端
	agentClient, err := cli.NewClient(ctx, cli.Config{
		Address: "localhost:19530",
		DBName:  common.MilvusDBName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent database: %w", err)
	}
	// 4. 检查biz collection是否存在
	collections, err := agentClient.ListCollections(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	bizCollectionExists := false
	for _, collection := range collections {
		if collection.Name == common.MilvusCollectionName {
			bizCollectionExists = true
			break
		}
	}

	if !bizCollectionExists {
		// 创建biz collection的schema
		schema := &entity.Schema{
			CollectionName: common.MilvusCollectionName,
			Description:    "Business knowledge collection",
			Fields:         common.MilvusFields(),
		}

		err = agentClient.CreateCollection(ctx, schema, entity.DefaultShardNumber)
		if err != nil {
			return nil, fmt.Errorf("failed to create biz collection: %w", err)
		}

		// BinaryVector 默认使用 HAMMING 度量。
		vectorIndex, err := entity.NewIndexAUTOINDEX(entity.HAMMING)
		if err != nil {
			return nil, fmt.Errorf("failed to create vector index: %w", err)
		}
		err = agentClient.CreateIndex(ctx, common.MilvusCollectionName, common.MilvusFieldVector, vectorIndex, false)
		if err != nil {
			return nil, fmt.Errorf("failed to create vector index: %w", err)
		}
	}
	// 加载集合到内存
	fmt.Printf("[info] Loading collection: %s\n", common.MilvusCollectionName)
	err = agentClient.LoadCollection(ctx, common.MilvusCollectionName, false)
	if err != nil {
		return nil, fmt.Errorf("failed to load collection: %w", err)
	}
	fmt.Printf("[info] Collection loaded successfully\n")
	// 关闭default数据库连接
	defaultClient.Close()

	return agentClient, nil
}
