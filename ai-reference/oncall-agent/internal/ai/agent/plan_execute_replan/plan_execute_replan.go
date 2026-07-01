// Package plan_execute_replan 组装规划、执行和重规划 Agent，并输出最终分析结果。
package plan_execute_replan

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino-examples/adk/common/prints"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/schema"
)

// todo：这里可以改成流式输出
// todo:面试:这里做了plan输出判断，判断哪部分是真正输出
func BuildPlanAgent(ctx context.Context, query string) (string, []string, error) {
	//获取模型只是
	planAgent, err := NewPlanner(ctx)
	if err != nil {
		return "", []string{}, err
	}
	//获取模型但是给他添加了工具
	executeAgent, err := NewExecutor(ctx)
	if err != nil {
		return "", []string{}, err
	}
	//这个只是一个模型
	replanAgent, err := NewRePlanAgent(ctx)
	if err != nil {
		return "", []string{}, err
	}
	planExecuteAgent, err := planexecute.New(ctx, &planexecute.Config{
		Planner:       planAgent,
		Executor:      executeAgent,
		Replanner:     replanAgent,
		MaxIterations: 1000,
	})
	if err != nil {
		return "", []string{}, fmt.Errorf("build PlanExecuteAgent Error: %v", err)
	}
	r := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent: planExecuteAgent,
	})
	iter := r.Query(ctx, query)
	var lastMessage adk.Message
	var lastReplannerContent string
	var detail []string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		fmt.Println("------------- Event -------------")
		prints.Event(event)
		if event.Output != nil {
			lastMessage, _, err = adk.GetMessage(event)
			detail = append(detail, lastMessage.String())
			if lastMessage != nil &&
				event.AgentName == "Replanner" &&
				event.Output.MessageOutput != nil &&
				event.Output.MessageOutput.Role == schema.Assistant &&
				lastMessage.Content != "" {
				lastReplannerContent = lastMessage.Content
			}
		}
	}
	if lastMessage == nil {
		return "", []string{}, fmt.Errorf("get lastMessage Error")
	}
	if lastReplannerContent != "" {
		var payload struct {
			Response string `json:"response"`
		}
		if err := json.Unmarshal([]byte(lastReplannerContent), &payload); err == nil && payload.Response != "" {
			return payload.Response, detail, nil
		}
		return lastReplannerContent, detail, nil
	}
	if len(lastMessage.ToolCalls) > 0 {
		return lastMessage.ToolCalls[0].Function.Arguments, detail, nil
	}
	return "", detail, nil
}
