// Package plan_execute_replan 中的本文件负责创建拆解任务步骤的 Planner Agent。
package plan_execute_replan

import (
	"context"

	"oncall-agent/internal/ai/models"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
)

func NewPlanner(ctx context.Context) (adk.Agent, error) {
	planModel, err := models.GoogleGeminiModel(ctx)
	if err != nil {
		return nil, err
	}
	return planexecute.NewPlanner(ctx, &planexecute.PlannerConfig{
		ToolCallingChatModel: planModel,
	})
}
