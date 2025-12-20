package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"example.com/agent-server/internal/service/llm"
	"example.com/agent-server/internal/store"
)

// AgentEngine 负责编排一次 Run 的全过程
type AgentEngine struct {
	Store       *store.MemoryStore
	LLMClient   *llm.Client
	runningRuns sync.Map // map[string]context.CancelFunc
}

func NewEngine(s *store.MemoryStore, c *llm.Client) *AgentEngine {
	if c != nil {
		return &AgentEngine{Store: s, LLMClient: c}
	}
	apiKey := os.Getenv("LLM_API_KEY")
	baseURL := os.Getenv("LLM_BASE_URL")
	model := os.Getenv("LLM_MODEL")
	tempStr := os.Getenv("LLM_TEMPERATURE")
	if apiKey == "" {
		apiKey = "apikey"
	}
	if baseURL == "" {
		baseURL = "https://api.siliconflow.com/v1"
	}
	if model == "" {
		model = "Qwen/QwQ-32B"
	}
	var temperature float32 = 0.5
	if tempStr != "" {
		if v, err := strconv.ParseFloat(tempStr, 32); err == nil {
			temperature = float32(v)
		}
	}
	cfg := llm.LLMConfig{ApiKey: apiKey, BaseURL: baseURL, ModelName: model, Temperature: temperature}
	return &AgentEngine{Store: s, LLMClient: llm.NewClient(cfg)}
}

// ExecuteRun 核心方法：执行 Agent 的思考循环
func (e *AgentEngine) ExecuteRun(parentCtx context.Context, runID string) (string, error) {
	// 1. 创建可取消的上下文
	ctx, cancel := context.WithCancel(parentCtx)
	e.runningRuns.Store(runID, cancel)
	defer func() {
		cancel()
		e.runningRuns.Delete(runID)
	}()

	run := e.Store.GetRun(runID)
	if run == nil {
		return "", fmt.Errorf("run not found")
	}
	session := e.Store.GetChatSession(run.SessionID)
	if session == nil {
		return "", fmt.Errorf("session not found")
	}
	agent := e.Store.GetAgent(run.AgentID)
	if agent == nil {
		return "", fmt.Errorf("agent not found")
	}

	// 限制最大思考步数，防止死循环烧钱
	maxSteps := 5

	for i := 0; i < maxSteps; i++ {
		// 1. 准备上下文
		history := e.Store.ListChatMessagesBySession(session.ID)
		tools := e.Store.ListMCPToolsByAgent(agent.ID)

		req := llm.ChatRequest{
			SystemPrompt: agent.SystemPrompt,
			History:      history,
			Tools:        tools,
		}

		fmt.Printf("======== DEBUG TOOLS ========\n")
		for _, t := range tools {
			fmt.Printf("- Tool: %s (ServerID: %s)\n", t.Name, t.ServerID)
		}
		fmt.Printf("=============================\n")

		fmt.Printf("[Agent] Step %d: Thinking...\n", i+1)

		// 2. LLM 推理
		resp, err := e.LLMClient.ChatCompletion(ctx, &req)
		if err != nil {
			return "", fmt.Errorf("step %d error: %v", i+1, err)
		}

		// 3. 判断是否调用工具
		if len(resp.ToolCalls) > 0 {
			fmt.Printf("[Agent] Step %d: Tool Call(s) detected: %d calls\n", i+1, len(resp.ToolCalls))

			// A. 先把 Assistant 的决定存入历史 (Question 1 修正)
			// 注意：这里 ToolCallID 留空，因为这是一条 Assistant 消息
			// 我们把 tool_calls 完整结构存入 Content，以便下次发给 LLM
			// (前提是你的 llm.Client.buildMessages 能正确处理 Content 里的 tool_calls 字段)

			// 为了简单，我们把 ToolCalls 转成 map 存进去
			toolCallsMap := make([]map[string]interface{}, 0)
			for _, tc := range resp.ToolCalls {
				toolCallsMap = append(toolCallsMap, map[string]interface{}{
					"id":   tc.ID,
					"type": "function",
					"function": map[string]interface{}{
						"name":      tc.Function.Name,
						"arguments": tc.Function.Arguments,
					},
				})
			}

			e.Store.CreateChatMessage(&store.ChatMessage{
				SessionID: session.ID,
				RunID:     run.ID,
				Role:      "assistant",
				Content:   map[string]interface{}{"text": resp.Content, "tool_calls": toolCallsMap}, //tool_calls[i].id为某个具体工具的id
				CreatedAt: time.Now(),
				// ToolCallID: "", // 这里不要填！
			})

			// B. 挨个执行工具
			for _, call := range resp.ToolCalls {
				fmt.Printf("[Agent] Executing Tool: %s (ID: %s)\n", call.Function.Name, call.ID)

				// 解析参数用于 Trace (Question 2 修正)
				var args map[string]interface{}
				_ = json.Unmarshal([]byte(call.Function.Arguments), &args) // 忽略错误，仅仅为了记录

				// 记录 Trace: Tool Start (不再是 nil)
				e.saveStep(run, "tool_start", call.Function.Name, args)

				// 执行工具逻辑
				output, err := e.executeToolCall(ctx, call)
				if err != nil {
					output = fmt.Sprintf("Tool Execution Error: %v", err)
				}

				// 记录 Trace: Tool End
				e.saveStep(run, "tool_end", call.Function.Name, map[string]interface{}{
					"output": output,
				})

				// 将工具结果存入对话历史 (User 不可见，LLM 可见)
				e.saveToolOutput(run, call.ID, output)
			}

			// 继续下一轮循环 (将工具结果发回给 LLM)
			continue

		} else {
			// 4. 没有工具调用，说明是最终回复
			fmt.Printf("[Agent] Step %d: Final Response: %s\n", i+1, resp.Content)
			e.saveMessage(run, "assistant", resp.Content, "")

			// 更新 Run 状态
			e.Store.FinishRun(run.ID, map[string]interface{}{"response": resp.Content}, "succeeded")

			return resp.Content, nil
		}
	}
	return "", fmt.Errorf("max steps reached")
}

// CancelRun 异步取消任务
func (e *AgentEngine) CancelRun(runID string) error {
	val, ok := e.runningRuns.Load(runID)
	if !ok {
		// 任务可能已经结束，或者根本不存在
		// 这种情况下也可以认为“取消成功”（幂等），或者返回特定错误
		return fmt.Errorf("run not running or not found")
	}
	cancel, ok := val.(context.CancelFunc)
	if !ok {
		return fmt.Errorf("invalid context type")
	}
	cancel() // 触发 Context Done 信号
	return nil
}

// 辅助函数：存消息
func (e *AgentEngine) saveMessage(run *store.Run, role, content, toolCallID string) {
	e.Store.CreateChatMessage(&store.ChatMessage{
		SessionID:  run.SessionID,
		RunID:      run.ID,
		Role:       role,
		Content:    map[string]interface{}{"type": "text", "text": content},
		ToolCallID: toolCallID,
		CreatedAt:  time.Now(),
	})
}

// 辅助函数：存步骤
func (e *AgentEngine) saveStep(run *store.Run, stepType, name string, payload map[string]interface{}) {
	e.Store.CreateRunStep(&store.RunStep{
		RunID:         run.ID,
		StepType:      stepType,
		Name:          name,
		OutputPayload: payload,
		StartedAt:     time.Now(),
	})
}

// 简单的 Mock 判断逻辑
func isFinalAnswer(s string) bool { return len(s) > 12 } // 简单模拟
func isToolCall(s string) bool    { return s == "TOOL_CALL: git_status" }
