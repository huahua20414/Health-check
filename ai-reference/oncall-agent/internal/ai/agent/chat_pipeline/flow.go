// Package chat_pipeline 中的本文件负责创建带工具能力的 ReAct Agent。
package chat_pipeline

import (
	"context"

	"oncall-agent/internal/ai/tools"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
)

// 获取模型，配合四个工具创建一个react的agent实例
// agent的四个工具，查询服务器日志工具(mcp自动暴露)，grafana连接并查询所有的alert警告并返回给大模型能看懂的结构体字符串，
// mysql根据大模型输入自动查询数据并返回数据,获取当前时间,向量检索工具(由向量数据库和大模型组成，大模型用于将输入转换为向量)
func newReactAgentLambda(ctx context.Context) (lba *compose.Lambda, err error) {
	config := &react.AgentConfig{
		MaxStep:            25,
		ToolReturnDirectly: map[string]struct{}{}}
	//获取模型
	chatModelIns11, err := newChatModel(ctx)
	if err != nil {
		return nil, err
	}
	//模型赋值
	config.ToolCallingModel = chatModelIns11
	//searchTool, err := newSearchTool(ctx)
	//if err != nil {
	//	return nil, err
	//}
	//获取查询日志工具
	mcpTool, err := tools.GetLogMcpTool()
	if err != nil {
		return nil, err
	}
	//react agent赋值
	config.ToolsConfig.Tools = mcpTool
	config.ToolsConfig.Tools = append(config.ToolsConfig.Tools, tools.NewGrafanaAlertsQueryTool())
	config.ToolsConfig.Tools = append(config.ToolsConfig.Tools, tools.NewMysqlCrudTool())
	config.ToolsConfig.Tools = append(config.ToolsConfig.Tools, tools.NewGetCurrentTimeTool())
	//向量检索
	config.ToolsConfig.Tools = append(config.ToolsConfig.Tools, tools.NewQueryInternalDocsTool())

	ins, err := react.NewAgent(ctx, config)
	if err != nil {
		return nil, err
	}
	//创建 ReAct Agent 实例
	//将 Agent 包装为 Graph 可用的 Lambda 节点
	lba, err = compose.AnyLambda(ins.Generate, ins.Stream, nil, nil)
	if err != nil {
		return nil, err
	}
	return lba, nil
}
