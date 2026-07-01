package mem

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	embedder2 "oncall-agent/internal/ai/embedder"
	agentclient "oncall-agent/utility/client"
	"oncall-agent/utility/common"

	"github.com/cloudwego/eino/components/embedding"
	milvusclient "github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

const (
	// 这些常量集中定义 memory collection 的字段名，避免字符串散落在代码里。
	memoryCollectionIDField        = "id"
	memoryCollectionSessionIDField = "session_id"
	memoryCollectionVectorField    = "vector"
	memoryCollectionContentField   = "content"
	memoryCollectionMetadataField  = "metadata"
)

type semanticMemoryStore struct {
	client   milvusclient.Client
	embedder embedding.Embedder
	// 这张 collection 专门存“会话长期事实”，避免和业务知识库 biz collection 混在一起。
	collection string
	topK       int
}

func newSemanticMemoryStore(ctx context.Context, collection string, topK int) (*semanticMemoryStore, error) {
	// 复用项目现有的 Milvus client 创建逻辑。
	cli, err := agentclient.NewMilvusClient(ctx)
	if err != nil {
		return nil, err
	}

	// 语义记忆和知识库检索共用同一个 embedding 模型，
	// 这样 query 和 facts 才处在同一个向量空间里。
	eb, err := embedder2.Embedding(ctx)
	if err != nil {
		return nil, err
	}

	// 构造完成后立刻确保 collection 存在并已 load，
	// 这样后面的读写路径不需要每次再检查一遍。
	store := &semanticMemoryStore{
		client:     cli,
		embedder:   eb,
		collection: collection,
		topK:       topK,
	}
	if err := store.ensureCollection(ctx); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *semanticMemoryStore) ensureCollection(ctx context.Context) error {
	// 先看 collection 是否已存在；如果已经存在就不重复建。
	ok, err := s.client.HasCollection(ctx, s.collection)
	if err != nil {
		return fmt.Errorf("check memory collection failed: %w", err)
	}
	if !ok {
		// 这里使用 float vector，而不是项目原来知识库里的 binary vector。
		// 原因是当前 embedding 模型返回的是 float 向量，直接按语义相似度检索更自然。
		schema := entity.NewSchema().
			WithName(s.collection).
			WithDescription("Long-term memory facts for chat sessions").
			WithField(entity.NewField().
				WithName(memoryCollectionIDField).
				WithDataType(entity.FieldTypeVarChar).
				WithIsPrimaryKey(true).
				WithMaxLength(256)).
			WithField(entity.NewField().
				WithName(memoryCollectionSessionIDField).
				WithDataType(entity.FieldTypeVarChar).
				WithMaxLength(128)).
			WithField(entity.NewField().
				WithName(memoryCollectionVectorField).
				WithDataType(entity.FieldTypeFloatVector).
				WithDim(common.MilvusEmbeddingDim)).
			WithField(entity.NewField().
				WithName(memoryCollectionContentField).
				WithDataType(entity.FieldTypeVarChar).
				WithMaxLength(4096)).
			WithField(entity.NewField().
				WithName(memoryCollectionMetadataField).
				WithDataType(entity.FieldTypeJSON))

		if err := s.client.CreateCollection(ctx, schema, entity.DefaultShardNumber); err != nil {
			return fmt.Errorf("create memory collection failed: %w", err)
		}

		// 记忆检索只关心“语义接近”，所以这里直接建 COSINE 的自动索引。
		index, err := entity.NewIndexAUTOINDEX(entity.COSINE)
		if err != nil {
			return fmt.Errorf("create memory index failed: %w", err)
		}
		if err := s.client.CreateIndex(ctx, s.collection, memoryCollectionVectorField, index, false); err != nil {
			return fmt.Errorf("create memory vector index failed: %w", err)
		}
	}

	// collection 存在之后要 load 到内存，不然 search 可能直接失败。
	return s.client.LoadCollection(ctx, s.collection, false)
}

func (s *semanticMemoryStore) UpsertFacts(ctx context.Context, sessionID string, facts []memoryFact) error {
	if len(facts) == 0 {
		return nil
	}

	// 先把所有事实文本取出来，批量做 embedding，避免一条一条调用模型。
	texts := make([]string, 0, len(facts))
	for _, fact := range facts {
		texts = append(texts, fact.Content)
	}

	vectors, err := s.embedder.EmbedStrings(ctx, texts)
	if err != nil {
		return fmt.Errorf("embed memory facts failed: %w", err)
	}
	if len(vectors) != len(facts) {
		return fmt.Errorf("memory fact embedding count mismatch: got=%d want=%d", len(vectors), len(facts))
	}

	// Milvus float vector 需要 [][]float32，这里后面会把 embedding 的 []float64 转过去。
	ids := make([]string, 0, len(facts))
	sessionIDs := make([]string, 0, len(facts))
	contents := make([]string, 0, len(facts))
	metadata := make([][]byte, 0, len(facts))
	floatVectors := make([][]float32, 0, len(facts))
	for idx, fact := range facts {
		// 用 sessionID + factHash 作为主键，保证同一条事实多次同步时会覆盖更新，而不是重复插入。
		ids = append(ids, semanticFactID(sessionID, fact.Hash))
		sessionIDs = append(sessionIDs, sessionID)
		contents = append(contents, trimForMilvus(fact.Content, 4096))
		meta, err := json.Marshal(map[string]any{
			"session_id": sessionID,
			"hash":       fact.Hash,
			"source":     "memory_fact",
		})
		if err != nil {
			return fmt.Errorf("marshal memory fact metadata failed: %w", err)
		}
		metadata = append(metadata, meta)
		floatVectors = append(floatVectors, toFloat32Slice(vectors[idx]))
	}

	_, err = s.client.Upsert(
		ctx,
		s.collection,
		"",
		entity.NewColumnVarChar(memoryCollectionIDField, ids),
		entity.NewColumnVarChar(memoryCollectionSessionIDField, sessionIDs),
		entity.NewColumnFloatVector(memoryCollectionVectorField, common.MilvusEmbeddingDim, floatVectors),
		entity.NewColumnVarChar(memoryCollectionContentField, contents),
		entity.NewColumnJSONBytes(memoryCollectionMetadataField, metadata),
	)
	if err != nil {
		return fmt.Errorf("upsert memory facts failed: %w", err)
	}
	// Flush 之后这批写入才会稳定可检索。
	if err := s.client.Flush(ctx, s.collection, false); err != nil {
		return fmt.Errorf("flush memory facts failed: %w", err)
	}
	return nil
}

func (s *semanticMemoryStore) QueryFacts(ctx context.Context, sessionID, query string) ([]string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}

	// query 先向量化，后面才能和 facts 的向量做相似度搜索。
	vectors, err := s.embedder.EmbedStrings(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("embed memory query failed: %w", err)
	}
	if len(vectors) != 1 {
		return nil, fmt.Errorf("memory query embedding count mismatch: got=%d want=1", len(vectors))
	}

	// AUTOINDEX search param 是 Milvus 自动索引对应的搜索参数。
	searchParam, err := entity.NewIndexAUTOINDEXSearchParam(10)
	if err != nil {
		return nil, fmt.Errorf("create memory search param failed: %w", err)
	}

	// expr 里按 session_id 过滤，保证一个 session 只召回自己的长期记忆。
	results, err := s.client.Search(
		ctx,
		s.collection,
		nil,
		fmt.Sprintf(`%s == "%s"`, memoryCollectionSessionIDField, escapeMilvusString(sessionID)),
		[]string{memoryCollectionContentField},
		// Search 接口要求的是 entity.Vector，这里把 query 向量包装成 FloatVector。
		[]entity.Vector{entity.FloatVector(toFloat32Slice(vectors[0]))},
		memoryCollectionVectorField,
		entity.COSINE,
		s.topK,
		searchParam,
	)
	if err != nil {
		return nil, fmt.Errorf("search memory facts failed: %w", err)
	}

	// 检索结果可能有重复或相似表述，这里做一次去重后再回注到 prompt。
	seen := make(map[string]struct{}, s.topK)
	facts := make([]string, 0, s.topK)
	for _, result := range results {
		if result.Err != nil {
			return nil, fmt.Errorf("search memory facts result failed: %w", result.Err)
		}
		column := result.Fields.GetColumn(memoryCollectionContentField)
		if column == nil {
			continue
		}
		for i := 0; i < result.ResultCount; i++ {
			value, err := column.GetAsString(i)
			if err != nil {
				// 有些列实现不支持 GetAsString，这里退回通用 Get 再手动断言。
				raw, rawErr := column.Get(i)
				if rawErr != nil {
					return nil, fmt.Errorf("read memory fact failed: %w", err)
				}
				str, ok := raw.(string)
				if !ok {
					continue
				}
				value = str
			}
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			facts = append(facts, value)
		}
	}

	return facts, nil
}

func semanticFactID(sessionID, hash string) string {
	// 这个主键设计成 sessionID:hash，便于幂等 upsert。
	return sessionID + ":" + hash
}

func toFloat32Slice(vector []float64) []float32 {
	// embedding 组件返回的是 float64，Milvus float vector 列要求 float32。
	result := make([]float32, 0, len(vector))
	for _, value := range vector {
		result = append(result, float32(value))
	}
	return result
}

func escapeMilvusString(value string) string {
	// expr 里要放字符串字面量，所以先转义引号和反斜杠，避免过滤表达式被破坏。
	value = strings.ReplaceAll(value, `\`, `\\`)
	return strings.ReplaceAll(value, `"`, `\"`)
}

func trimForMilvus(value string, maxRunes int) string {
	// Milvus VarChar 列有最大长度限制，入库前先做截断。
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes])
}
