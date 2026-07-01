package mem

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"oncall-agent/internal/ai/models"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/gogf/gf/v2/frame/g"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const summaryPromptPrefix = "以下是当前会话的历史摘要，请把它当作长期上下文参考，不要逐字复述：\n"
const factsPromptPrefix = "以下是从历史对话中提取出的长期事实记忆，仅在与当前问题相关时使用：\n"

// mysqlStoreConfig 是 memory 模块自己的配置聚合。
// 这里先把配置文件里的零散字段读出来，后面就不用到处直接访问 g.Cfg()。
type mysqlStoreConfig struct {
	DSN             string
	MaxWindowSize   int
	MaxSummaryChars int
	MaxFactCount    int
	// SemanticRecallEnabled 打开后，会把长期事实同步到 Milvus，
	// 读取历史时再按当前 query 做相关性召回。
	SemanticRecallEnabled bool
	SemanticTopK          int
	SemanticCollection    string
}

// mysqlStore 是当前真正的 memory 主实现：
// 1. MySQL 存原始消息、摘要、事实；
// 2. 可选地把 facts 同步到 Milvus 做语义召回。
type mysqlStore struct {
	db              *gorm.DB
	maxWindowSize   int
	maxSummaryChars int
	maxFactCount    int
	summarizer      Summarizer
	factExtractor   FactExtractor
	// semanticStore 是“可选增强”。
	// 它不可用时，主链路仍然可以退回到 MySQL 的摘要 + 最近事实，不影响对话可用性。
	semanticStore *semanticMemoryStore
}

// Summarizer 负责把旧消息压缩成可回放的摘要文本。
// 当前默认优先使用 LLM 摘要器，失败时回退到规则摘要器。
type Summarizer interface {
	Summarize(ctx context.Context, existing string, msgs []*schema.Message) (string, error)
}

// FactExtractor 负责从会话里提取稳定、可长期复用的事实记忆。
type FactExtractor interface {
	Extract(ctx context.Context, msgs []*schema.Message) ([]string, error)
}

type rollingSummarizer struct {
	maxChars int
}

// memorySession 表示“一个 session 的会话级状态”。
// 这里不存每一条消息，只存滚动摘要。
type memorySession struct {
	SessionID        string    `gorm:"column:session_id;primaryKey;size:128"`
	Summary          string    `gorm:"column:summary;type:longtext"`
	SummaryUpdatedAt time.Time `gorm:"column:summary_updated_at"`
	CreatedAt        time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt        time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (memorySession) TableName() string {
	return "memory_sessions"
}

type memoryTurn struct {
	ID        uint64 `gorm:"column:id;primaryKey;autoIncrement"`
	SessionID string `gorm:"column:session_id;index:idx_memory_turns_session_id_id,priority:1;size:128"`
	Role      string `gorm:"column:role;size:32"`
	Content   string `gorm:"column:content;type:longtext"`
	Payload   string `gorm:"column:payload;type:longtext"`
	// CompactedAt 为空表示这条原始消息仍属于“最近窗口”，会直接回放给模型。
	// 一旦被压进 summary，就只打标记，不物理删除，这样历史记录仍然可以保留给 UI 或审计使用。
	CompactedAt *time.Time `gorm:"column:compacted_at;index:idx_memory_turns_session_id_compacted_at,priority:2"`
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime"`
}

func (memoryTurn) TableName() string {
	return "memory_turns"
}

// memoryFact 用来存从历史对话里抽出来的“稳定事实”。
// 它和原始 turn 分开存，是因为事实记忆的生命周期通常更长，也更适合后续做召回。
type memoryFact struct {
	ID        uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	SessionID string    `gorm:"column:session_id;index:idx_memory_facts_session_id_updated_at,priority:1;uniqueIndex:idx_memory_facts_session_id_hash,priority:1;size:128"`
	Hash      string    `gorm:"column:hash;uniqueIndex:idx_memory_facts_session_id_hash,priority:2;size:64"`
	Content   string    `gorm:"column:content;type:text"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime;index:idx_memory_facts_session_id_updated_at,priority:2"`
}

func (memoryFact) TableName() string {
	return "memory_facts"
}

func NewMySQLStoreFromConfig(ctx context.Context) (Store, error) {
	// 第一步先把配置读进来，避免后面构造 store 时到处穿插配置读取逻辑。
	cfg, err := loadMySQLStoreConfig(ctx)
	if err != nil {
		return nil, err
	}

	// 第二步连 MySQL。后面所有 memory 的权威数据都先落在这里。
	db, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open memory mysql failed: %w", err)
	}

	// AutoMigrate 保证第一次启动时表会自动建出来，不需要手动执行 SQL。
	if err := db.AutoMigrate(&memorySession{}, &memoryTurn{}, &memoryFact{}); err != nil {
		return nil, fmt.Errorf("migrate memory tables failed: %w", err)
	}

	// 摘要器和事实提取器优先走 LLM，失败时会自动降级。
	summarizer, extractor := newLLMMemoryHelpers(ctx, cfg.MaxSummaryChars)
	var semanticStore *semanticMemoryStore
	if cfg.SemanticRecallEnabled {
		// 语义记忆召回失败时不阻塞 memory store 初始化，
		// 这样即使 Milvus 不可用，普通多轮对话仍然可以工作。
		semanticStore, err = newSemanticMemoryStore(ctx, cfg.SemanticCollection, cfg.SemanticTopK)
		if err != nil {
			g.Log().Warningf(ctx, "init semantic memory store failed, fallback to mysql-only memory: %v", err)
		}
	}

	return &mysqlStore{
		db:              db,
		maxWindowSize:   cfg.MaxWindowSize,
		maxSummaryChars: cfg.MaxSummaryChars,
		maxFactCount:    cfg.MaxFactCount,
		summarizer:      summarizer,
		factExtractor:   extractor,
		semanticStore:   semanticStore,
	}, nil
}

func loadMySQLStoreConfig(ctx context.Context) (*mysqlStoreConfig, error) {
	// 这里逐个读取配置项，而不是直接把整段 YAML 映射成 struct，
	// 是沿用项目里现有的 g.Cfg().Get(...) 风格。
	dsnValue, err := g.Cfg().Get(ctx, "memory_store.dsn")
	if err != nil {
		return nil, err
	}
	maxWindowSizeValue, err := g.Cfg().Get(ctx, "memory_store.max_window_size")
	if err != nil {
		return nil, err
	}
	maxSummaryCharsValue, err := g.Cfg().Get(ctx, "memory_store.max_summary_chars")
	if err != nil {
		return nil, err
	}
	maxFactCountValue, err := g.Cfg().Get(ctx, "memory_store.max_fact_count")
	if err != nil {
		return nil, err
	}
	semanticRecallEnabledValue, err := g.Cfg().Get(ctx, "memory_store.semantic_recall_enabled")
	if err != nil {
		return nil, err
	}
	semanticTopKValue, err := g.Cfg().Get(ctx, "memory_store.semantic_top_k")
	if err != nil {
		return nil, err
	}
	semanticCollectionValue, err := g.Cfg().Get(ctx, "memory_store.semantic_collection")
	if err != nil {
		return nil, err
	}

	// 先把所有配置值收敛成一个 struct，后面统一做校验和默认值处理。
	cfg := &mysqlStoreConfig{
		DSN:                   dsnValue.String(),
		MaxWindowSize:         maxWindowSizeValue.Int(),
		MaxSummaryChars:       maxSummaryCharsValue.Int(),
		MaxFactCount:          maxFactCountValue.Int(),
		SemanticRecallEnabled: semanticRecallEnabledValue.Bool(),
		SemanticTopK:          semanticTopKValue.Int(),
		SemanticCollection:    semanticCollectionValue.String(),
	}
	if cfg.DSN == "" {
		return nil, fmt.Errorf("memory_store.dsn is empty")
	}
	// 以下几个 if 都是在做“配置兜底”。
	// 这样即使配置文件没写或者写成非法值，memory 模块也能按默认策略跑起来。
	if cfg.MaxWindowSize <= 0 {
		cfg.MaxWindowSize = 6
	}
	if cfg.MaxSummaryChars <= 0 {
		cfg.MaxSummaryChars = 4000
	}
	if cfg.MaxFactCount <= 0 {
		cfg.MaxFactCount = 12
	}
	if cfg.SemanticTopK <= 0 {
		cfg.SemanticTopK = 4
	}
	if cfg.SemanticCollection == "" {
		cfg.SemanticCollection = "memory_facts"
	}
	return cfg, nil
}

func (s *mysqlStore) GetMessages(ctx context.Context, sessionID string) ([]*schema.Message, error) {
	// 不带 query 的旧接口直接复用新接口，只是不触发语义召回。
	return s.GetMessagesForQuery(ctx, sessionID, "")
}

func (s *mysqlStore) GetMessagesForQuery(ctx context.Context, sessionID, query string) ([]*schema.Message, error) {
	// 先拿会话级摘要。
	var session memorySession
	err := s.db.WithContext(ctx).First(&session, "session_id = ?", sessionID).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("query memory session failed: %w", err)
	}

	// 再拿长期事实：
	// 有 query 时优先走语义召回；没有 query 时退回最近 facts。
	factContents, err := s.loadFactContents(ctx, sessionID, query)
	if err != nil {
		return nil, err
	}

	// 最后拿最近窗口消息。
	// 这样最终 prompt 顺序就是：摘要 -> facts -> recent turns。
	var turns []memoryTurn
	if err := s.db.WithContext(ctx).
		Where("session_id = ? AND compacted_at IS NULL", sessionID).
		Order("id DESC").
		Limit(s.maxWindowSize).
		Find(&turns).Error; err != nil {
		return nil, fmt.Errorf("query memory turns failed: %w", err)
	}

	// 这里的 messages 就是最后喂给模型的历史上下文。
	messages := make([]*schema.Message, 0, len(turns)+1)
	if session.Summary != "" {
		messages = append(messages, schema.SystemMessage(summaryPromptPrefix+session.Summary))
	}
	if len(factContents) > 0 {
		// 长期事实作为一条单独的 system message 注入，
		// 让模型知道“这是历史里稳定成立的信息”，不是用户当前回合的新输入。
		lines := make([]string, 0, len(factContents))
		for _, fact := range factContents {
			if strings.TrimSpace(fact) == "" {
				continue
			}
			lines = append(lines, "- "+fact)
		}
		messages = append(messages, schema.SystemMessage(factsPromptPrefix+strings.Join(lines, "\n")))
	}
	for i := len(turns) - 1; i >= 0; i-- {
		// 数据库里是按 id DESC 查出来的，这里倒序回放，恢复真实对话顺序。
		turn := turns[i]
		msg, err := decodeTurn(turn)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

// AppendMessages Controller 在回答后把 user + assistant 一起写入记忆：见 chat_v1_chat.go (line 56) 和 chat_v1_chat_stream.go (line 54)。
// mem.AppendMessages 进入 MySQL 事务写 memory_turns，同时确保 memory_sessions 存在：见 mysql_store.go (line 280)。
// 如果“未压缩窗口”超过 max_window_size，触发压缩：把最早的消息做增量摘要写回 memory_sessions.summary，并把这些旧消息打 compacted_at 标记（不物理删）：见 mysql_store.go (line 329)。
// 同时从被压缩的旧消息抽取长期事实，幂等写入 memory_facts（按 session_id + hash 去重）：见 mysql_store.go (line 400) 和 mysql_store.go (line 414)。
// 事务提交后，如果开了语义召回，把全量 facts 同步到 Milvus memory_facts collection（float 向量）：见 mysql_store.go (line 319) 和 milvus_memory.go (line 114)。
func (s *mysqlStore) AppendMessages(ctx context.Context, sessionID string, msgs ...*schema.Message) error {
	if len(msgs) == 0 {
		return nil
	}

	// 原始消息的落库、压缩、摘要更新必须放在一个事务里，
	// 否则中间任何一步失败，session 状态都会不一致。
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 如果 session 还不存在，就先建一个空 session。
		if err := tx.FirstOrCreate(&memorySession{SessionID: sessionID}, memorySession{SessionID: sessionID}).Error; err != nil {
			return fmt.Errorf("init memory session failed: %w", err)
		}

		for _, msg := range msgs {
			if msg == nil {
				continue
			}
			// 整个 schema.Message 序列化后存起来，
			// 是为了后面恢复时不只拿到 role/content，也能保住 tool calls 等结构。
			payload, err := json.Marshal(msg)
			if err != nil {
				return fmt.Errorf("marshal memory message failed: %w", err)
			}
			row := memoryTurn{
				SessionID: sessionID,
				Role:      string(msg.Role),
				Content:   msg.Content,
				Payload:   string(payload),
			}
			if err := tx.Create(&row).Error; err != nil {
				return fmt.Errorf("insert memory turn failed: %w", err)
			}
		}

		return s.compactSession(ctx, tx, sessionID)
	}); err != nil {
		return err
	}

	if s.semanticStore != nil {
		// 事务提交成功后再同步向量记忆，避免 MySQL 回滚但 Milvus 已写入造成双写不一致。用sessionid+hash做主键避免重复插入
		if err := s.syncFactsToSemanticStore(ctx, sessionID); err != nil {
			g.Log().Warningf(ctx, "sync semantic memory facts failed: %v", err)
		}
	}

	return nil
}

func (s *mysqlStore) compactSession(ctx context.Context, tx *gorm.DB, sessionID string) error {
	// 先看当前窗口有没有超过阈值。没超就不用压缩。
	var count int64
	if err := tx.WithContext(ctx).Model(&memoryTurn{}).Where("session_id = ? AND compacted_at IS NULL", sessionID).Count(&count).Error; err != nil {
		return fmt.Errorf("count memory turns failed: %w", err)
	}
	if count <= int64(s.maxWindowSize) {
		return nil
	}

	excess := int(count) - s.maxWindowSize
	// 当前 controller 以 user/assistant 成对写入，优先偶数压缩，避免拆散一轮对话。
	if excess%2 != 0 {
		excess++
	}
	// 取出“最早的 excess 条消息”，这些就是要被压进摘要里的旧历史。
	var toCompact []memoryTurn
	if err := tx.WithContext(ctx).
		Where("session_id = ? AND compacted_at IS NULL", sessionID).
		Order("id ASC").
		Limit(excess).
		Find(&toCompact).Error; err != nil {
		return fmt.Errorf("load turns for compaction failed: %w", err)
	}
	if len(toCompact) == 0 {
		return nil
	}
	// 读出当前已有摘要，后面做的是“增量摘要”，不是每次全量重写。
	var session memorySession
	if err := tx.WithContext(ctx).First(&session, "session_id = ?", sessionID).Error; err != nil {
		return fmt.Errorf("load memory session failed: %w", err)
	}
	// oldMessages 是真正要送去做摘要/事实提取的旧消息切片。
	oldMessages := make([]*schema.Message, 0, len(toCompact))
	ids := make([]uint64, 0, len(toCompact))
	// ids 记录的是这些旧 turn 在数据库里的主键，摘要成功后会把它们删掉。
	for _, turn := range toCompact {
		// decodeTurn 会优先从完整 payload 恢复消息，避免只剩 role/content。
		msg, err := decodeTurn(turn)
		if err != nil {
			return err
		}
		oldMessages = append(oldMessages, msg)
		ids = append(ids, turn.ID)
	}
	// 这里把“旧摘要 + 新压缩的一批旧消息”一起交给摘要器，得到新的滚动摘要。
	newSummary, err := s.summarizer.Summarize(ctx, session.Summary, oldMessages)
	if err != nil {
		return fmt.Errorf("summarize old memory failed: %w", err)
	}

	if err := tx.WithContext(ctx).
		Model(&memorySession{}).
		Where("session_id = ?", sessionID).
		Updates(map[string]any{
			"summary":            newSummary,
			"summary_updated_at": time.Now(),
		}).Error; err != nil {
		return fmt.Errorf("update memory summary failed: %w", err)
	}

	// 这里不再物理删除旧 turns，而是只打 compacted_at 标记。
	// 这样 prompt 构建时不会再读到它们，但原始历史消息仍然保留在库里。
	compactedAt := time.Now()
	if err := tx.WithContext(ctx).
		Model(&memoryTurn{}).
		Where("id IN ?", ids).
		Update("compacted_at", compactedAt).Error; err != nil {
		return fmt.Errorf("mark compacted turns failed: %w", err)
	}

	if s.factExtractor != nil {
		// 旧消息被压缩掉之前，先抽取其中稳定、可长期复用的事实，
		// 避免窗口裁剪之后这些信息彻底丢失。
		facts, err := s.factExtractor.Extract(ctx, oldMessages)
		if err != nil {
			g.Log().Warningf(ctx, "extract memory facts failed: %v", err)
		} else if err := s.upsertFacts(ctx, tx, sessionID, facts); err != nil {
			return err
		}
	}

	return nil
}

func (s *mysqlStore) upsertFacts(ctx context.Context, tx *gorm.DB, sessionID string, facts []string) error {
	// facts 可能来自 LLM，也可能有重复、空字符串，所以这里统一做清洗和幂等写入。
	for _, fact := range facts {
		fact = strings.TrimSpace(fact)
		if fact == "" {
			continue
		}
		row := memoryFact{
			SessionID: sessionID,
			// hash 是“同一个 session 内这条事实”的稳定指纹，用来防止重复插入。
			Hash:    hashFact(sessionID, fact),
			Content: fact,
		}

		var existing memoryFact
		// 先查是否已有同 hash 的事实。
		err := tx.WithContext(ctx).First(&existing, "session_id = ? AND hash = ?", sessionID, row.Hash).Error
		if err == nil {
			// 已存在时做更新，不重复插入。
			if err := tx.WithContext(ctx).
				Model(&memoryFact{}).
				Where("id = ?", existing.ID).
				Updates(map[string]any{
					"content":    fact,
					"updated_at": time.Now(),
				}).Error; err != nil {
				return fmt.Errorf("update memory fact failed: %w", err)
			}
			continue
		}
		if err != nil && err != gorm.ErrRecordNotFound {
			return fmt.Errorf("query memory fact failed: %w", err)
		}
		if err := tx.WithContext(ctx).Create(&row).Error; err != nil {
			return fmt.Errorf("insert memory fact failed: %w", err)
		}
	}
	return nil
}

func (s *mysqlStore) loadFactContents(ctx context.Context, sessionID, query string) ([]string, error) {
	if s.semanticStore != nil && strings.TrimSpace(query) != "" {
		// 有 query 时优先做语义召回，这样长期记忆是“按需取用”，
		// 而不是把所有 facts 无差别塞进上下文。
		facts, err := s.semanticStore.QueryFacts(ctx, sessionID, query)
		if err != nil {
			g.Log().Warningf(ctx, "semantic memory recall failed, fallback to recent facts: %v", err)
		} else if len(facts) > 0 {
			return facts, nil
		}
	}

	facts, err := s.listRecentFacts(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	// 数据库里是按 updated_at DESC 查出来的，这里反过来拼回 prompt，让阅读顺序更自然。
	result := make([]string, 0, len(facts))
	for i := len(facts) - 1; i >= 0; i-- {
		result = append(result, facts[i].Content)
	}
	return result, nil
}

func (s *mysqlStore) listRecentFacts(ctx context.Context, sessionID string) ([]memoryFact, error) {
	// 这是“降级路径”用的最近 facts。
	// 语义召回不可用时，就把最近更新过的长期事实直接回放给模型。
	var facts []memoryFact
	if err := s.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("updated_at DESC").
		Limit(s.maxFactCount).
		Find(&facts).Error; err != nil {
		return nil, fmt.Errorf("query memory facts failed: %w", err)
	}
	return facts, nil
}

func (s *mysqlStore) listAllFacts(ctx context.Context, sessionID string) ([]memoryFact, error) {
	// 这是“同步到 Milvus”用的全量 facts，不做 limit。
	var facts []memoryFact
	if err := s.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("updated_at DESC").
		Find(&facts).Error; err != nil {
		return nil, fmt.Errorf("query all memory facts failed: %w", err)
	}
	return facts, nil
}

func (s *mysqlStore) syncFactsToSemanticStore(ctx context.Context, sessionID string) error {
	if s.semanticStore == nil {
		return nil
	}

	// 这里同步全量 facts，而不是只同步最近几条，
	// 因为长期记忆的价值就在于“老事实也能被后续 query 召回”。
	facts, err := s.listAllFacts(ctx, sessionID)
	if err != nil {
		return err
	}
	if len(facts) == 0 {
		return nil
	}
	return s.semanticStore.UpsertFacts(ctx, sessionID, facts)
}

func decodeTurn(turn memoryTurn) (*schema.Message, error) {
	if turn.Payload != "" {
		// 优先从完整 payload 恢复，
		// 因为只靠 role/content 会丢掉 tool calls、extra 等结构化信息。
		var msg schema.Message
		if err := json.Unmarshal([]byte(turn.Payload), &msg); err == nil {
			return &msg, nil
		}
	}

	// 如果 payload 解析失败，就退回最小可用版本，至少保证历史还能被读出来。
	return &schema.Message{
		Role:    schema.RoleType(turn.Role),
		Content: turn.Content,
	}, nil
}

func (s *rollingSummarizer) Summarize(_ context.Context, existing string, msgs []*schema.Message) (string, error) {
	// 规则摘要器很简单：把旧摘要和新消息摘要片段拼起来，再截断到最大长度。
	lines := make([]string, 0, len(msgs)+1)
	if existing != "" {
		lines = append(lines, existing)
	}

	for _, msg := range msgs {
		if msg == nil {
			continue
		}
		snippet := summarizeMessage(msg)
		if snippet == "" {
			continue
		}
		lines = append(lines, snippet)
	}

	merged := strings.Join(lines, "\n")
	if utf8.RuneCountInString(merged) <= s.maxChars {
		return merged, nil
	}

	// 超长时只保留尾部，是因为越新的信息通常越有价值。
	runes := []rune(merged)
	if s.maxChars <= 3 {
		return string(runes[len(runes)-s.maxChars:]), nil
	}
	return "..." + string(runes[len(runes)-(s.maxChars-3):]), nil
}

func summarizeMessage(msg *schema.Message) string {
	if msg == nil {
		return ""
	}

	content := strings.TrimSpace(msg.Content)
	if content == "" && len(msg.ToolCalls) > 0 {
		// assistant 纯工具调用消息通常没有 Content，这里把工具名提炼出来，避免摘要阶段整条丢失。
		names := make([]string, 0, len(msg.ToolCalls))
		for _, toolCall := range msg.ToolCalls {
			names = append(names, toolCall.Function.Name)
		}
		content = "调用工具: " + strings.Join(names, ", ")
	}
	if content == "" {
		return ""
	}

	content = strings.Join(strings.Fields(content), " ")
	if utf8.RuneCountInString(content) > 160 {
		// 单条摘要片段也要截断，避免一条超长消息把 summary 预算吃光。
		runes := []rune(content)
		content = string(runes[:160]) + "..."
	}

	role := "对话"
	switch msg.Role {
	case schema.User:
		role = "用户"
	case schema.Assistant:
		role = "助手"
	case schema.System:
		role = "系统"
	case schema.Tool:
		role = "工具"
	}

	return fmt.Sprintf("- %s: %s", role, content)
}

func hashFact(sessionID, fact string) string {
	// 把 sessionID 也放进 hash，是为了让“不同 session 中相同事实文本”互不影响。
	sum := sha256.Sum256([]byte(sessionID + "\n" + fact))
	return hex.EncodeToString(sum[:])
}

// llmMemoryHelper 同时实现 Summarizer 和 FactExtractor。
// 也就是：同一个 Gemini 模型既负责做摘要，也负责抽取长期事实。
type llmMemoryHelper struct {
	chatModel       model.ToolCallingChatModel
	fallback        Summarizer
	maxSummaryChars int
}

func newLLMMemoryHelpers(ctx context.Context, maxSummaryChars int) (Summarizer, FactExtractor) {
	// fallback 是兜底策略：
	// 就算大模型不可用，最起码还能用规则摘要继续工作。
	fallback := &rollingSummarizer{maxChars: maxSummaryChars}
	chatModel, err := models.GoogleGeminiModel(ctx)
	if err != nil {
		// 这里返回 (fallback, nil) 的意思是：
		// 摘要还能做，但长期事实抽取先关闭。
		return fallback, nil
	}
	helper := &llmMemoryHelper{
		chatModel:       chatModel,
		fallback:        fallback,
		maxSummaryChars: maxSummaryChars,
	}
	return helper, helper
}

func (h *llmMemoryHelper) Summarize(ctx context.Context, existing string, msgs []*schema.Message) (string, error) {
	// 先把消息格式化成更适合喂给 LLM 的纯文本历史。
	history := formatMessagesForMemory(msgs)
	if strings.TrimSpace(history) == "" {
		return existing, nil
	}

	// prompt 里显式告诉模型做“增量摘要”，避免它每次把旧摘要整段重写得面目全非。
	prompt := fmt.Sprintf(`你是一个会话记忆压缩器。
你的任务是把较早的对话内容压缩成可长期保留的历史摘要。

要求：
1. 只保留后续对话真正需要的事实、用户目标、约束条件、重要上下文和关键结论。
2. 删除寒暄、重复表达和无关细节。
3. 输出纯文本，不要使用 markdown，不要输出 JSON。
4. 如果已有历史摘要，请在其基础上增量合并，而不是重复改写所有内容。
5. 控制在 %d 个字符以内。

已有历史摘要：
%s

新增要压缩的历史消息：
%s`, h.maxSummaryChars, blankIfEmpty(existing), history)

	text, err := h.generatePlainText(ctx, prompt)
	if err != nil || strings.TrimSpace(text) == "" {
		// LLM 失败时立刻回退规则摘要，而不是让整条写入链路报错。
		return h.fallback.Summarize(ctx, existing, msgs)
	}
	if utf8.RuneCountInString(text) > h.maxSummaryChars {
		runes := []rune(text)
		text = string(runes[:h.maxSummaryChars])
	}
	return strings.TrimSpace(text), nil
}

func (h *llmMemoryHelper) Extract(ctx context.Context, msgs []*schema.Message) ([]string, error) {
	// 事实抽取也先把消息压成干净的纯文本输入，减少提示词噪音。
	history := formatMessagesForMemory(msgs)
	if strings.TrimSpace(history) == "" {
		return nil, nil
	}

	// 这里强制要求模型输出 JSON，是因为后面要结构化解析成 []string。
	prompt := fmt.Sprintf(`你是一个会话长期记忆提取器。
请从下面的对话里提取值得长期保存的稳定事实。

要求：
1. 只提取稳定、可复用、对后续任务可能有帮助的信息。
2. 不要提取瞬时闲聊、一次性无用细节、无证据猜测。
3. 每条事实必须简洁、独立、可直接复用。
4. 输出 JSON，格式必须是 {"facts":["fact1","fact2"]}。
5. 如果没有值得保留的事实，输出 {"facts":[]}。

对话内容：
%s`, history)

	text, err := h.generatePlainText(ctx, prompt)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Facts []string `json:"facts"`
	}
	// extractJSON 是为了兼容模型前后夹带解释文字的情况，只截取中间的 JSON 主体。
	if err := json.Unmarshal([]byte(extractJSON(text)), &payload); err != nil {
		return nil, err
	}
	return dedupeFacts(payload.Facts), nil
}

func (h *llmMemoryHelper) generatePlainText(ctx context.Context, prompt string) (string, error) {
	// 这里直接把 memory 相关任务当成一轮普通的单消息调用。
	msg, err := h.chatModel.Generate(ctx, []*schema.Message{
		schema.UserMessage(prompt),
	})
	if err != nil {
		return "", err
	}
	if msg == nil {
		return "", fmt.Errorf("memory llm returned nil message")
	}
	return strings.TrimSpace(msg.Content), nil
}

func formatMessagesForMemory(msgs []*schema.Message) string {
	// 统一把复杂的 schema.Message 序列压成适合摘要/事实抽取的文本格式。
	lines := make([]string, 0, len(msgs))
	for _, msg := range msgs {
		snippet := summarizeMessage(msg)
		if snippet != "" {
			lines = append(lines, snippet)
		}
	}
	return strings.Join(lines, "\n")
}

func dedupeFacts(facts []string) []string {
	// 做一次标准化去重，避免“同一句话只是空格不同”被存成多条事实。
	result := make([]string, 0, len(facts))
	seen := make(map[string]struct{}, len(facts))
	for _, fact := range facts {
		normalized := strings.TrimSpace(strings.Join(strings.Fields(fact), " "))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func extractJSON(text string) string {
	// 模型有时会输出“解释 + JSON + 结尾说明”，这里只截取最外层的大括号部分。
	text = strings.TrimSpace(text)
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start >= 0 && end > start {
		return text[start : end+1]
	}
	return text
}

func blankIfEmpty(s string) string {
	// 给 prompt 里的“已有摘要”留一个占位词，避免传空字符串让模型理解不清。
	if strings.TrimSpace(s) == "" {
		return "（无）"
	}
	return s
}
