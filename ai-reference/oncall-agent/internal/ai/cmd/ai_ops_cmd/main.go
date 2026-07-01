// Package main 提供运维 Agent 的命令行入口，用于本地调试 Plan-Execute-Replan 流程。
package main

import (
	"context"
	"fmt"

	"oncall-agent/internal/ai/agent/plan_execute_replan"
)

func main() {
	ctx := context.Background()
	query := `
1. 你是一个智能的服务告警运维分析助手，先调用工具query_grafana_alerts获取当前所有活跃告警。
2. 如果没有活跃告警，直接说明当前无活跃告警，不要继续虚构分析。
3. 对每条告警，优先根据alert_name调用工具query_internal_docs查询内部处理方案；如果告警名称不足以定位处理方案，可以结合summary、instance等告警字段继续检索。
4.你可以自己思考一下错误原因结合内部处理方案
5.生成告警运维分析报告,格式如下：
告警分析报告
---
# 告警处理详情
## 活跃告警清单
## 告警根因分析N(第N个告警)
## 处理方案执行N(第N个告警)
## 结论
`
	resp, detail, err := plan_execute_replan.BuildPlanAgent(ctx, query)
	if err != nil {
		panic(err)
	}
	fmt.Println("----- Final detail -----")
	fmt.Println(detail)
	fmt.Println("----- Final Response -----")
	fmt.Println(resp)
}
