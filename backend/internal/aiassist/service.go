package aiassist

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"health-checkup/backend/internal/config"

	"gorm.io/gorm"
)

//go:embed knowledge/*.md
var embeddedKnowledgeFS embed.FS

type Service struct {
	cfg        config.Config
	db         *gorm.DB
	once       sync.Once
	docs       []KnowledgeDoc
	loadErr    error
	httpClient *http.Client
}

type KnowledgeDoc struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Source  string `json:"source"`
	Content string `json:"content"`
}

type Citation struct {
	Title   string `json:"title"`
	Source  string `json:"source"`
	Snippet string `json:"snippet"`
}

type ChatResult struct {
	Answer      string     `json:"answer"`
	Citations   []Citation `json:"citations"`
	Mode        string     `json:"mode"`
	UsedModel   bool       `json:"usedModel"`
	KnowledgeOn bool       `json:"knowledgeOn"`
}

type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature float64         `json:"temperature,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message openAIMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type providerConfig struct {
	name      string
	baseURL   string
	apiKey    string
	model     string
}

func New(cfg config.Config, db *gorm.DB) *Service {
	return &Service{
		cfg: cfg,
		db:  db,
		httpClient: &http.Client{Timeout: 45 * time.Second},
	}
}

func (s *Service) Enabled() bool {
	return s != nil && s.cfg.AIEnabled
}

func (s *Service) Chat(ctx context.Context, role, question string) (ChatResult, error) {
	question = strings.TrimSpace(question)
	if question == "" {
		return ChatResult{}, fmt.Errorf("请输入问题")
	}
	docs, err := s.retrieve(question, 4)
	if err != nil {
		return ChatResult{}, err
	}
	citations := make([]Citation, 0, len(docs))
	for _, doc := range docs {
		citations = append(citations, Citation{
			Title:   doc.Title,
			Source:  doc.Source,
			Snippet: summarizeSnippet(doc.Content, question),
		})
	}
	providers := s.providerChain()
	if len(providers) == 0 {
		return ChatResult{
			Answer:      fallbackAnswer(question, role, citations),
			Citations:   citations,
			Mode:        "retrieval_only",
			UsedModel:   false,
			KnowledgeOn: true,
		}, nil
	}
	prompt := fmt.Sprintf("用户问题：%s\n\n可参考资料：\n%s", question, buildKnowledgeContext(citations))
	var lastErr error
	for _, provider := range providers {
		answer, err := s.generateAnswer(ctx, role, provider, prompt)
		if err == nil && strings.TrimSpace(answer) != "" {
			return ChatResult{
				Answer:      answer,
				Citations:   citations,
				Mode:        provider.name,
				UsedModel:   true,
				KnowledgeOn: true,
			}, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("ai model returned empty answer")
	}
	return ChatResult{
		Answer:      fallbackAnswer(question, role, citations),
		Citations:   citations,
		Mode:        "retrieval_fallback",
		UsedModel:   false,
		KnowledgeOn: true,
	}, nil
}

func (s *Service) retrieve(query string, limit int) ([]KnowledgeDoc, error) {
	if err := s.loadKnowledge(); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 4
	}
	terms := extractTerms(query)
	type scoredDoc struct {
		KnowledgeDoc
		Score int
	}
	rows := make([]scoredDoc, 0, len(s.docs))
	for _, doc := range s.docs {
		score := scoreDoc(doc, terms)
		if score <= 0 {
			continue
		}
		rows = append(rows, scoredDoc{KnowledgeDoc: doc, Score: score})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Score == rows[j].Score {
			return rows[i].Title < rows[j].Title
		}
		return rows[i].Score > rows[j].Score
	})
	if len(rows) > limit {
		rows = rows[:limit]
	}
	result := make([]KnowledgeDoc, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.KnowledgeDoc)
	}
	if len(result) == 0 && len(s.docs) > 0 {
		fallbackLimit := min(limit, len(s.docs))
		result = append(result, s.docs[:fallbackLimit]...)
	}
	return result, nil
}

func (s *Service) loadKnowledge() error {
	s.once.Do(func() {
		entries, err := embeddedKnowledgeFS.ReadDir("knowledge")
		if err != nil {
			s.loadErr = err
			return
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			body, err := embeddedKnowledgeFS.ReadFile(filepath.ToSlash(filepath.Join("knowledge", entry.Name())))
			if err != nil {
				s.loadErr = err
				return
			}
			s.docs = append(s.docs, KnowledgeDoc{
				ID:      entry.Name(),
				Title:   markdownTitle(entry.Name(), string(body)),
				Source:  entry.Name(),
				Content: string(body),
			})
		}
	})
	return s.loadErr
}

func (s *Service) generateAnswer(ctx context.Context, role string, provider providerConfig, prompt string) (string, error) {
	body, err := json.Marshal(openAIRequest{
		Model: provider.model,
		Messages: []openAIMessage{
			{Role: "system", Content: systemPrompt(role)},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.2,
	})
	if err != nil {
		return "", err
	}
	endpoint := strings.TrimRight(provider.baseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+provider.apiKey)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var payload openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 {
		if payload.Error != nil && payload.Error.Message != "" {
			return "", fmt.Errorf("%s", payload.Error.Message)
		}
		return "", fmt.Errorf("ai model request failed: %s", resp.Status)
	}
	if len(payload.Choices) == 0 {
		return "", fmt.Errorf("ai model returned empty choices")
	}
	answer := strings.TrimSpace(payload.Choices[0].Message.Content)
	if answer == "" {
		return "", fmt.Errorf("ai model returned empty answer")
	}
	return answer, nil
}

func (s *Service) providerChain() []providerConfig {
	providers := []providerConfig{}
	primary := strings.ToLower(strings.TrimSpace(s.cfg.AIPrimaryProvider))
	deepseek := providerConfig{name: "deepseek", baseURL: s.cfg.AIDeepSeekBaseURL, apiKey: s.cfg.AIDeepSeekAPIKey, model: s.cfg.AIDeepSeekModel}
	gemini := providerConfig{name: "gemini", baseURL: s.cfg.AIGeminiBaseURL, apiKey: s.cfg.AIGeminiAPIKey, model: s.cfg.AIGeminiModel}
	switch primary {
	case "gemini":
		providers = append(providers, gemini, deepseek)
	default:
		providers = append(providers, deepseek, gemini)
	}
	filtered := make([]providerConfig, 0, len(providers))
	for _, provider := range providers {
		if provider.ready() {
			filtered = append(filtered, provider)
		}
	}
	return filtered
}

func (p providerConfig) ready() bool {
	return strings.TrimSpace(p.baseURL) != "" && strings.TrimSpace(p.apiKey) != "" && strings.TrimSpace(p.model) != ""
}

func systemPrompt(role string) string {
	return fmt.Sprintf("你是熙心健康体检管理系统的 AI 助手。当前提问者角色是：%s。请优先依据提供的项目知识、规则和数据字典回答。不要编造系统没有的功能、字段或流程。如果资料不足，请明确说明依据有限。回答使用简体中文，直接给结论，再给必要说明。", roleLabel(role))
}

func roleLabel(role string) string {
	switch role {
	case "admin":
		return "管理员"
	case "doctor":
		return "医生"
	case "user":
		return "用户"
	default:
		return "访客"
	}
}

func buildKnowledgeContext(citations []Citation) string {
	if len(citations) == 0 {
		return "暂无命中文档。"
	}
	parts := make([]string, 0, len(citations))
	for i, item := range citations {
		parts = append(parts, fmt.Sprintf("[%d] %s (%s)\n%s", i+1, item.Title, item.Source, item.Snippet))
	}
	return strings.Join(parts, "\n\n")
}

func fallbackAnswer(question, role string, citations []Citation) string {
	var builder strings.Builder
	builder.WriteString("当前 AI 模型未配置或本次调用失败，我先按已收录资料给你整理结论。")
	if len(citations) == 0 {
		builder.WriteString("\n\n目前知识库里没有直接命中的内容，建议补充相关业务文档后再问一次。")
		return builder.String()
	}
	builder.WriteString("\n\n问题：")
	builder.WriteString(question)
	builder.WriteString("\n提问角色：")
	builder.WriteString(roleLabel(role))
	builder.WriteString("\n\n命中资料：")
	for i, item := range citations {
		builder.WriteString(fmt.Sprintf("\n%d. %s：%s", i+1, item.Title, item.Snippet))
	}
	builder.WriteString("\n\n如果你希望我给出更自然的总结，需要配置 DeepSeek 或 Gemini 的 API 参数。")
	return builder.String()
}

func markdownTitle(filename, body string) string {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			return strings.TrimSpace(strings.TrimLeft(line, "#"))
		}
	}
	return strings.TrimSuffix(filename, filepath.Ext(filename))
}

var punctuationPattern = regexp.MustCompile(`[\p{P}\p{S}\s]+`)

func extractTerms(query string) []string {
	normalized := strings.ToLower(strings.TrimSpace(query))
	normalized = punctuationPattern.ReplaceAllString(normalized, " ")
	fields := strings.Fields(normalized)
	seen := make(map[string]struct{}, len(fields))
	terms := make([]string, 0, len(fields)+8)
	for _, field := range fields {
		if utf8.RuneCountInString(field) >= 2 {
			if _, ok := seen[field]; !ok {
				seen[field] = struct{}{}
				terms = append(terms, field)
			}
		}
		runes := []rune(field)
		for i := 0; i+1 < len(runes); i++ {
			gram := string(runes[i : i+2])
			if _, ok := seen[gram]; ok {
				continue
			}
			seen[gram] = struct{}{}
			terms = append(terms, gram)
		}
	}
	return terms
}

func scoreDoc(doc KnowledgeDoc, terms []string) int {
	content := strings.ToLower(doc.Title + "\n" + doc.Content)
	score := 0
	for _, term := range terms {
		if term == "" {
			continue
		}
		count := strings.Count(content, term)
		if count == 0 {
			continue
		}
		weight := 2
		if strings.Contains(strings.ToLower(doc.Title), term) {
			weight = 5
		}
		score += count * weight
	}
	return score
}

func summarizeSnippet(content, query string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return "-"
	}
	plain := strings.ReplaceAll(content, "\n", " ")
	plain = strings.Join(strings.Fields(plain), " ")
	terms := extractTerms(query)
	bestIndex := 0
	lowerPlain := strings.ToLower(plain)
	for _, term := range terms {
		if idx := strings.Index(lowerPlain, term); idx >= 0 {
			bestIndex = idx
			break
		}
	}
	runes := []rune(plain)
	start := max(0, bestIndex-40)
	end := min(len(runes), start+140)
	snippet := strings.TrimSpace(string(runes[start:end]))
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(runes) {
		snippet += "..."
	}
	return snippet
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
