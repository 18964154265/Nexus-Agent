package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"example.com/agent-server/internal/store" // 引入你的 store 包
	"github.com/sashabaranov/go-openai"
)

// LLMConfig 配置
type LLMConfig struct {
	ApiKey      string
	BaseURL     string
	ModelName   string
	Temperature float32
}

// 定义一个接口，方便 Runner 做 Mock 测试
type Provider interface {
	ChatCompletion(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
}

type Client struct {
	client *openai.Client
	config LLMConfig
}

// NewClient 初始化
func NewClient(cfg LLMConfig) *Client {
	openaiConfig := openai.DefaultConfig(cfg.ApiKey)
	if cfg.BaseURL != "" {
		openaiConfig.BaseURL = cfg.BaseURL
	}
	c := openai.NewClientWithConfig(openaiConfig)
	return &Client{client: c, config: cfg}
}

// ChatRequest 统一请求参数
// 【优化点】：这里使用 store 中的类型，解耦 Runner 和 OpenAI SDK
type ChatRequest struct {
	SystemPrompt string
	UserPrompt   string // 可选，如果 history 里已经包含了最新消息，这里可为空

	History []*store.ChatMessage // 使用数据库模型
	Tools   []*store.MCPTool     // 使用数据库模型

	// 允许 LLM 选择其它 Agent（handoff）
	HandoffCandidates []HandoffCandidate

	// 是否强制提示 LLM 返回携带 handoff 字段的 JSON
	ForceHandoff bool
}

// ChatResponse 统一响应结果
// 【优化点】：这里也不要直接返回 openai 类型，而是返回基础类型或自定义类型
// 方便 Runner 处理，不用 import openai 包
type ChatResponse struct {
	Content string
	// ToolCalls 我们需要自定义一个结构，或者暂时透传，但在 Runner 里解析时要注意
	// 为了简单起见，这里先保留 openai.ToolCall，但理想情况应该转换成自定义 DTO
	ToolCalls []openai.ToolCall

	// handoff 决策；TargetAgentID 为空代表不切换
	Handoff *HandoffDecision

	// 原始 JSON 文本（若 Content 为结构化 JSON）
	RawPayload map[string]interface{}

	Usage map[string]int // {prompt: 10, completion: 20}
}

// HandoffCandidate 描述可供切换的 Agent
type HandoffCandidate struct {
	AgentID     string `json:"agent_id"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

// HandoffDecision 模型返回的切换决策
type HandoffDecision struct {
	TargetAgentID    string                 `json:"target_agent_id"`
	Reason           string                 `json:"reason,omitempty"`
	PreferredServer  string                 `json:"preferred_server,omitempty"` // 允许模型建议 MCP server
	AdditionalFields map[string]interface{} `json:"additional_fields,omitempty"`
}

// ==========================================
// 流式相关结构体
// ==========================================

// StreamEvent 定义流式返回的事件类型
type StreamEvent struct {
	Type    string `json:"type"` // "content", "tool_call", "handoff", "error", "done"
	Content string `json:"content,omitempty"`

	// 完整的工具调用信息（内部拼接完成后才发送）
	ToolCalls []ToolCallInfo `json:"tool_calls,omitempty"`

	// handoff 决策
	Handoff *HandoffDecision `json:"handoff,omitempty"`

	// 错误信息
	Error string `json:"error,omitempty"`
}

// ToolCallInfo 工具调用信息（简化版，不依赖 openai 包）
type ToolCallInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// ChatStream 流式调用大模型，返回 channel
func (c *Client) ChatStream(ctx context.Context, req *ChatRequest) (<-chan StreamEvent, error) {
	messages := c.buildMessages(req)
	tools := c.buildTools(req.Tools)

	apiReq := openai.ChatCompletionRequest{
		Model:       c.config.ModelName,
		Messages:    messages,
		Tools:       tools,
		Temperature: c.config.Temperature,
		Stream:      true,
	}

	stream, err := c.client.CreateChatCompletionStream(ctx, apiReq)
	if err != nil {
		return nil, fmt.Errorf("llm stream error: %w", err)
	}

	ch := make(chan StreamEvent, 10)

	go func() {
		defer close(ch)
		defer stream.Close()

		var contentBuffer strings.Builder
		// toolCallsBuffer: map[index] -> {id, name, argsBuffer}
		toolCallsBuffer := make(map[int]*toolCallAccumulator)

		for {
			chunk, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				ch <- StreamEvent{Type: "error", Error: err.Error()}
				return
			}

			if len(chunk.Choices) == 0 {
				continue
			}

			delta := chunk.Choices[0].Delta
			finishReason := chunk.Choices[0].FinishReason

			// 1. 处理文本内容增量
			if delta.Content != "" {
				contentBuffer.WriteString(delta.Content)
				ch <- StreamEvent{Type: "content", Content: delta.Content}
			}

			// 2. 处理工具调用增量（需要拼接）
			for _, tc := range delta.ToolCalls {
				idx := 0
				if tc.Index != nil {
					idx = *tc.Index
				}

				if _, ok := toolCallsBuffer[idx]; !ok {
					toolCallsBuffer[idx] = &toolCallAccumulator{}
				}
				acc := toolCallsBuffer[idx]

				if tc.ID != "" {
					acc.ID = tc.ID
				}
				if tc.Function.Name != "" {
					acc.Name = tc.Function.Name
				}
				if tc.Function.Arguments != "" {
					acc.Args.WriteString(tc.Function.Arguments)
				}
			}

			// 3. 检查结束原因
			if finishReason == openai.FinishReasonToolCalls {
				// 工具调用结束，发送完整的工具调用信息
				var calls []ToolCallInfo
				for i := 0; i < len(toolCallsBuffer); i++ {
					if acc, ok := toolCallsBuffer[i]; ok {
						calls = append(calls, ToolCallInfo{
							ID:        acc.ID,
							Name:      acc.Name,
							Arguments: acc.Args.String(),
						})
					}
				}
				ch <- StreamEvent{Type: "tool_call", ToolCalls: calls}
				return
			}

			if finishReason == openai.FinishReasonStop || finishReason == "stop" {
				// 正常结束，解析 handoff
				fullContent := contentBuffer.String()
				_, handoff, _ := parseContentAndHandoff(fullContent)

				if handoff != nil && handoff.TargetAgentID != "" {
					ch <- StreamEvent{Type: "handoff", Handoff: handoff}
				} else {
					ch <- StreamEvent{Type: "done"}
				}
				return
			}
		}

		// 流正常结束但没有明确的 finish_reason
		ch <- StreamEvent{Type: "done"}
	}()

	return ch, nil
}

// toolCallAccumulator 用于拼接流式工具调用
type toolCallAccumulator struct {
	ID   string
	Name string
	Args strings.Builder
}

// ChatCompletion 调用大模型（同步版本，保留兼容）
func (c *Client) ChatCompletion(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// 1. 转换 History (Store -> OpenAI)
	messages := c.buildMessages(req)

	// 2. 转换 Tools (Store -> OpenAI)
	tools := c.buildTools(req.Tools)

	// 3. 发起请求
	apiReq := openai.ChatCompletionRequest{
		Model:       c.config.ModelName,
		Messages:    messages,
		Tools:       tools,
		Temperature: c.config.Temperature,
	}

	resp, err := c.client.CreateChatCompletion(ctx, apiReq)
	if err != nil {
		return nil, fmt.Errorf("llm api error: %w", err)
	}

	// 4. 解析结果
	choice := resp.Choices[0]

	text, handoff, raw := parseContentAndHandoff(choice.Message.Content)

	usage := map[string]int{
		"prompt_tokens":     resp.Usage.PromptTokens,
		"completion_tokens": resp.Usage.CompletionTokens,
		"total_tokens":      resp.Usage.TotalTokens,
	}

	return &ChatResponse{
		Content:    text,
		ToolCalls:  choice.Message.ToolCalls,
		Handoff:    handoff,
		RawPayload: raw,
		Usage:      usage,
	}, nil
}

// ==========================================
// 私有辅助方法：负责脏活累活 (Type Conversion)
// ==========================================

func (c *Client) buildMessages(req *ChatRequest) []openai.ChatCompletionMessage {
	var msgs []openai.ChatCompletionMessage

	// 1. System Prompt
	if req.SystemPrompt != "" {
		msgs = append(msgs, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: req.SystemPrompt,
		})
	}

	// 1.1 handoff 规范提示（确保模型总带 handoff 字段）
	if req.ForceHandoff {
		msgs = append(msgs, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: buildHandoffInstruction(req.HandoffCandidates),
		})
	}

	// 2. History
	for _, m := range req.History {
		msg := openai.ChatCompletionMessage{
			Role: m.Role,
		}

		// 处理 Content (store里是map，这里转string)
		msg.Content = extractContent(m.Content)

		// 处理 Tool Calls (Assistant 产生的)
		// 如果你的 store 存了 ToolCalls 结构，需要在这里还原成 openai.ToolCall
		// 这里假设 m.Content 里如果存了 json 格式的 tool_calls，需要特殊处理
		// 这是一个复杂的点，取决于你 ChatMessage 如何存储 ToolCall。
		// 简单起见，如果 Role 是 Tool，需要填 ToolCallID
		if m.Role == "tool" {
			msg.ToolCallID = m.ToolCallID
		}

		msgs = append(msgs, msg)
	}

	// 3. User Prompt (Current Turn)
	if req.UserPrompt != "" {
		msgs = append(msgs, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: req.UserPrompt,
		})
	}

	return msgs
}

func (c *Client) buildTools(dbTools []*store.MCPTool) []openai.Tool {
	if len(dbTools) == 0 {
		return nil
	}
	var res []openai.Tool
	for _, t := range dbTools {
		res = append(res, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema, // JSON Schema 直接透传
			},
		})
	}
	return res
}

// extractContent 容错处理
func extractContent(contentMap map[string]interface{}) string {
	// 尝试取 "text"
	if v, ok := contentMap["text"]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	// 尝试取 "content"
	if v, ok := contentMap["content"]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	// 转 JSON
	if len(contentMap) > 0 {
		b, _ := json.Marshal(contentMap)
		return string(b)
	}
	return ""
}

// parseContentAndHandoff 解析返回文本，提取 handoff
func parseContentAndHandoff(content string) (string, *HandoffDecision, map[string]interface{}) {
	// 期望模型返回 JSON：{"text":"...", "handoff":{...}}
	var wrapper map[string]interface{}
	if err := json.Unmarshal([]byte(content), &wrapper); err != nil {
		// 不是 JSON，直接返回原文
		return content, nil, nil
	}

	var handoff *HandoffDecision
	if hv, ok := wrapper["handoff"]; ok {
		if hb, err := json.Marshal(hv); err == nil {
			tmp := &HandoffDecision{}
			if err := json.Unmarshal(hb, tmp); err == nil {
				// 允许空 TargetAgentID 表示“不切换”
				handoff = tmp
			}
		}
	}

	// text 字段优先，否则尝试 message/content
	text := content
	if tv, ok := wrapper["text"].(string); ok && tv != "" {
		text = tv
	} else if mv, ok := wrapper["message"].(string); ok && mv != "" {
		text = mv
	}

	return text, handoff, wrapper
}

// buildHandoffInstruction 生成系统提示，强制模型输出 handoff 字段
func buildHandoffInstruction(candidates []HandoffCandidate) string {
	var sb strings.Builder
	sb.WriteString("You have the ability to transfer conversations to other agents.\n")
	sb.WriteString("Available transfers:\n\n")

	if len(candidates) == 0 {
		sb.WriteString("  (no available agents; keep target_agent_id empty)\n\n")
	} else {
		for i, c := range candidates {
			sb.WriteString(fmt.Sprintf("%d. %s - %s (%s)\n", i+1, safeName(c.Name), safeDesc(c.Description), c.AgentID))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Each agent has specific expertise. Choose to transfer when:\n")
	sb.WriteString("- The user's question is outside your scope\n")
	sb.WriteString("- Another agent is better suited for the task\n")
	sb.WriteString("- You cannot complete the user's request\n\n")
	sb.WriteString("To transfer, ALWAYS reply with JSON only:\n")
	sb.WriteString("{\"text\": \"...\", \"handoff\": {\"target_agent_id\": \"\", \"reason\": \"\", \"preferred_server\": \"\"}}\n")
	sb.WriteString("Rules:\n")
	sb.WriteString("- target_agent_id empty string => do not transfer.\n")
	sb.WriteString("- preferred_server optional; leave empty if not sure.\n")
	sb.WriteString("- Respond with JSON only, no extra text.")
	return sb.String()
}

func safeName(v string) string {
	if strings.TrimSpace(v) == "" {
		return "Unnamed Agent"
	}
	return v
}

func safeDesc(v string) string {
	if strings.TrimSpace(v) == "" {
		return "No description"
	}
	return v
}
