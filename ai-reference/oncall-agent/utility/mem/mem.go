// Package mem 提供可持久化的会话记忆能力。
// 当前实现采用“摘要 + 最近窗口”的两层结构：
// 1. 最近若干轮原始消息直接回放，保证短期上下文细节；
// 2. 更早的历史被压缩进 summary，避免 prompt 无限膨胀。
package mem

import (
	"context"
	"fmt"
	"sync"

	"github.com/cloudwego/eino/schema"
)

// Store 定义记忆存储的最小接口。
// 后续如果要接 Redis、Milvus 或其他存储，只需要实现这个接口。
type Store interface {
	GetMessages(ctx context.Context, sessionID string) ([]*schema.Message, error)
	// GetMessagesForQuery 允许存储层根据当前 query 做更智能的长期记忆召回，
	// 例如从向量库里挑出“和这次问题最相关”的事实，而不是把所有长期事实都塞进 prompt。
	GetMessagesForQuery(ctx context.Context, sessionID, query string) ([]*schema.Message, error)
	AppendMessages(ctx context.Context, sessionID string, msgs ...*schema.Message) error
}

var (
	defaultStore  Store
	initStoreErr  error
	initStoreOnce sync.Once
)

func getStore(ctx context.Context) (Store, error) {
	initStoreOnce.Do(func() {
		defaultStore, initStoreErr = NewMySQLStoreFromConfig(ctx)
	})
	if initStoreErr != nil {
		return nil, fmt.Errorf("init memory store failed: %w", initStoreErr)
	}
	return defaultStore, nil
}

// GetMessages 返回某个 session 当前可用的 prompt 记忆。
// 返回结果会自动拼上历史摘要和最近若干轮消息。
func GetMessages(ctx context.Context, sessionID string) ([]*schema.Message, error) {
	store, err := getStore(ctx)
	if err != nil {
		return nil, err
	}
	return store.GetMessages(ctx, sessionID)
}

// GetMessagesForQuery 先按 session_id + 当前query 取记忆：mem.GetMessagesForQuery，见 chat_v1_chat.go (line 20)。
// 读取时按顺序拼上下文：summary -> facts -> recent turns，见 mysql_store.go (line 225)。
// facts 的策略是：有 query 且 semanticStore 可用时，先去 Milvus 做语义召回；失败或无结果就降级拿 MySQL 最近 facts：见 mysql_store.go (line 454)。
// 最近窗口消息只取 compacted_at IS NULL 的最新 N 条，再倒序恢复真实对话顺序喂给模型：见 mysql_store.go (line 243) 和 mysql_store.go (line 268)。
func GetMessagesForQuery(ctx context.Context, sessionID, query string) ([]*schema.Message, error) {
	store, err := getStore(ctx)
	if err != nil {
		return nil, err
	}
	return store.GetMessagesForQuery(ctx, sessionID, query)
}

// AppendMessages Controller 在回答后把 user + assistant 一起写入记忆：见 chat_v1_chat.go (line 56) 和 chat_v1_chat_stream.go (line 54)。
// mem.AppendMessages 进入 MySQL 事务写 memory_turns，同时确保 memory_sessions 存在：见 mysql_store.go (line 280)。
// 如果“未压缩窗口”超过 max_window_size，触发压缩：把最早的消息做增量摘要写回 memory_sessions.summary，并把这些旧消息打 compacted_at 标记（不物理删）：见 mysql_store.go (line 329)。
// 同时从被压缩的旧消息抽取长期事实，幂等写入 memory_facts（按 session_id + hash 去重）：见 mysql_store.go (line 400) 和 mysql_store.go (line 414)。
// 事务提交后，如果开了语义召回，把全量 facts 同步到 Milvus memory_facts collection（float 向量）：见 mysql_store.go (line 319) 和 milvus_memory.go (line 114)。
// 用 sessionID + factHash 作为主键，保证同一条事实多次同步时会覆盖更新，而不是重复插入。
func AppendMessages(ctx context.Context, sessionID string, msgs ...*schema.Message) error {
	store, err := getStore(ctx)
	if err != nil {
		return err
	}
	return store.AppendMessages(ctx, sessionID, msgs...)
}
