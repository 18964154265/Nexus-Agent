package runner

import (
	"context"
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
// 这里替换掉之前 handler 里的 simulateAgentExecution
func (e *AgentEngine) ExecuteRun(ctx context.Context, runID string) (string, error) {

	run := e.Store.GetRun(runID) // 根据 runID 获取 run
	if run == nil {
		return "", fmt.Errorf("run not found")
	}
	session := e.Store.GetChatSession(run.SessionID) // 获取 run 所在的 session
	if session == nil {
		return "", fmt.Errorf("session not found")
	}
	dbMsgs := e.Store.ListChatMessagesBySession(run.SessionID)
	history := DBMessageToOpenAI(dbMsgs)

	resp, err := e.LLMClient.ChatCompletion(ctx, llm.ChatRequest{
		SystemPrompt: "you are a devops assistant",
		History:      history,
		Tools:        nil, //先置空
	})
	if err != nil {
		return "", err
	}

	if len(resp.ToolCalls) > 0 {
		// =======================================================
		// 🚀 The ReAct Loop (核心循环)
		// =======================================================
		// 为了防止死循环，设置最大步数，比如 10 步
		maxSteps := 10

		for i := 0; i < maxSteps; i++ {
			// Step A: 思考 (Call LLM)
			// llmResp, err := e.LLMClient.ChatCompletion(prompt)
			// ---------------------------------------------------
			// 【模拟 LLM 返回】: 假设第一次返回 ToolCall，第二次返回文本
			var llmDecision string
			if i == 0 {
				llmDecision = "TOOL_CALL: git_status" // 模拟想调工具
			} else {
				llmDecision = "FINAL_ANSWER: 仓库很干净" // 模拟最终回复
			}
			// ---------------------------------------------------

			// Step B: 处理决策
			if isFinalAnswer(llmDecision) {
				// 1. 记录 Assistant 消息
				e.saveMessage(run, "assistant", "仓库很干净", "")
				return "仓库很干净", nil
			}

			if isToolCall(llmDecision) {
				// 1. 记录 "我要调工具" 的想法
				e.saveMessage(run, "assistant", "正在检查状态...", "call_id_123")

				// 2. 记录 RunStep (Tool Start)
				e.saveStep(run, "tool_start", "git_status", nil)

				// Step C: 行动 (Execute Tool)
				// toolResult := e.executeTool("git_status", args)
				toolResult := "On branch main, nothing to commit" // 模拟结果

				// 3. 记录 RunStep (Tool End)
				e.saveStep(run, "tool_end", "git_status", map[string]interface{}{"output": toolResult})

				// 4. 记录 Tool Message (观察)
				// 这一步非常重要！把结果喂回给 LLM
				e.saveMessage(run, "tool", toolResult, "call_id_123")

				// Continue Loop -> LLM 看到结果后，进入下一次迭代
			}
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
