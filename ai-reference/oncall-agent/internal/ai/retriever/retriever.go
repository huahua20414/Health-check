// Package retriever 负责创建基于 Milvus 的知识检索器，供 RAG 流程复用。
package retriever

import (
	// json 用于编解码改写输入载荷（query + history）。
	"encoding/json"
	// context 用于控制请求生命周期和超时取消。
	"context"
	// fmt 用于拼接错误信息与改写提示词。
	"fmt"
	// math 用于计算向量相似度时的开方。
	"math"
	// sort 用于按精排分数降序排序。
	"sort"
	// strings 用于 query 和文本的清洗。
	"strings"
	// utf8 用于限制改写 query 的字符长度。
	"unicode/utf8"

	// embedder 负责初始化 embedding 模型。
	"oncall-agent/internal/ai/embedder"
	// models 负责初始化用于 query 改写的 LLM。
	"oncall-agent/internal/ai/models"
	// client 负责初始化 Milvus 客户端。
	"oncall-agent/utility/client"
	// common 存放 Milvus collection/field 常量。
	"oncall-agent/utility/common"

	// milvus 是 Eino 提供的 Milvus Retriever 组件实现。
	"github.com/cloudwego/eino-ext/components/retriever/milvus"
	// embedding 接口用于持有 embedding 实例。
	"github.com/cloudwego/eino/components/embedding"
	// model 接口用于调用改写 LLM。
	"github.com/cloudwego/eino/components/model"
	// retriever 是统一检索接口与调用选项定义。
	"github.com/cloudwego/eino/components/retriever"
	// schema 提供 Document/Message 等通用结构。
	"github.com/cloudwego/eino/schema"
)

const (
	// defaultFinalTopK 是最终返回给上游的文档数量默认值。
	defaultFinalTopK = 3
	// defaultCoarseRecallTopK 是粗排阶段默认召回条数。
	defaultCoarseRecallTopK = 12
	// coarseRecallMultiplier 表示粗排条数相对最终 TopK 的放大倍数。
	coarseRecallMultiplier = 4
	// maxRewriteQueryLen 用于限制改写结果过长导致检索噪音增加。
	maxRewriteQueryLen = 256
	// rewriteInputPrefix 用于标记“带历史上下文的改写输入”。
	rewriteInputPrefix = "__rag_rewrite_input__:"
)

// rewriteInputPayload 是传给改写器的结构化输入。
type rewriteInputPayload struct {
	// Query 是当前用户问题。
	Query string `json:"query"`
	// History 是用于代词消解的最近对话摘要文本。
	History string `json:"history,omitempty"`
}

// enhancedMilvusRetriever 在基础 Milvus 检索外增加 query 改写和 rerank。
type enhancedMilvusRetriever struct {
	// baseRetriever 是底层 Milvus 粗排检索器。
	baseRetriever retriever.Retriever
	// embedder 用于 query 与 doc 的向量化精排。
	embedder embedding.Embedder
	// rewriteModel 用于把用户 query 改写成更可检索表达。
	rewriteModel model.BaseChatModel
	// cfg 保存检索链路的 TopK、阈值和 metadata filter 配置。
	cfg RetrievalConfig
}

// EncodeRewriteInput 把 query 和历史上下文打包成字符串，供检索器解码后做代词消解改写。
func EncodeRewriteInput(query, history string) string {
	// 清理 query 首尾空白。
	query = strings.TrimSpace(query)
	// 空 query 直接返回空字符串。
	if query == "" {
		// 返回空。
		return ""
	}
	// 组装结构化载荷。
	payload := rewriteInputPayload{
		// 写入 query。
		Query: query,
		// 写入历史上下文（可为空）。
		History: strings.TrimSpace(history),
	}
	// 序列化为 JSON。
	data, err := json.Marshal(payload)
	// 序列化失败时回退原 query，避免中断主链路。
	if err != nil {
		// 返回原 query。
		return query
	}
	// 加上前缀，避免和普通 query 混淆。
	return rewriteInputPrefix + string(data)
}

// decodeRewriteInput 解析 EncodeRewriteInput 的结果；普通 query 会原样返回。
func decodeRewriteInput(input string) (query string, history string) {
	// 清理输入首尾空白。
	input = strings.TrimSpace(input)
	// 空输入直接返回空。
	if input == "" {
		// 返回空 query 和空 history。
		return "", ""
	}
	// 不带前缀说明是普通 query，不携带历史上下文。
	if !strings.HasPrefix(input, rewriteInputPrefix) {
		// 原样作为 query 返回。
		return input, ""
	}
	// 去掉前缀拿到 JSON 载荷文本。
	payloadText := strings.TrimSpace(strings.TrimPrefix(input, rewriteInputPrefix))
	// 前缀后没有内容时，降级按原输入处理。
	if payloadText == "" {
		// 返回原输入，避免丢 query。
		return input, ""
	}
	// 反序列化载荷。
	var payload rewriteInputPayload
	// 解析失败时降级按原输入处理。
	if err := json.Unmarshal([]byte(payloadText), &payload); err != nil {
		// 返回原输入，避免丢 query。
		return input, ""
	}
	// 清理载荷中的 query。
	query = strings.TrimSpace(payload.Query)
	// 清理载荷中的 history。
	history = strings.TrimSpace(payload.History)
	// 载荷 query 为空时降级按原输入处理。
	if query == "" {
		// 返回原输入，避免丢 query。
		return input, history
	}
	// 返回解码后的 query 与 history。
	return query, history
}

// NewMilvusRetriever 创建增强版检索器（改写 + 粗排 + 精排）。
func NewMilvusRetriever(ctx context.Context) (rtr retriever.Retriever, err error) {
	// 加载检索配置；配置缺失时保持原默认行为。
	cfg := loadRetrievalConfig(ctx)
	// 创建 Milvus 客户端。
	cli, err := client.NewMilvusClient(ctx)
	// 客户端创建失败直接返回错误。
	if err != nil {
		// 返回原始错误让上游可观察。
		return nil, err
	}
	// 初始化 embedding 模型实例。
	eb, err := embedder.Embedding(ctx)
	// embedding 初始化失败直接返回错误。
	if err != nil {
		// 返回原始错误让上游可观察。
		return nil, err
	}
	// 构建 Milvus 基础检索器作为粗排入口。
	baseRetriever, err := milvus.NewRetriever(ctx, &milvus.RetrieverConfig{
		// 注入 Milvus 客户端。
		Client: cli,
		// 指定检索的 collection。
		Collection: common.MilvusCollectionName,
		// 指定向量字段名。
		VectorField: common.MilvusFieldVector,
		// 指定需要回传给上游的字段。
		OutputFields: []string{
			// 回传文档 ID。
			common.MilvusFieldID,
			// 回传文档正文。
			common.MilvusFieldContent,
			// 回传元数据字段。
			common.MilvusFieldMetadata,
		},
		// 粗排阶段默认召回更多候选供后续 rerank。
		TopK: defaultCoarseRecallTopK,
		// 粗排检索同样需要 query embedding。
		Embedding: eb,
	})
	// 基础检索器构建失败直接返回错误。
	if err != nil {
		// 返回原始错误让上游可观察。
		return nil, err
	}
	// Query rewrite 是增强能力，不应该影响主链路可用性。
	// 所以这里初始化失败时降级为“只做向量检索 + rerank”。
	rewriteModel, rewriteErr := models.GoogleGeminiModel(ctx)
	// 如果改写模型初始化失败，改写能力降级关闭。
	if rewriteErr != nil {
		// 置空表示后续直接回退原 query。
		rewriteModel = nil
	}
	// 返回增强检索器实例。
	return &enhancedMilvusRetriever{
		// 注入基础粗排检索器。
		baseRetriever: baseRetriever,
		// 注入 embedding 实例用于精排。
		embedder: eb,
		// 注入改写模型（可能为空表示降级）。
		rewriteModel: rewriteModel,
		// 注入检索配置。
		cfg: cfg,
	}, nil
}

// Retrieve 执行完整在线链路：query 改写 -> 粗排 -> rerank -> 截断。
func (r *enhancedMilvusRetriever) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]*schema.Document, error) {
	// 清理 query 首尾空白，减少空输入干扰。
	query = strings.TrimSpace(query)
	// 空 query 直接返回空结果。
	if query == "" {
		// 返回空 slice 保持调用方遍历安全。
		return []*schema.Document{}, nil
	}
	// 解析结构化输入，拿到真实 query 和用于代词消解的 history。
	userQuery, historyContext := decodeRewriteInput(query)
	// 真实 query 为空时，回退到原 query。
	if userQuery == "" {
		// 回退原 query，确保后续流程可继续。
		userQuery = query
	}
	// 解析最终需要返回多少条文档。
	finalTopK := resolveFinalTopKWithDefault(r.cfg.FinalTopK, opts...)
	// 基于 finalTopK 计算粗排召回条数。
	coarseTopK := resolveCoarseTopKWithMultiplier(finalTopK, r.cfg.CoarseRecallMultiplier)
	// 解析相似度阈值，调用方传入的阈值优先于配置。
	scoreThreshold := resolveScoreThreshold(r.cfg.ScoreThreshold, opts...)
	// 先尝试改写 query，失败会自动回退原 query。
	rewrittenQuery := r.rewriteQuery(ctx, userQuery, historyContext)

	// 复制一份 opts，避免修改调用方传入切片。
	coarseOpts := append([]retriever.Option{}, opts...)
	// 覆盖粗排 TopK，让 Milvus 召回更多候选文档。
	coarseOpts = append(coarseOpts, retriever.WithTopK(coarseTopK))
	// 如果配置了 Milvus metadata filter，则在粗排阶段减少无关 chunk 进入候选集。
	if r.cfg.MetadataFilter != "" {
		coarseOpts = append(coarseOpts, milvus.WithFilter(r.cfg.MetadataFilter))
	}
	// 如果配置了阈值，先交给底层检索器过滤一轮。
	if scoreThreshold > 0 {
		coarseOpts = append(coarseOpts, retriever.WithScoreThreshold(scoreThreshold))
	}

	// 执行 Milvus 粗排检索。
	docs, err := r.baseRetriever.Retrieve(ctx, rewrittenQuery, coarseOpts...)
	// 粗排失败直接返回错误，由上游决定是否重试。
	if err != nil {
		// 保留原始错误链路。
		return nil, err
	}
	// 粗排无结果时直接返回。
	if len(docs) == 0 {
		// 返回空结果给上游。
		return docs, nil
	}

	// 对粗排结果做向量精排。
	reranked, err := r.rerank(ctx, userQuery, rewrittenQuery, docs)
	// 精排失败时降级返回粗排结果，避免链路中断。
	if err != nil {
		// 仍按最终 TopK 截断，保证返回规模稳定。
		return limitTopK(filterByScoreThreshold(docs, scoreThreshold), finalTopK), nil
	}
	// 精排成功后返回最终 TopK。
	return limitTopK(filterByScoreThreshold(reranked, scoreThreshold), finalTopK), nil
}

// rewriteQuery 用 LLM 把原始 query 改写成更适合向量检索的表达。
func (r *enhancedMilvusRetriever) rewriteQuery(ctx context.Context, query, historyContext string) string {
	// 改写模型不可用时直接回退原 query。
	if r.rewriteModel == nil {
		// 返回原 query。
		return query
	}
	// 没有历史时用占位文本，保持 prompt 结构稳定。
	historyBlock := strings.TrimSpace(historyContext)
	// 历史为空时写入显式占位。
	if historyBlock == "" {
		// 用占位提示模型当前无历史。
		historyBlock = "[无历史上下文]"
	}
	// 组装改写提示词输入。
	rewritePrompt := fmt.Sprintf("最近对话历史（仅用于理解指代关系，不要复述）：\n%s\n\n用户原始查询：%s", historyBlock, query)
	// 调用模型执行 query 改写。
	resp, err := r.rewriteModel.Generate(ctx, []*schema.Message{
		// 系统提示要求模型只输出改写结果。
		schema.SystemMessage(`你是检索查询改写器。任务：把用户问题改写成更利于企业知识库向量检索的一句话查询。
要求：
1. 只输出改写后的查询，不要解释。
2. 保留原问题中的关键实体、告警名、组件名、时间约束。
3. 不要引入原问题没有出现的新事实。
4. 如果问题里出现“他/它/这个/那个/这件事”等指代词，优先结合最近对话历史做消解，把代词替换成明确实体。`),
		// 用户消息携带原 query。
		schema.UserMessage(rewritePrompt),
		// 低温度减少改写发散。
	}, model.WithTemperature(0))
	// 生成失败或返回空对象时回退原 query。
	if err != nil || resp == nil {
		// 返回原 query。
		return query
	}
	// 规范化改写文本，剔除前缀和杂质。
	rewritten := normalizeRewriteQuery(query, resp.Content)
	// 规范化后为空则回退原 query。
	if rewritten == "" {
		// 返回原 query。
		return query
	}
	// 返回清洗后的改写 query。
	return rewritten
}

// rerank 对粗排候选做向量相似度精排。
func (r *enhancedMilvusRetriever) rerank(ctx context.Context, originalQuery, rewrittenQuery string, docs []*schema.Document) ([]*schema.Document, error) {
	// 候选少于等于 1 时无需排序。
	if len(docs) <= 1 {
		// 直接返回原候选。
		return docs, nil
	}

	// 用原 query 和改写 query 的平均向量作为精排查询向量，避免改写偏移。
	queryTexts := []string{originalQuery}
	// 改写 query 存在且不同于原 query 时一并参与平均。
	if rewrittenQuery != "" && rewrittenQuery != originalQuery {
		// 追加改写 query。
		queryTexts = append(queryTexts, rewrittenQuery)
	}
	// 计算 query 侧向量。
	queryVectors, err := r.embedder.EmbedStrings(ctx, queryTexts)
	// query 向量化失败返回错误。
	if err != nil {
		// 包装错误便于定位阶段。
		return nil, fmt.Errorf("embed rerank query failed: %w", err)
	}
	// query 向量为空同样视为异常。
	if len(queryVectors) == 0 {
		// 返回明确错误。
		return nil, fmt.Errorf("embed rerank query failed: empty embedding result")
	}
	// 对多个 query 向量求平均作为最终检索向量。
	queryVector := averageVectors(queryVectors)

	// 预分配 doc 文本数组，减少扩容。
	docTexts := make([]string, 0, len(docs))
	// 预分配有效 doc 数组，保持与 docTexts 对齐。
	validDocs := make([]*schema.Document, 0, len(docs))
	// 遍历粗排候选，过滤空内容文档。
	for _, doc := range docs {
		// 去除文档内容首尾空白。
		content := strings.TrimSpace(doc.Content)
		// 空文档跳过，避免向量化噪声。
		if content == "" {
			// 继续处理下一条文档。
			continue
		}
		// 收集有效文档文本用于批量向量化。
		docTexts = append(docTexts, content)
		// 记录对应文档对象用于回写分数。
		validDocs = append(validDocs, doc)
	}
	// 没有可用文档时返回原结果。
	if len(validDocs) == 0 {
		// 直接返回。
		return docs, nil
	}

	// 批量计算文档向量。
	docVectors, err := r.embedder.EmbedStrings(ctx, docTexts)
	// 文档向量化失败返回错误。
	if err != nil {
		// 包装错误便于定位阶段。
		return nil, fmt.Errorf("embed rerank docs failed: %w", err)
	}
	// 向量数量和文档数量不一致时返回错误。
	if len(docVectors) != len(validDocs) {
		// 返回结构化错误便于排查。
		return nil, fmt.Errorf("rerank docs embedding mismatch: got=%d want=%d", len(docVectors), len(validDocs))
	}

	// scoredDoc 表示文档和其计算得到的最终分数。
	type scoredDoc struct {
		// doc 是原始文档对象。
		doc *schema.Document
		// score 是融合后的精排分数。
		score float64
	}
	// 预分配打分结果数组。
	scored := make([]scoredDoc, 0, len(validDocs))
	// 对每条候选文档计算精排分数。
	for idx, doc := range validDocs {
		// 计算 query 和 doc 的余弦相似度。一个是平均向量，一个是文档向量
		cosineScore := cosineSimilarity(queryVector, docVectors[idx])
		// 保留一部分粗排分数，避免 rerank 对边界样本过度抖动。
		finalScore := cosineScore*0.8 + doc.Score()*0.2
		// 回写文档分数，方便上游观测与调试。
		doc.WithScore(finalScore)
		// 记录分数和文档映射关系。
		scored = append(scored, scoredDoc{doc: doc, score: finalScore})
	}

	// 按分数降序稳定排序。
	sort.SliceStable(scored, func(i, j int) bool {
		// 分数高的排前面。
		return scored[i].score > scored[j].score
	})

	// 预分配结果切片。
	result := make([]*schema.Document, 0, len(scored))
	// 按排序后的顺序提取文档对象。
	for _, row := range scored {
		// 依次追加到最终结果。
		result = append(result, row.doc)
	}
	// 返回精排完成的文档列表。
	return result, nil
}

// resolveFinalTopK 解析调用方传入的最终 TopK，没传则取默认值。
func resolveFinalTopK(opts ...retriever.Option) int {
	return resolveFinalTopKWithDefault(defaultFinalTopK, opts...)
}

// resolveFinalTopKWithDefault 解析调用方传入的最终 TopK，没传则取给定默认值。
func resolveFinalTopKWithDefault(defaultTopK int, opts ...retriever.Option) int {
	// 先使用默认值。
	topK := defaultTopK
	// 提取 retriever 通用选项。
	commonOpt := retriever.GetCommonOptions(nil, opts...)
	// 如果调用方显式传了合法 TopK，则覆盖默认值。
	if commonOpt.TopK != nil && *commonOpt.TopK > 0 {
		// 使用调用方指定值。
		topK = *commonOpt.TopK
	}
	// 防御非法值，小于 1 时强制矫正。
	if topK < 1 {
		// 至少返回 1 条。
		return 1
	}
	// 返回最终 TopK。
	return topK
}

// resolveCoarseTopK 根据最终 TopK 计算粗排召回数量。
func resolveCoarseTopK(finalTopK int) int {
	return resolveCoarseTopKWithMultiplier(finalTopK, coarseRecallMultiplier)
}

// resolveCoarseTopKWithMultiplier 根据最终 TopK 和配置倍数计算粗排召回数量。
func resolveCoarseTopKWithMultiplier(finalTopK int, multiplier int) int {
	// 从粗排默认值开始。
	topK := defaultCoarseRecallTopK
	if multiplier <= 0 {
		multiplier = coarseRecallMultiplier
	}
	// 根据倍数规则计算最低粗排需求。
	requiredTopK := finalTopK * multiplier
	// 如果倍数需求更高，则提升粗排条数。
	if requiredTopK > topK {
		// 提升粗排条数。
		topK = requiredTopK
	}
	// 粗排条数不能小于最终条数。
	if topK < finalTopK {
		// 兜底矫正。
		topK = finalTopK
	}
	// 防御非法值，小于 1 时强制矫正。
	if topK < 1 {
		// 至少召回 1 条。
		return 1
	}
	// 返回粗排 TopK。
	return topK
}

// resolveScoreThreshold 解析调用侧或配置侧的相似度阈值。
func resolveScoreThreshold(defaultThreshold float64, opts ...retriever.Option) float64 {
	threshold := defaultThreshold
	commonOpt := retriever.GetCommonOptions(nil, opts...)
	if commonOpt.ScoreThreshold != nil {
		threshold = *commonOpt.ScoreThreshold
	}
	if threshold < 0 {
		return 0
	}
	return threshold
}

// filterByScoreThreshold 在精排后再做一次阈值过滤，避免低相似 chunk 进入模型上下文。
func filterByScoreThreshold(docs []*schema.Document, threshold float64) []*schema.Document {
	if threshold <= 0 || len(docs) == 0 {
		return docs
	}
	filtered := make([]*schema.Document, 0, len(docs))
	for _, doc := range docs {
		if doc != nil && doc.Score() >= threshold {
			filtered = append(filtered, doc)
		}
	}
	return filtered
}

// limitTopK 把结果截断到指定条数。
func limitTopK(docs []*schema.Document, topK int) []*schema.Document {
	// 如果无需截断，直接返回原切片。
	if topK <= 0 || len(docs) <= topK {
		// 原样返回。
		return docs
	}
	// 仅返回前 topK 条。
	return docs[:topK]
}

// normalizeRewriteQuery 清洗 LLM 改写结果文本。
func normalizeRewriteQuery(original, rewritten string) string {
	// 去掉首尾空白。
	rewritten = strings.TrimSpace(rewritten)
	// 空字符串直接返回空，触发上层回退原 query。
	if rewritten == "" {
		// 返回空值。
		return ""
	}
	// 去除常见包裹符号。
	rewritten = strings.Trim(rewritten, "`\"' \n\t")
	// 去掉中文前缀“改写后查询：”。
	rewritten = strings.TrimPrefix(rewritten, "改写后查询：")
	// 去掉中文前缀“改写后的查询：”。
	rewritten = strings.TrimPrefix(rewritten, "改写后的查询：")
	// 去掉中文前缀“查询：”。
	rewritten = strings.TrimPrefix(rewritten, "查询：")
	// 去掉英文前缀“query:”。
	rewritten = strings.TrimPrefix(rewritten, "query:")
	// 二次去空白，处理前缀剥离后残留。
	rewritten = strings.TrimSpace(rewritten)
	// 清洗后为空则返回空。
	if rewritten == "" {
		// 返回空值。
		return ""
	}
	// 改写结果过长时回退原 query，避免召回噪声。
	if utf8.RuneCountInString(rewritten) > maxRewriteQueryLen {
		// 返回原 query。
		return original
	}
	// 返回规范化后的 query。
	return rewritten
}

// averageVectors 对多个向量按维度求平均。
func averageVectors(vectors [][]float64) []float64 {
	// 输入为空时返回 nil。
	if len(vectors) == 0 {
		// 返回 nil 表示无向量。
		return nil
	}
	// 以第一条向量长度作为目标维度。
	dim := len(vectors[0])
	// 初始化均值向量。
	result := make([]float64, dim)
	// 记录参与平均的有效向量数量。
	valid := 0
	// 遍历所有向量。
	for _, vector := range vectors {
		// 维度不一致则跳过，避免越界与污染。
		if len(vector) != dim {
			// 继续处理下一条向量。
			continue
		}
		// 有效向量计数加一。
		valid++
		// 按维度累加。
		for i := 0; i < dim; i++ {
			// 累加当前维度值。
			result[i] += vector[i]
		}
	}
	// 没有有效向量时返回 nil。
	if valid == 0 {
		// 返回 nil 表示无法计算平均值。
		return nil
	}
	// 计算平均除数。
	divisor := float64(valid)
	// 遍历每个维度做平均。
	for i := 0; i < dim; i++ {
		// 当前维度累加和除以有效向量数量。
		result[i] /= divisor
	}
	// 返回均值向量。
	return result
}

// cosineSimilarity 计算两个向量的余弦相似度。
func cosineSimilarity(a, b []float64) float64 {
	// 向量为空或维度不一致时返回 0。
	if len(a) == 0 || len(a) != len(b) {
		// 返回 0 代表不可比较。
		return 0
	}
	// dot 是点积，normA/normB 是范数平方和。
	var dot, normA, normB float64
	// 遍历向量每个维度。
	for i := range a {
		// 计算点积累加。
		dot += a[i] * b[i]
		// 计算向量 a 范数平方和。
		normA += a[i] * a[i]
		// 计算向量 b 范数平方和。
		normB += b[i] * b[i]
	}
	// 任一向量范数为 0 时无法计算余弦值。
	if normA == 0 || normB == 0 {
		// 返回 0 表示无效相似度。
		return 0
	}
	// 返回余弦相似度。
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
