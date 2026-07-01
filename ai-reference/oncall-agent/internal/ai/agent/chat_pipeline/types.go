// Package chat_pipeline 中的本文件定义对话流水线使用的输入数据结构。
package chat_pipeline

import "github.com/cloudwego/eino/schema"

type UserMessage struct {
	ID      string            `json:"id"`
	Query   string            `json:"query"`
	History []*schema.Message `json:"history"`
}
