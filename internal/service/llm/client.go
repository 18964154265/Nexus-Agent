package llm

import (
	"context"
	"encoding/json"
	"fmt"

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
}

// ChatResponse 统一响应结果
// 【优化点】：这里也不要直接返回 openai 类型，而是返回基础类型或自定义类型
// 方便 Runner 处理，不用 import openai 包
type ChatResponse struct {
	Content string
	// ToolCalls 我们需要自定义一个结构，或者暂时透传，但在 Runner 里解析时要注意
	// 为了简单起见，这里先保留 openai.ToolCall，但理想情况应该转换成自定义 DTO
	ToolCalls []openai.ToolCall

	Usage map[string]int // {prompt: 10, completion: 20}
}

// ChatCompletion 调用大模型
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

	usage := map[string]int{
		"prompt_tokens":     resp.Usage.PromptTokens,
		"completion_tokens": resp.Usage.CompletionTokens,
		"total_tokens":      resp.Usage.TotalTokens,
	}

	return &ChatResponse{
		Content:   choice.Message.Content,
		ToolCalls: choice.Message.ToolCalls,
		Usage:     usage,
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
