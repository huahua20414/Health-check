// Package chat 中的本文件实现基于 SSE 的流式对话接口。
package chat

import (
	"context"
	"errors"
	"io"
	"strings"

	"oncall-agent/api/chat/v1"
	"oncall-agent/internal/ai/agent/chat_pipeline"
	"oncall-agent/utility/log_call_back"
	"oncall-agent/utility/mem"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/gogf/gf/v2/frame/g"
)

func (c *ControllerV1) ChatStream(ctx context.Context, req *v1.ChatStreamReq) (res *v1.ChatStreamRes, err error) {
	id := req.Id
	msg := req.Question

	ctx = context.WithValue(ctx, "client_id", req.Id)
	client, err := c.service.Create(ctx, g.RequestFromCtx(ctx))
	if err != nil {
		return nil, err
	}

	// 流式对话和普通对话共用同一套记忆读取逻辑，避免两条链路的上下文不一致。
	history, err := mem.GetMessagesForQuery(ctx, id, msg)
	if err != nil {
		return nil, err
	}
	userMessage := &chat_pipeline.UserMessage{
		ID:      id,
		Query:   msg,
		History: history,
	}

	runner, err := chat_pipeline.BuildChatAgent(ctx)
	sr, err := runner.Stream(ctx, userMessage, compose.WithCallbacks(log_call_back.LogCallback(nil)))
	if err != nil {
		client.SendToClient("error", err.Error())
		return nil, err
	}
	defer sr.Close()

	var fullResponse strings.Builder

	defer func() {
		completeResponse := fullResponse.String()
		if completeResponse != "" {
			if appendErr := mem.AppendMessages(ctx, id,
				schema.UserMessage(msg),
				schema.AssistantMessage(completeResponse, nil),
			); appendErr != nil {
				g.Log().Error(ctx, appendErr)
			}
		}
	}()

	for {
		chunk, err := sr.Recv()
		if errors.Is(err, io.EOF) {
			client.SendToClient("done", "Stream completed")
			return &v1.ChatStreamRes{}, nil
		}
		if err != nil {
			client.SendToClient("error", err.Error())
			return &v1.ChatStreamRes{}, nil
		}
		fullResponse.WriteString(chunk.Content)
		client.SendToClient("message", chunk.Content)
	}
}
