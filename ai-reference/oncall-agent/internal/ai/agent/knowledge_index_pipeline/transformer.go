// Package knowledge_index_pipeline 中的本文件负责初始化知识库分片节点。
package knowledge_index_pipeline

import (
	// context 用来贯穿配置读取和组件初始化过程。
	"context"
	// fmt 用来包装带上下文的错误信息。
	"fmt"

	// document 提供文档切分器接口定义。
	"github.com/cloudwego/eino/components/document"
	// g 用来从配置中心读取分片参数。
	"github.com/gogf/gf/v2/frame/g"
)

// chunkingConfig 是混合分片器依赖的配置集合。
type chunkingConfig struct {
	// MaxChunkSize 控制单个 chunk 的目标上限。
	// 这里按字符数粗略控制，避免一个 chunk 过大导致 embedding 和检索效果变差。
	MaxChunkSize int
	// ChunkOverlap 让相邻 chunk 保留一部分重叠上下文，
	// 避免关键信息正好落在切分边界上被截断。
	ChunkOverlap int
	// MinChunkSize 用来避免切出太碎的小块。
	// 太短的 chunk 往往语义不完整，检索价值也低。
	MinChunkSize int
}

// newDocumentTransformer 负责构造文档切分节点。
// 这里不再直接使用默认 splitter，而是接入自定义 hybrid splitter。
func newDocumentTransformer(ctx context.Context) (document.Transformer, error) {
	// 先读取 hybrid splitter 依赖的配置。
	cfg, err := loadChunkingConfig(ctx)
	// 配置读取失败时直接返回。
	if err != nil {
		return nil, err
	}
	// 用读取到的配置创建混合分片器。
	return newHybridDocumentSplitter(cfg), nil
}

// loadChunkingConfig 负责从配置中心读取 hybrid splitter 需要的参数。
func loadChunkingConfig(ctx context.Context) (*chunkingConfig, error) {
	// 读取单个 chunk 的最大长度配置。
	maxChunkSizeValue, err := g.Cfg().Get(ctx, "knowledge_index.chunk_size")
	// 如果读取失败，就返回带字段名的错误。
	if err != nil {
		return nil, fmt.Errorf("read knowledge_index.chunk_size failed: %w", err)
	}
	// 读取 chunk 之间的 overlap 配置。
	chunkOverlapValue, err := g.Cfg().Get(ctx, "knowledge_index.chunk_overlap")
	// 如果读取失败，就返回带字段名的错误。
	if err != nil {
		return nil, fmt.Errorf("read knowledge_index.chunk_overlap failed: %w", err)
	}
	// 读取最小 chunk 长度配置。
	minChunkSizeValue, err := g.Cfg().Get(ctx, "knowledge_index.min_chunk_size")
	// 如果读取失败，就返回带字段名的错误。
	if err != nil {
		return nil, fmt.Errorf("read knowledge_index.min_chunk_size failed: %w", err)
	}

	// 把配置对象转换成当前代码里使用的强类型结构体。
	cfg := &chunkingConfig{
		// 读取并转换最大 chunk 长度。
		MaxChunkSize: maxChunkSizeValue.Int(),
		// 读取并转换 chunk overlap 长度。
		ChunkOverlap: chunkOverlapValue.Int(),
		// 读取并转换最小 chunk 长度。
		MinChunkSize: minChunkSizeValue.Int(),
	}
	// 下面这些默认值是给配置兜底。
	// 这样即使没显式配置，知识入库链路也能用一套可接受的默认分片策略跑起来。
	// 如果最大 chunk 长度没有配置或配置非法，就给一个默认值。
	if cfg.MaxChunkSize <= 0 {
		cfg.MaxChunkSize = 800
	}
	// overlap 不能是负数，负数没有实际意义。
	if cfg.ChunkOverlap < 0 {
		cfg.ChunkOverlap = 0
	}
	// overlap 不能大于 chunk 本身，否则滑窗会失去意义。
	// 如果 overlap 过大，就退回成最大长度的五分之一。
	if cfg.ChunkOverlap >= cfg.MaxChunkSize {
		cfg.ChunkOverlap = cfg.MaxChunkSize / 5
	}
	// 如果最小 chunk 长度没有配置或配置非法，就给一个默认值。
	if cfg.MinChunkSize <= 0 {
		cfg.MinChunkSize = 120
	}
	// 返回整理好的配置对象。
	return cfg, nil
}
