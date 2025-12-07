package llm

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

// LLMConfig 配置
type LLMConfig struct {
	ApiKey      string
	BaseURL     string // SiliconFlow: "https://api.siliconflow.cn/v1"
	ModelName   string // e.g., "tencent/hunyuan-standard"
	Temperature float32
}

type Client struct {
	client *openai.Client
	config LLMConfig
}

// NewClient 初始化
func NewClient(cfg LLMConfig) *Client {
	// 配置 OpenAI Client，但指向 SiliconFlow 的地址
	openaiConfig := openai.DefaultConfig(cfg.ApiKey)
	openaiConfig.BaseURL = cfg.BaseURL

	c := openai.NewClientWithConfig(openaiConfig)

	return &Client{
		client: c,
		config: cfg,
	}
}

// ChatRequest 统一请求参数
type ChatRequest struct {
	SystemPrompt string
	UserPrompt   string
	History      []openai.ChatCompletionMessage // 历史上下文
	Tools        []openai.Tool                  // MCP 工具定义
}

// ChatResponse 统一响应结果
type ChatResponse struct {
	Content    string
	ToolCalls  []openai.ToolCall
	TokenUsage openai.Usage
}

// ChatCompletion 调用大模型
func (c *Client) ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// 1. 构建 Messages
	messages := make([]openai.ChatCompletionMessage, 0)

	// 添加 System Prompt
	if req.SystemPrompt != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: req.SystemPrompt,
		})
	}

	// 添加历史记录 (User/Assistant/Tool 消息)
	if len(req.History) > 0 {
		messages = append(messages, req.History...)
	}

	// 添加当前用户输入 (如果 History 里没包含的话)
	if req.UserPrompt != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: req.UserPrompt,
		})
	}

	// 2. 发起请求
	resp, err := c.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:       c.config.ModelName,
			Messages:    messages,
			Tools:       req.Tools, // 这一步让模型知道有哪些工具可用
			Temperature: c.config.Temperature,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("llm api error: %v", err)
	}

	// 3. 解析结果
	choice := resp.Choices[0]

	return &ChatResponse{
		Content:    choice.Message.Content,
		ToolCalls:  choice.Message.ToolCalls, // 如果模型决定调工具，这里会有值
		TokenUsage: resp.Usage,
	}, nil
}
