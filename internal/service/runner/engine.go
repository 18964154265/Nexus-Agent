package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"example.com/agent-server/internal/service/llm"
	"example.com/agent-server/internal/store"
)

// AgentEngine 负责编排一次 Run 的全过程
type AgentEngine struct {
	Store     *store.MemoryStore
	LLMClient *llm.Client
}

func NewEngine(s *store.MemoryStore) *AgentEngine {
	cfg := llm.LLMConfig{
		ApiKey:      "apikey", //后续填入真实
		BaseURL:     "https://api.siliconflow.cn/v1",
		ModelName:   "Qwen2.5-VL-32B-Instruct",
		Temperature: 0.5,
	}
	return &AgentEngine{Store: s, LLMClient: llm.NewClient(cfg)}

}

// ExecuteRun 核心方法：执行 Agent 的思考循环
func (e *AgentEngine) ExecuteRun(ctx context.Context, runID string) (string, error) {

	run := e.Store.GetRun(runID) // 根据 runID 获取 run
	if run == nil {
		return "", fmt.Errorf("run not found")
	}
	session := e.Store.GetChatSession(run.SessionID) // 获取 run 所在的 session
	if session == nil {
		return "", fmt.Errorf("session not found")
	}
	agent := e.Store.GetAgent(run.AgentID)
	if agent == nil {
		return "", fmt.Errorf("agent not found")
	}

	maxSteps := 5

	for i := 0; i < maxSteps; i++ {
		history := e.Store.ListChatMessagesBySession(session.ID)
		tools := e.Store.ListMCPToolsByAgent(agent.ID)

		req := llm.ChatRequest{
			SystemPrompt: agent.SystemPrompt,
			History:      history,
			Tools:        tools,
		}
		fmt.Printf("[Agent] Step %d: Thinking...\n", i+1)
		resp, err := e.LLMClient.ChatCompletion(ctx, &req)
		if err != nil {
			return "", fmt.Errorf("step %d: %v", i+1, err)
		}

		if len(resp.ToolCalls) > 0 {
			// 处理工具调用
			fmt.Printf("[Agent] Step %d: Tool Call(s) detected: %v\n", i+1, resp.ToolCalls)
			toolCallsJson, _ := json.Marshal(resp.ToolCalls)
			e.Store.CreateChatMessage(&store.ChatMessage{
				SessionID:  session.ID,
				RunID:      run.ID,
				Role:       "assistant",
				Content:    map[string]interface{}{"text": resp.Content, "tool_calls": string(toolCallsJson)},
				ToolCallID: resp.ToolCalls[0].ID, //这里只需要存一个ID吗？
				CreatedAt:  time.Now(),
			})

			for _, call := range resp.ToolCalls {
				fmt.Printf("[Agent]: Tool Call %s: %s\n", call.ID, call.Function.Name)
				e.saveStep(run, "tool_start", call.Function.Name, nil) //为什么是nil
				output, err := e.executeToolCall(ctx, call)
				if err != nil {
					output = fmt.Sprintf("Tool error %s: %v", call.ID, err)
				}
				e.saveStep(run, "tool_end", call.Function.Name, map[string]interface{}{
					"output": output,
				})
				e.saveToolOutput(run, call.ID, output)
			}
			continue

		} else {
			// 处理普通回复
			fmt.Printf("[Agent] Step %d: Received response: %s\n", i+1, resp.Content)
			e.saveMessage(run, "assistant", resp.Content, "")
			return resp.Content, nil
		}
	}
	return "", fmt.Errorf("max steps reached")
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
