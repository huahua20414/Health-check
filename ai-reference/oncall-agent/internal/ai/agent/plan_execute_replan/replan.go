// Package plan_execute_replan 中的本文件负责创建评估进度并决定后续动作的 Replanner。
package plan_execute_replan

import (
	"context"

	"oncall-agent/internal/ai/models"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
)

func NewRePlanAgent(ctx context.Context) (adk.Agent, error) {
	model, err := models.GoogleGeminiModel(ctx)
	if err != nil {
		return nil, err
	}
	return planexecute.NewReplanner(ctx, &planexecute.ReplannerConfig{
		ChatModel: model,
	})
}
