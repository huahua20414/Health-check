// Package chat_pipeline 中的本文件负责为对话 Agent 提供聊天模型实例。
package chat_pipeline

import (
	"context"

	"oncall-agent/internal/ai/models"

	"github.com/cloudwego/eino/components/model"
)

func newChatModel(ctx context.Context) (cm model.ToolCallingChatModel, err error) {
	cm, err = models.GoogleGeminiModel(ctx)
	if err != nil {
		return nil, err
	}
	return cm, nil
}
