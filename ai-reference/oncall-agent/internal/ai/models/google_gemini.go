// Package models 提供基于官方 Google Gemini Content API 的 Eino ChatModel 适配层。
package models

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/google/uuid"
)

const (
	// 用在 Eino message.Extra / toolCall.Extra 里的自定义 key。
	// 当前项目的 Eino 版本还没有单独的 Signature 字段，所以先挂在 Extra 上透传。
	geminiThoughtSignature = "gemini_thought_signature"
)

// GeminiChatModelConfig 是这个适配层自己的初始化配置。
// 这里不直接暴露 Gemini SDK，而是保持为项目内部可控的最小配置集合。
type GeminiChatModelConfig struct {
	APIKey     string
	BaseURL    string
	Model      string
	Timeout    time.Duration
	HTTPClient *http.Client
}

// GeminiChatModel 负责把 Eino 的通用 ChatModel 接口翻译成 Gemini Content API 请求。
// 外层 ReAct Agent 只知道它是一个 ToolCallingChatModel，不关心底层是不是 Gemini。
type GeminiChatModel struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
	// tools 是通过 WithTools 绑定进来的工具声明。
	// ReAct 在创建 agent 时会调用 WithTools，把可用工具传进来。
	tools []*schema.ToolInfo
}

var _ model.ToolCallingChatModel = (*GeminiChatModel)(nil)

func GoogleGeminiModel(ctx context.Context) (model.ToolCallingChatModel, error) {
	// 复用项目现有配置结构 quick_chat_model.*，这样 chat_pipeline 只切模型实现，不改配置入口。
	modelValue, err := g.Cfg().Get(ctx, "quick_chat_model.model")
	if err != nil {
		return nil, err
	}
	apiKeyValue, err := g.Cfg().Get(ctx, "quick_chat_model.api_key")
	if err != nil {
		return nil, err
	}
	baseURLValue, err := g.Cfg().Get(ctx, "quick_chat_model.base_url")
	if err != nil {
		return nil, err
	}

	return NewGoogleGeminiChatModel(ctx, &GeminiChatModelConfig{
		Model:   modelValue.String(),
		APIKey:  apiKeyValue.String(),
		BaseURL: baseURLValue.String(),
		Timeout: 1200 * time.Second,
	})
}

func NewGoogleGeminiChatModel(_ context.Context, config *GeminiChatModelConfig) (*GeminiChatModel, error) {
	if config == nil {
		return nil, fmt.Errorf("gemini config is nil")
	}
	if config.Model == "" {
		return nil, fmt.Errorf("gemini model is empty")
	}
	if config.APIKey == "" {
		return nil, fmt.Errorf("gemini api key is empty")
	}

	// 去掉末尾的 "/"，避免后面拼 endpoint 时出现 "//v1beta/..."。
	baseURL := strings.TrimRight(config.BaseURL, "/")

	httpClient := config.HTTPClient
	if httpClient == nil {
		// 允许外部传自定义 httpClient；不传的话用一个带超时的默认 client。
		timeout := config.Timeout
		if timeout <= 0 {
			timeout = 120 * time.Second
		}
		httpClient = &http.Client{Timeout: timeout}
	}

	return &GeminiChatModel{
		apiKey:     config.APIKey,
		baseURL:    baseURL,
		model:      config.Model,
		httpClient: httpClient,
	}, nil
}

func (m *GeminiChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	// Eino 的 opts 是通用模型选项，这里先提取成公共结构，再映射到 Gemini 请求体。
	commonOpts := model.GetCommonOptions(nil, opts...)
	//将请求转换成gemini请求
	reqBody, err := m.buildGenerateRequest(input, commonOpts)
	if err != nil {
		return nil, err
	}

	// 当前走的是 Gemini Content API 的 generateContent 接口，而不是 OpenAI 兼容接口。
	endpoint := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", m.baseURL, m.model, m.apiKey)
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	// 对 429/5xx 做有限次重试，避免服务端高峰期直接把整个 ReAct 流程打崩。
	respBody, err := m.doGenerateContentWithRetry(ctx, endpoint, payload)
	if err != nil {
		return nil, err
	}

	var apiResp geminiGenerateContentResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, err
	}

	return convertGeminiResponse(&apiResp)
}

// doGenerateContentWithRetry 是一个很薄的重试包装。
// 这里只兜底瞬时错误，不做无限重试，避免 agent 长时间卡死。
func (m *GeminiChatModel) doGenerateContentWithRetry(ctx context.Context, endpoint string, payload []byte) ([]byte, error) {
	const maxAttempts = 4

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		respBody, retry, err := m.doGenerateContentOnce(ctx, endpoint, payload)
		if err == nil {
			return respBody, nil
		}
		lastErr = err
		if !retry || attempt == maxAttempts {
			break
		}

		// 0.5s / 2s / 4.5s 这种简单二次退避已经够用，逻辑比指数退避更直观。
		backoff := time.Duration(attempt*attempt) * 500 * time.Millisecond
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}

	return nil, lastErr
}

// doGenerateContentOnce 只负责发送一次 HTTP 请求，并告诉上层“这个错误值不值得重试”。
func (m *GeminiChatModel) doGenerateContentOnce(ctx context.Context, endpoint string, payload []byte) ([]byte, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, err
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return respBody, false, nil
	}

	// 这些状态码通常是瞬时问题：限流、服务端错误、网关错误、临时不可用、超时。
	retryable := resp.StatusCode == http.StatusTooManyRequests ||
		resp.StatusCode == http.StatusInternalServerError ||
		resp.StatusCode == http.StatusBadGateway ||
		resp.StatusCode == http.StatusServiceUnavailable ||
		resp.StatusCode == http.StatusGatewayTimeout

	return nil, retryable, fmt.Errorf("gemini generateContent failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(respBody)))
}

func (m *GeminiChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	commonOpts := model.GetCommonOptions(nil, opts...)
	reqBody, err := m.buildGenerateRequest(input, commonOpts)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s", m.baseURL, m.model, m.apiKey)
	payload, err := json.Marshal(reqBody)

	if err != nil {
		return nil, err
	}

	resp, err := m.doStreamGenerateContentWithRetry(ctx, endpoint, payload)
	if err != nil {
		return nil, err
	}

	sr, sw := schema.Pipe[*schema.Message](16)
	go m.streamGeminiResponse(resp, sw)

	return sr, nil
}

func (m *GeminiChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	// WithTools 必须返回“新实例”，而不是原地改当前 model。
	// 这是 Eino ToolCallingChatModel 的并发安全约定。
	cp := *m
	if tools == nil {
		cp.tools = nil
	} else {
		cp.tools = append([]*schema.ToolInfo(nil), tools...)
	}
	return &cp, nil
}

func (m *GeminiChatModel) doStreamGenerateContentWithRetry(ctx context.Context, endpoint string, payload []byte) (*http.Response, error) {
	const maxAttempts = 4

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, retry, err := m.doStreamGenerateContentOnce(ctx, endpoint, payload)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if !retry || attempt == maxAttempts {
			break
		}

		backoff := time.Duration(attempt*attempt) * 500 * time.Millisecond
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}

	return nil, lastErr
}

func (m *GeminiChatModel) doStreamGenerateContentOnce(ctx context.Context, endpoint string, payload []byte) (*http.Response, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, true, err
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return resp, false, nil
	}
	defer resp.Body.Close()

	respBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		retryable := resp.StatusCode == http.StatusTooManyRequests ||
			resp.StatusCode == http.StatusInternalServerError ||
			resp.StatusCode == http.StatusBadGateway ||
			resp.StatusCode == http.StatusServiceUnavailable ||
			resp.StatusCode == http.StatusGatewayTimeout
		return nil, retryable, fmt.Errorf("gemini streamGenerateContent failed: status=%d read_body_err=%w", resp.StatusCode, readErr)
	}

	retryable := resp.StatusCode == http.StatusTooManyRequests ||
		resp.StatusCode == http.StatusInternalServerError ||
		resp.StatusCode == http.StatusBadGateway ||
		resp.StatusCode == http.StatusServiceUnavailable ||
		resp.StatusCode == http.StatusGatewayTimeout

	return nil, retryable, fmt.Errorf("gemini streamGenerateContent failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(respBody)))
}

func (m *GeminiChatModel) streamGeminiResponse(resp *http.Response, sw *schema.StreamWriter[*schema.Message]) {
	defer sw.Close()
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)

	state := newGeminiStreamState()
	var dataLines []string

	flushEvent := func() bool {
		if len(dataLines) == 0 {
			return false
		}

		payload := strings.TrimSpace(strings.Join(dataLines, "\n"))
		dataLines = dataLines[:0]
		if payload == "" {
			return false
		}
		if payload == "[DONE]" {
			return true
		}

		var chunk geminiGenerateContentResponse
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			sw.Send(nil, fmt.Errorf("unmarshal gemini stream chunk failed: %w", err))
			return true
		}

		msgs, err := state.consume(&chunk)
		if err != nil {
			sw.Send(nil, err)
			return true
		}
		for _, msg := range msgs {
			if closed := sw.Send(msg, nil); closed {
				return true
			}
		}
		return false
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if flushEvent() {
				return
			}
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}

	if err := scanner.Err(); err != nil {
		sw.Send(nil, fmt.Errorf("read gemini stream chunk failed: %w", err))
		return
	}
	if flushEvent() {
		return
	}

	if finalMsg := state.finalMessage(); finalMsg != nil {
		sw.Send(finalMsg, nil)
	}
}

type geminiStreamState struct {
	toolCallIDs      map[int]string
	signature        string
	signatureEmitted bool
}

func newGeminiStreamState() *geminiStreamState {
	return &geminiStreamState{
		toolCallIDs: make(map[int]string),
	}
}

func (s *geminiStreamState) consume(resp *geminiGenerateContentResponse) ([]*schema.Message, error) {
	if resp == nil || len(resp.Candidates) == 0 {
		return nil, nil
	}

	candidate := resp.Candidates[0]
	messages := make([]*schema.Message, 0, len(candidate.Content.Parts)+1)

	for idx, part := range candidate.Content.Parts {
		if part.ThoughtSignature != "" {
			s.signature = part.ThoughtSignature
		}

		msg := schema.AssistantMessage("", nil)
		if part.Thought {
			msg.ReasoningContent = part.Text
		} else if part.Text != "" {
			msg.Content = part.Text
		}
		if part.FunctionCall != nil {
			argsJSON := "{}"
			if part.FunctionCall.Args != nil {
				raw, err := json.Marshal(part.FunctionCall.Args)
				if err != nil {
					return nil, fmt.Errorf("marshal gemini stream tool arguments failed: %w", err)
				}
				argsJSON = string(raw)
			}

			callID, ok := s.toolCallIDs[idx]
			if !ok {
				callID = uuid.NewString()
				s.toolCallIDs[idx] = callID
			}

			tc := schema.ToolCall{
				Index: ptrInt(idx),
				ID:    callID,
				Type:  "function",
				Function: schema.FunctionCall{
					Name:      part.FunctionCall.Name,
					Arguments: argsJSON,
				},
			}
			if part.ThoughtSignature != "" {
				tc.Extra = map[string]any{
					geminiThoughtSignature: part.ThoughtSignature,
				}
			}
			msg.ToolCalls = []schema.ToolCall{tc}
		}

		if msg.Content == "" && msg.ReasoningContent == "" && len(msg.ToolCalls) == 0 {
			continue
		}
		messages = append(messages, msg)
	}

	if candidate.FinishReason != "" || resp.UsageMetadata != nil {
		msg := schema.AssistantMessage("", nil)
		msg.ResponseMeta = &schema.ResponseMeta{
			FinishReason: candidate.FinishReason,
		}
		if resp.UsageMetadata != nil {
			msg.ResponseMeta.Usage = &schema.TokenUsage{
				PromptTokens:     resp.UsageMetadata.PromptTokenCount,
				CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
				TotalTokens:      resp.UsageMetadata.TotalTokenCount,
			}
		}
		if s.signature != "" {
			msg.Extra = map[string]any{
				geminiThoughtSignature: s.signature,
			}
			s.signatureEmitted = true
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

func (s *geminiStreamState) finalMessage() *schema.Message {
	if s.signature == "" || s.signatureEmitted {
		return nil
	}

	msg := schema.AssistantMessage("", nil)
	msg.Extra = map[string]any{
		geminiThoughtSignature: s.signature,
	}
	return msg
}

// buildGenerateRequest 是整个适配层最核心的函数：
// 负责把 Eino 的 Message 历史 + tool 信息，翻译成 Gemini 的 generateContent 请求体。
func (m *GeminiChatModel) buildGenerateRequest(messages []*schema.Message, opts *model.Options) (*geminiGenerateContentRequest, error) {
	//这个是user用的内容
	req := &geminiGenerateContentRequest{
		Contents: make([]geminiContent, 0, len(messages)),
	}

	// Gemini 的 system prompt 独立放在 systemInstruction 里，
	// 不像很多 OpenAI 风格接口那样把 system message 混在普通 messages 数组里。
	systemTexts := make([]string, 0, 1)
	for _, msg := range messages {
		if msg == nil {
			continue
		}

		switch msg.Role {
		case schema.System:
			if msg.Content != "" {
				systemTexts = append(systemTexts, msg.Content)
			}
		case schema.User:
			// Eino user message -> Gemini user content。
			req.Contents = append(req.Contents, geminiContent{
				Role:  "user",
				Parts: []geminiPart{{Text: msg.Content}},
			})
		case schema.Assistant:
			// Eino assistant message -> Gemini model content。
			// 这里除了普通文本，还要把 reasoning / tool call / thought_signature 一起带过去。
			content := geminiContent{
				Role:  "model",
				Parts: make([]geminiPart, 0, 1+len(msg.ToolCalls)),
			}

			// 如果上一轮模型返回过 thought_signature，先回传一个“空 thought part”。
			// 这样 Gemini 才能把上一次的推理上下文接起来。
			if sig, ok := getThoughtSignature(msg.Extra); ok {
				content.Parts = append(content.Parts, geminiPart{
					Thought:          true,
					ThoughtSignature: sig,
				})
			}
			// Eino v0.6.0 只有 ReasoningContent，没有独立 reasoning part 结构，
			// 所以这里手动映射成 Gemini 的 thought part。
			if msg.ReasoningContent != "" {
				content.Parts = append(content.Parts, geminiPart{
					Text:             msg.ReasoningContent,
					Thought:          true,
					ThoughtSignature: getThoughtSignatureOrEmpty(msg.Extra),
				})
			}
			if msg.Content != "" {
				content.Parts = append(content.Parts, geminiPart{Text: msg.Content})
			}
			for _, tc := range msg.ToolCalls {
				var args map[string]any
				if strings.TrimSpace(tc.Function.Arguments) != "" {
					// Eino ToolCall 里 arguments 是 JSON string，
					// Gemini functionCall 里要求的是对象，所以这里要反序列化一次。
					if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
						return nil, fmt.Errorf("unmarshal tool arguments for %s failed: %w", tc.Function.Name, err)
					}
				}

				part := geminiPart{
					FunctionCall: &geminiFunctionCall{
						Name: tc.Function.Name,
						Args: args,
					},
				}
				// toolCall 自己有签名就优先用自己的；
				// 没有的话，退回到整个 assistant message 的签名。
				if sig, ok := getThoughtSignature(tc.Extra); ok {
					part.ThoughtSignature = sig
				}
				content.Parts = append(content.Parts, part)
			}
			if len(content.Parts) > 0 {
				req.Contents = append(req.Contents, content)
			}
		case schema.Tool:
			// 工具执行结果在 Gemini 协议里不是单纯的 role=tool 文本，
			// 而是 user 侧的一段 functionResponse。
			response := map[string]any{
				"content": msg.Content,
			}
			req.Contents = append(req.Contents, geminiContent{
				Role: "user",
				Parts: []geminiPart{{
					FunctionResponse: &geminiFunctionResponse{
						Name:     msg.ToolName,
						Response: response,
					},
				}},
			})
		}
	}

	if len(systemTexts) > 0 {
		// 多条 system message 合并成一段，避免丢失上游 prompt 模板内容。
		req.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{
				Text: strings.Join(systemTexts, "\n\n"),
			}},
		}
	}

	toolList := m.tools
	if len(opts.Tools) > 0 {
		// 调用期传入的 tools 优先级更高，覆盖实例上绑定的 tools。
		toolList = opts.Tools
	}
	if len(toolList) > 0 {
		decls, err := convertToolDeclarations(toolList)
		if err != nil {
			return nil, err
		}
		if len(decls) > 0 {
			req.Tools = []geminiTool{{FunctionDeclarations: decls}}
			req.ToolConfig = buildToolConfig(opts.ToolChoice)
		}
	}

	// generationConfig 对应 Gemini 的生成参数。
	req.GenerationConfig = &geminiGenerationConfig{}
	req.GenerationConfig.ThinkingConfig = &geminiThinkingConfig{
		// 打开这个开关，Gemini 才可能返回 thought / thoughtSignature。
		IncludeThoughts: true,
	}
	if opts.Temperature != nil {
		req.GenerationConfig.Temperature = opts.Temperature
	}
	if opts.TopP != nil {
		req.GenerationConfig.TopP = opts.TopP
	}
	if opts.MaxTokens != nil {
		req.GenerationConfig.MaxOutputTokens = opts.MaxTokens
	}
	if len(opts.Stop) > 0 {
		req.GenerationConfig.StopSequences = opts.Stop
	}
	if isZeroGenerationConfig(req.GenerationConfig) {
		req.GenerationConfig = nil
	}

	return req, nil
}

// buildToolConfig 把 Eino 的 ToolChoice 映射成 Gemini 的 function calling mode。
func buildToolConfig(choice *schema.ToolChoice) *geminiToolConfig {
	if choice == nil {
		return nil
	}

	mode := "AUTO"
	switch *choice {
	case schema.ToolChoiceForbidden:
		mode = "NONE"
	case schema.ToolChoiceAllowed:
		mode = "AUTO"
	case schema.ToolChoiceForced:
		mode = "ANY"
	}

	return &geminiToolConfig{
		FunctionCallingConfig: &geminiFunctionCallingConfig{
			Mode: mode,
		},
	}
}

// convertToolDeclarations 把 Eino 的 ToolInfo 转成 Gemini functionDeclarations。
// 这里最大的坑是：Eino 导出的 JSON Schema 比 Gemini 接受的 schema 更宽松，
// 所以后面必须再过一遍 sanitize。
func convertToolDeclarations(tools []*schema.ToolInfo) ([]geminiFunctionDeclaration, error) {
	result := make([]geminiFunctionDeclaration, 0, len(tools))
	for _, toolInfo := range tools {
		if toolInfo == nil {
			continue
		}

		var params map[string]any
		if toolInfo.ParamsOneOf != nil {
			jsonSchema, err := toolInfo.ToJSONSchema()
			if err != nil {
				return nil, err
			}
			if jsonSchema != nil {
				// 先借助 json marshal/unmarshal，把第三方 schema struct 打平成 map，
				// 这样后面可以统一递归裁剪字段。
				raw, err := json.Marshal(jsonSchema)
				if err != nil {
					return nil, err
				}
				if err := json.Unmarshal(raw, &params); err != nil {
					return nil, err
				}
				params = sanitizeGeminiSchemaMap(params)
			}
		}

		result = append(result, geminiFunctionDeclaration{
			Name:        toolInfo.Name,
			Description: toolInfo.Desc,
			Parameters:  params,
		})
	}
	return result, nil
}

// sanitizeGeminiSchemaMap 会递归删掉 Gemini 不认识的 schema 字段。
// 目前最关键的是 additionalProperties；不删的话 generateContent 会直接 400。
func sanitizeGeminiSchemaMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return in
	}

	out := make(map[string]any, len(in))
	for k, v := range in {
		switch k {
		case "additionalProperties", "$schema", "$defs", "definitions":
			continue
		default:
			out[k] = sanitizeGeminiSchemaValue(v)
		}
	}
	return out
}

// 对 map / array 递归处理，其他标量值原样返回。
func sanitizeGeminiSchemaValue(v any) any {
	switch vv := v.(type) {
	case map[string]any:
		return sanitizeGeminiSchemaMap(vv)
	case []any:
		out := make([]any, 0, len(vv))
		for _, item := range vv {
			out = append(out, sanitizeGeminiSchemaValue(item))
		}
		return out
	default:
		return v
	}
}

// convertGeminiResponse 把 Gemini 返回的 candidate 重新组装成 Eino 的 schema.Message。
// 这样上层 ReAct Agent 无需感知 Gemini 协议细节。
func convertGeminiResponse(resp *geminiGenerateContentResponse) (*schema.Message, error) {
	if resp == nil || len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("gemini returned no candidates")
	}

	candidate := resp.Candidates[0]
	msg := schema.AssistantMessage("", nil)
	msg.ResponseMeta = &schema.ResponseMeta{
		FinishReason: candidate.FinishReason,
	}

	if resp.UsageMetadata != nil {
		msg.ResponseMeta.Usage = &schema.TokenUsage{
			PromptTokens:     resp.UsageMetadata.PromptTokenCount,
			CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      resp.UsageMetadata.TotalTokenCount,
		}
	}

	var contentBuilder strings.Builder
	var reasoningBuilder strings.Builder
	toolCalls := make([]schema.ToolCall, 0)
	// 记录最后一个看到的 thought_signature，并挂到 msg.Extra 上，供下一轮继续传回 Gemini。
	var signature string

	for idx, part := range candidate.Content.Parts {
		if part.ThoughtSignature != "" {
			signature = part.ThoughtSignature
		}

		if part.Thought {
			// thought part -> Eino 的 ReasoningContent
			reasoningBuilder.WriteString(part.Text)
			continue
		}
		if part.Text != "" {
			contentBuilder.WriteString(part.Text)
		}
		if part.FunctionCall != nil {
			// Gemini functionCall.args 是对象，Eino ToolCall.Arguments 要求 JSON string，
			// 所以这里要再编码回字符串。
			argsJSON := "{}"
			if part.FunctionCall.Args != nil {
				raw, err := json.Marshal(part.FunctionCall.Args)
				if err != nil {
					return nil, err
				}
				argsJSON = string(raw)
			}

			toolCall := schema.ToolCall{
				Index: ptrInt(idx),
				ID:    uuid.NewString(),
				Type:  "function",
				Function: schema.FunctionCall{
					Name:      part.FunctionCall.Name,
					Arguments: argsJSON,
				},
			}
			if part.ThoughtSignature != "" {
				// toolCall 也挂一份签名，方便后面如果需要做更细粒度透传。
				toolCall.Extra = map[string]any{
					geminiThoughtSignature: part.ThoughtSignature,
				}
			}
			toolCalls = append(toolCalls, toolCall)
		}
	}

	msg.Content = contentBuilder.String()
	msg.ReasoningContent = reasoningBuilder.String()
	msg.ToolCalls = toolCalls
	if signature != "" {
		msg.Extra = map[string]any{
			geminiThoughtSignature: signature,
		}
	}

	return msg, nil
}

// 从 Extra 里读出上一轮缓存的 thought_signature。
func getThoughtSignature(extra map[string]any) (string, bool) {
	if len(extra) == 0 {
		return "", false
	}
	value, ok := extra[geminiThoughtSignature]
	if !ok {
		return "", false
	}
	s, ok := value.(string)
	return s, ok && s != ""
}

func getThoughtSignatureOrEmpty(extra map[string]any) string {
	s, _ := getThoughtSignature(extra)
	return s
}

// 如果 generationConfig 里一个有效字段都没有，就不必发这个对象。
// 这样请求体更干净，也避免某些 provider 对空对象行为不一致。
func isZeroGenerationConfig(cfg *geminiGenerationConfig) bool {
	if cfg == nil {
		return true
	}
	return cfg.Temperature == nil &&
		cfg.TopP == nil &&
		cfg.MaxOutputTokens == nil &&
		len(cfg.StopSequences) == 0 &&
		cfg.ThinkingConfig == nil
}

func ptrInt(v int) *int {
	return &v
}

// 下面这些 struct 基本都是 Gemini Content API 请求/响应体的镜像定义。
// 它们的作用只有一个：让 Go 代码能稳定地和 Gemini JSON 协议互转。
type geminiGenerateContentRequest struct {
	SystemInstruction *geminiContent          `json:"systemInstruction,omitempty"`
	Contents          []geminiContent         `json:"contents"`
	Tools             []geminiTool            `json:"tools,omitempty"`
	ToolConfig        *geminiToolConfig       `json:"toolConfig,omitempty"`
	GenerationConfig  *geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text             string                  `json:"text,omitempty"`
	Thought          bool                    `json:"thought,omitempty"`
	ThoughtSignature string                  `json:"thoughtSignature,omitempty"`
	FunctionCall     *geminiFunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *geminiFunctionResponse `json:"functionResponse,omitempty"`
}

type geminiFunctionCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args,omitempty"`
}

type geminiFunctionResponse struct {
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
}

type geminiTool struct {
	FunctionDeclarations []geminiFunctionDeclaration `json:"functionDeclarations,omitempty"`
}

type geminiFunctionDeclaration struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type geminiToolConfig struct {
	FunctionCallingConfig *geminiFunctionCallingConfig `json:"functionCallingConfig,omitempty"`
}

type geminiFunctionCallingConfig struct {
	Mode string `json:"mode,omitempty"`
}

type geminiGenerationConfig struct {
	Temperature     *float32              `json:"temperature,omitempty"`
	TopP            *float32              `json:"topP,omitempty"`
	MaxOutputTokens *int                  `json:"maxOutputTokens,omitempty"`
	StopSequences   []string              `json:"stopSequences,omitempty"`
	ThinkingConfig  *geminiThinkingConfig `json:"thinkingConfig,omitempty"`
}

type geminiThinkingConfig struct {
	IncludeThoughts bool `json:"includeThoughts,omitempty"`
}

type geminiGenerateContentResponse struct {
	Candidates    []geminiCandidate    `json:"candidates"`
	UsageMetadata *geminiUsageMetadata `json:"usageMetadata,omitempty"`
}

type geminiCandidate struct {
	Content      geminiContent `json:"content"`
	FinishReason string        `json:"finishReason,omitempty"`
}

type geminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount,omitempty"`
	CandidatesTokenCount int `json:"candidatesTokenCount,omitempty"`
	TotalTokenCount      int `json:"totalTokenCount,omitempty"`
}
