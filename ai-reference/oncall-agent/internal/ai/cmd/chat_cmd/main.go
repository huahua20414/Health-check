// Package main 提供对话 Agent 的命令行入口，用于本地验证多轮对话主链路。
package main

import (
	"context"
	"fmt"

	"oncall-agent/internal/ai/agent/chat_pipeline"
	"oncall-agent/utility/mem"

	"github.com/cloudwego/eino/schema"
)

func main() {
	ctx := context.Background()
	id := "111"
	history, err := mem.GetMessages(ctx, id)
	if err != nil {
		return
	}
	userMessage := &chat_pipeline.UserMessage{
		ID:      id,
		Query:   "帮我利用腾讯云日志工具mcp查询一下最近10条日志，不管时间只要能查询到就可以",
		History: history,
	}
	runner, err := chat_pipeline.BuildChatAgent(ctx)
	if err != nil {
		panic(err)
	}
	// 第一次对话
	out, err := runner.Invoke(ctx, userMessage)
	if err != nil {
		panic(err)
	}
	answer := out.Content
	fmt.Println("A:", answer)
	return

	mem.AppendMessages(ctx, id,
		schema.UserMessage("你好"),
		schema.AssistantMessage(out.Content, nil),
	)
	history, _ = mem.GetMessages(ctx, id)

	// 第二次对话
	userMessage = &chat_pipeline.UserMessage{
		ID:      id,
		Query:   "现在是几点",
		History: history,
	}
	out, err = runner.Invoke(ctx, userMessage)
	if err != nil {
		panic(err)
	}
	answer = out.Content
	fmt.Println("----------------")
	fmt.Println("Q: 现在是几点")
	fmt.Println("A:", answer)
}
