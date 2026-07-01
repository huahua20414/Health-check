// Package tools 中的本文件实现内部文档查询工具，用于从知识库召回处理方案。
package tools

import (
	// context 用于控制工具调用生命周期。
	"context"
	// json 用于把检索结果序列化为字符串输出。
	"encoding/json"
	// fmt 用于包装错误信息。
	"fmt"
	// retriever 提供统一的增强检索能力（改写+粗排+精排）。
	"oncall-agent/internal/ai/retriever"

	// tool 是 Eino 的工具接口定义。
	"github.com/cloudwego/eino/components/tool"
	// utils 提供工具构造辅助函数。
	"github.com/cloudwego/eino/components/tool/utils"
)

// QueryInternalDocsInput 是 query_internal_docs 工具输入。
type QueryInternalDocsInput struct {
	// Query 是用户传入的文档检索查询词。
	Query string `json:"query" jsonschema:"description=The query string to search in internal documentation for relevant information and processing steps"`
}

// NewQueryInternalDocsTool 创建内部文档检索工具实例。
func NewQueryInternalDocsTool() tool.InvokableTool {
	// 通过 InferOptionableTool 构建强类型工具。
	t, err := utils.InferOptionableTool(
		// 工具名称，给 Agent 进行 function calling 识别。
		"query_internal_docs",
		// 工具描述，帮助模型理解什么时候调用该工具。
		"Use this tool to search internal documentation and knowledge base for relevant information. It performs RAG (Retrieval-Augmented Generation) to find similar documents and extract processing steps. This is useful when you need to understand internal procedures, best practices, or step-by-step guides stored in the company's documentation.",
		// 具体执行逻辑：输入 query，输出 JSON 字符串。
		func(ctx context.Context, input *QueryInternalDocsInput, opts ...tool.Option) (output string, err error) {
			// 创建增强检索器实例（内部包含 query 改写、粗排、精排）。
			rr, err := retriever.NewMilvusRetriever(ctx)
			// 创建失败则返回错误给 Agent，而不是直接退出进程。
			if err != nil {
				// 包装错误并返回。
				return "", fmt.Errorf("create internal docs retriever failed: %w", err)
			}
			// 执行检索。
			resp, err := rr.Retrieve(ctx, input.Query)
			// 检索失败则返回错误给 Agent。
			if err != nil {
				// 包装错误并返回。
				return "", fmt.Errorf("retrieve internal docs failed: %w", err)
			}
			// 把检索结果转成 JSON 字符串，便于模型读取。
			respBytes, err := json.Marshal(resp)
			// 序列化失败返回错误。
			if err != nil {
				// 包装错误并返回。
				return "", fmt.Errorf("marshal internal docs result failed: %w", err)
			}
			// 写入工具输出字符串。
			output = string(respBytes)
			// 返回成功结果。
			return output, nil
		})
	// 工具初始化失败属于启动期配置错误，保留 panic 以便尽早暴露。
	if err != nil {
		// 抛出带上下文的 panic 错误。
		panic(fmt.Errorf("init query_internal_docs tool failed: %w", err))
	}
	// 返回工具实例。
	return t
}
