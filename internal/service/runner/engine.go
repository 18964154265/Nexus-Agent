package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"example.com/agent-server/internal/service/llm"
	"example.com/agent-server/internal/store"
)

// AgentEngine 负责编排一次 Run 的全过程
type AgentEngine struct {
	Store       store.Store
	LLMClient   *llm.Client
	runningRuns sync.Map        // map[string]context.CancelFunc
	rootCtx     context.Context // 全局根上下文
}

func NewEngine(s store.Store, c *llm.Client) *AgentEngine {
	e := &AgentEngine{Store: s, runningRuns: sync.Map{}, rootCtx: context.Background()}
	if c != nil {
		e.LLMClient = c
		return e
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
// 注意：runID 对应的任务将在 Engine 的 rootCtx 下运行，而非依赖调用者的 ctx
func (e *AgentEngine) ExecuteRun(runID string) (string, error) {
	// 1. 创建可取消的上下文，挂载在 rootCtx 下
	ctx, cancel := context.WithCancel(e.rootCtx)
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

		// 组装 handoff 目标（当前实现：列出除自身外的所有 Agent）
		handoffCandidates := e.buildHandoffCandidates(agent.ID)

		req := llm.ChatRequest{
			SystemPrompt:      agent.SystemPrompt + "\n\n" + buildToolInstruction(tools),
			History:           history,
			Tools:             tools,
			HandoffCandidates: handoffCandidates,
			ForceHandoff:      true,
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

		// 3.1 处理 handoff（target_agent_id 为空表示不切换）
		if resp.Handoff != nil && resp.Handoff.TargetAgentID != "" {
			fmt.Printf("[Agent] Step %d: Handoff -> Agent %s\n", i+1, resp.Handoff.TargetAgentID)

			// 记录 Assistant 消息，包含 handoff 信息
			e.Store.CreateChatMessage(&store.ChatMessage{
				SessionID: session.ID,
				RunID:     run.ID,
				Role:      "assistant",
				Content: map[string]interface{}{
					"text":    resp.Content,
					"handoff": resp.Handoff,
				},
				CreatedAt: time.Now(),
			})

			// 创建 handoff Step
			step := e.createStep(run, "handoff", "agent_handoff", map[string]interface{}{
				"target_agent_id":   resp.Handoff.TargetAgentID,
				"reason":            resp.Handoff.Reason,
				"preferred_server":  resp.Handoff.PreferredServer,
				"parent_agent_id":   agent.ID,
				"parent_agent_name": agent.Name,
			})

			// 触发子 Agent Run
			childResp, childRun, err := e.executeHandoff(ctx, run, resp.Handoff)
			status := "completed"
			errMsg := ""
			if err != nil {
				status = "failed"
				errMsg = err.Error()
			}

			e.finishStep(step.ID, map[string]interface{}{
				"child_run_id":   childRunIDOrEmpty(childRun),
				"child_agent_id": resp.Handoff.TargetAgentID,
				"response":       childResp,
			}, status, errMsg)

			if err != nil {
				e.Store.FinishRun(run.ID, map[string]interface{}{
					"error": errMsg,
				}, "failed")
				return "", err
			}

			// 父 Run 成功，输出子 Agent 结果
			e.Store.FinishRun(run.ID, map[string]interface{}{
				"child_run_id": childRun.ID,
				"response":     childResp,
			}, "succeeded")
			return childResp, nil
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

				// 记录 Trace: Tool Start
				step := e.createStep(run, "tool_call", call.Function.Name, args)

				// 执行工具逻辑
				output, err := e.executeToolCall(ctx, call)
				status := "completed"
				errMsg := ""
				if err != nil {
					status = "failed"
					errMsg = err.Error()
					output = fmt.Sprintf("Tool Execution Error: %v", err)
				}

				// 记录 Trace: Tool End
				e.finishStep(step.ID, map[string]interface{}{"output": output}, status, errMsg)

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

// 辅助函数：创建步骤 (开始)
func (e *AgentEngine) createStep(run *store.Run, stepType, name string, input map[string]interface{}) *store.RunStep {
	step := &store.RunStep{
		RunID:        run.ID,
		StepType:     stepType,
		Name:         name,
		InputPayload: input,
		Status:       "running",
		StartedAt:    time.Now(),
	}
	return e.Store.CreateRunStep(step)
}

// 辅助函数：结束步骤 (完成)
func (e *AgentEngine) finishStep(stepID string, output map[string]interface{}, status, errMsg string) {
	start := time.Now() // 兜底，实际应该从 Store 取出来算
	if step := e.Store.GetRunStep(stepID); step != nil {
		start = step.StartedAt
	}
	latency := int(time.Since(start).Milliseconds())

	e.Store.FinishRunStep(stepID, output, status, latency, errMsg)
}

// 简单的 Mock 判断逻辑
func isFinalAnswer(s string) bool { return len(s) > 12 } // 简单模拟
func isToolCall(s string) bool    { return s == "TOOL_CALL: git_status" }

// buildHandoffCandidates 构造可切换的 Agent 列表（排除当前）
func (e *AgentEngine) buildHandoffCandidates(currentAgentID string) []llm.HandoffCandidate {
	agents := e.Store.ListAgents()
	res := make([]llm.HandoffCandidate, 0, len(agents))
	for _, a := range agents {
		if a == nil || a.ID == currentAgentID {
			continue
		}
		res = append(res, llm.HandoffCandidate{
			AgentID:     a.ID,
			Name:        a.Name,
			Description: a.Description,
		})
	}
	return res
}

// executeHandoff 创建子 Run 并递归执行
func (e *AgentEngine) executeHandoff(ctx context.Context, parentRun *store.Run, decision *llm.HandoffDecision) (string, *store.Run, error) {
	if decision == nil || decision.TargetAgentID == "" {
		return "", nil, fmt.Errorf("invalid handoff decision")
	}

	targetAgent := e.Store.GetAgent(decision.TargetAgentID)
	if targetAgent == nil {
		return "", nil, fmt.Errorf("target agent not found: %s", decision.TargetAgentID)
	}

	childRun := &store.Run{
		SessionID:     parentRun.SessionID,
		UserID:        parentRun.UserID,
		AgentID:       decision.TargetAgentID,
		ParentRunID:   parentRun.ID,
		TraceID:       parentRun.TraceID,
		Status:        "running",
		InputPayload:  map[string]interface{}{"from_handoff": true, "reason": decision.Reason},
		OutputPayload: map[string]interface{}{},
	}

	created, err := e.Store.CreateRun(childRun)
	if err != nil {
		return "", nil, fmt.Errorf("create child run failed: %w", err)
	}

	// 递归执行子 Agent
	resp, err := e.ExecuteRun(created.ID)
	if err != nil {
		e.Store.FinishRun(created.ID, map[string]interface{}{"error": err.Error()}, "failed")
		return "", created, err
	}
	return resp, created, nil
}

func childRunIDOrEmpty(run *store.Run) string {
	if run == nil {
		return ""
	}
	return run.ID
}

// buildToolInstruction 生成系统提示，引导 LLM 优先使用可用工具
func buildToolInstruction(tools []*store.MCPTool) string {
	if len(tools) == 0 {
		return "You have no available tools. Answer directly."
	}
	builder := strings.Builder{}
	builder.WriteString("You have the ability to call tools to complete tasks. Prefer using tools when they can provide accurate or fresh information.\nAvailable tools:\n")
	for i, t := range tools {
		builder.WriteString(fmt.Sprintf("%d. %s - %s\n", i+1, t.Name, t.Description))
	}
	builder.WriteString("When a tool is relevant, call it with correct arguments; otherwise answer directly.")
	return builder.String()
}
