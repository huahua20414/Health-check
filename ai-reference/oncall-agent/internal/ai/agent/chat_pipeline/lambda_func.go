// Package chat_pipeline 中的本文件负责把用户输入转换为检索和对话节点所需格式。
package chat_pipeline

import (
	"context"
	"strings"
	"time"

	"oncall-agent/internal/ai/retriever"

	"github.com/cloudwego/eino/schema"
)

// agent开始的时候分成两步，这里将用户输入分成两部分,
// newInputToRagLambda component initialization function of node 'InputToQuery' in graph 'EinoAgent'
// 这个是用来检索出document填充模版
func newInputToRagLambda(ctx context.Context, input *UserMessage, opts ...any) (output string, err error) {
	//这里是截取历史然后，有最大条数和单条消息最大字符数限制
	historyContext := buildRewriteHistoryContext(input.History)
	return retriever.EncodeRewriteInput(input.Query, historyContext), nil
}

// newInputToChatLambda component initialization function of node 'InputToHistory' in graph 'EinoAgent'
// 这里用来直接填充模版
func newInputToChatLambda(ctx context.Context, input *UserMessage, opts ...any) (output map[string]any, err error) {
	return map[string]any{
		"content": input.Query,
		"history": input.History,
		"date":    time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}

const (
	rewriteHistoryMaxMessages = 6
	rewriteHistoryMaxChars    = 1200
	rewriteMessageMaxChars    = 240
)

func buildRewriteHistoryContext(history []*schema.Message) string {
	if len(history) == 0 {
		return ""
	}

	type line struct {
		role    string
		content string
	}

	selected := make([]line, 0, rewriteHistoryMaxMessages)
	for idx := len(history) - 1; idx >= 0 && len(selected) < rewriteHistoryMaxMessages; idx-- {
		msg := history[idx]
		if msg == nil {
			continue
		}
		if msg.Role != schema.User && msg.Role != schema.Assistant {
			continue
		}
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}
		role := "assistant"
		if msg.Role == schema.User {
			role = "user"
		}
		selected = append(selected, line{
			role:    role,
			content: truncateRunes(content, rewriteMessageMaxChars),
		})
	}
	if len(selected) == 0 {
		return ""
	}
	//倒序
	for i, j := 0, len(selected)-1; i < j; i, j = i+1, j-1 {
		selected[i], selected[j] = selected[j], selected[i]
	}

	lines := make([]string, 0, len(selected))
	totalRunes := 0
	for _, item := range selected {
		text := item.role + ": " + item.content
		runes := len([]rune(text))
		if totalRunes+runes > rewriteHistoryMaxChars {
			break
		}
		lines = append(lines, text)
		totalRunes += runes
	}
	return strings.Join(lines, "\n")
}

func truncateRunes(content string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(content)
	if len(runes) <= limit {
		return content
	}
	return string(runes[:limit])
}
