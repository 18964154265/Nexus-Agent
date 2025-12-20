package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"example.com/agent-server/internal/middleware"
	"example.com/agent-server/internal/store"
	"example.com/agent-server/pkg/response"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/google/uuid"
)

// ==========================================
// DTOs
// ==========================================

type SendChatReq struct {
	Content string `json:"content" vd:"required"` // 用户发送的文本
}

// ChatMessageResp 返回给前端的消息格式
type ChatMessageResp struct {
	ID         string                 `json:"id"`
	Role       string                 `json:"role"`
	Content    map[string]interface{} `json:"content"` // JSON 结构
	ToolCallID string                 `json:"tool_call_id,omitempty"`
	CreatedAt  string                 `json:"created_at"`
}

// ==========================================
// Helpers
// ==========================================

func toMessageResp(m *store.ChatMessage) *ChatMessageResp {
	return &ChatMessageResp{
		ID:         m.ID,
		Role:       m.Role,
		Content:    m.Content,
		ToolCallID: m.ToolCallID,
		CreatedAt:  m.CreatedAt.Format(time.RFC3339),
	}
}

// createTextContent 构造标准的 {"type": "text", "text": "..."} 结构
func createTextContent(text string) map[string]interface{} {
	return map[string]interface{}{
		"type": "text",
		"text": text,
	}
}

// ==========================================
// Handlers
// ==========================================

// ListChatMessages 获取会话历史
func (h *Handler) ListChatMessages(c context.Context, ctx *app.RequestContext) {
	// 1. 鉴权
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		response.Unauthorized(ctx, "Unauthorized")
		return
	}

	// 2. 校验 Session 归属
	sessionID := ctx.Param("id")
	session := h.Store.GetChatSession(sessionID)
	if session == nil || session.UserID != userID {
		response.Error(ctx, http.StatusNotFound, 40400, "Session not found")
		return
	}

	// 3. 获取消息
	msgs := h.Store.ListChatMessagesBySession(sessionID)

	// 4. 转换格式
	res := make([]*ChatMessageResp, 0, len(msgs))
	for _, m := range msgs {
		res = append(res, toMessageResp(m))
	}

	response.Success(ctx, res)
}

// SendChatMessage 发送消息 (核心!)
func (h *Handler) SendChatMessage(c context.Context, ctx *app.RequestContext) {
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		response.Unauthorized(ctx, "Unauthorized")
		return
	}

	sessionID := ctx.Param("id")
	session := h.Store.GetChatSession(sessionID)
	if session == nil || session.UserID != userID {
		response.Error(ctx, http.StatusNotFound, 40400, "Session not found")
		return
	}

	var req SendChatReq
	if err := ctx.BindAndValidate(&req); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}

	// =============================================================
	// Step 1: 保存用户消息 (User Message)
	// =============================================================
	userMsg := &store.ChatMessage{
		SessionID: sessionID,
		Role:      "user",
		Content:   createTextContent(req.Content),
	}
	h.Store.CreateChatMessage(userMsg)

	// =============================================================
	// Step 2: 创建运行任务 (Run) - 状态为 Running
	// =============================================================
	run := &store.Run{
		SessionID:    sessionID,
		UserID:       userID,
		AgentID:      session.AgentID,
		TraceID:      uuid.New().String(), // 生成新的 Trace
		Status:       "running",
		InputPayload: map[string]interface{}{"content": req.Content},
	}
	// 关联一下 UserMsg (如果表里有 RunID 字段的话)
	// userMsg.RunID = run.ID
	h.Store.CreateRun(run)

	// =============================================================
	// Step 3: 执行 Agent 逻辑 (Mock Engine)
	// =============================================================
	// 注意：真实场景中，这里应该是异步的，或者 SSE 流式返回
	// 这里我们模拟一个同步阻塞的过程，让前端一次性拿到结果

	finalResponse, err := h.Engine.ExecuteRun(run.ID)
	if err != nil {
		h.Store.FinishRun(run.ID, nil, "failed")
		response.ServerError(ctx, err)
		return
	}

	// =============================================================
	// Step 4: 结束 Run
	// =============================================================
	h.Store.FinishRun(run.ID, map[string]interface{}{"response": finalResponse}, "succeeded")

	// 返回最终结果 (通常前端发一条消息，期望立刻看到 Agent 的回复)
	// 如果是流式，这里就不返回 JSON，而是 Hijack 连接推流
	// 这里我们返回【最新的一条】Assistant 消息

	// 重新查一下最新的消息
	allMsgs := h.Store.ListChatMessagesBySession(sessionID)
	lastMsg := allMsgs[len(allMsgs)-1]

	response.Success(ctx, toMessageResp(lastMsg))
}

// =================================================================
// 核心：仿真执行引擎 (Mock LLM & Tool Execution)
// =================================================================
// 这个函数模拟了 "LLM 思考 -> 调工具 -> 工具返回 -> LLM 回复" 的全过程
func (h *Handler) simulateAgentExecution(run *store.Run, userPrompt string) string {

	// 模拟思考耗时
	time.Sleep(500 * time.Millisecond)

	// 场景 A: 用户提到了 "git" -> 触发工具调用模拟
	if strings.Contains(strings.ToLower(userPrompt), "git") {
		return h.mockToolExecutionFlow(run)
	}

	// 场景 B: 普通对话 -> 直接回复
	responseText := fmt.Sprintf("收到，我是 Nexus Agent。你刚才说的是: \"%s\"。这是一个模拟回复，并未接入真实 LLM。", userPrompt)

	// 记录思考步骤 (Trace)
	h.Store.CreateRunStep(&store.RunStep{
		RunID:        run.ID,
		StepType:     "thought",
		Name:         "SimpleReply",
		Status:       "completed",
		StartedAt:    time.Now(),
		FinishedAt:   time.Now(),
		InputPayload: map[string]interface{}{"prompt": userPrompt},
	})

	// 记录 Assistant 回复
	h.Store.CreateChatMessage(&store.ChatMessage{
		SessionID: run.SessionID,
		RunID:     run.ID,
		Role:      "assistant",
		Content:   createTextContent(responseText),
	})

	return responseText
}

// mockToolExecutionFlow 模拟复杂的工具调用链路
func (h *Handler) mockToolExecutionFlow(run *store.Run) string {
	// 1. Assistant 决定调用工具 (Thinking)
	toolCallID := uuid.New().String()
	toolName := "git_status"

	// 记录 Message: "我打算调 git_status"
	h.Store.CreateChatMessage(&store.ChatMessage{
		SessionID: run.SessionID,
		RunID:     run.ID,
		Role:      "assistant",
		Content: map[string]interface{}{
			"type": "text",
			"text": "检测到 Git 相关请求，正在查看仓库状态...",
		},
		// 关键：模拟 Tool Calls 结构 (OpenAI 格式)
		// 这里简化存入 content，实际应该有专门字段或结构
	})

	// 记录 Run Step: Tool Start
	h.Store.CreateRunStep(&store.RunStep{
		RunID:        run.ID,
		StepType:     "tool_start",
		Name:         toolName,
		Status:       "running",
		InputPayload: map[string]interface{}{"repo": "."},
		StartedAt:    time.Now(),
	})

	// 模拟工具执行耗时
	time.Sleep(800 * time.Millisecond)

	// 2. 工具执行结果 (Tool Output)
	toolOutput := "On branch main\nYour branch is up to date with 'origin/main'.\nNothing to commit, working tree clean."

	// 记录 Message: Tool 的返回值
	h.Store.CreateChatMessage(&store.ChatMessage{
		SessionID:  run.SessionID,
		RunID:      run.ID,
		Role:       "tool",
		ToolCallID: toolCallID, // 对应之前的 ID
		Content:    createTextContent(toolOutput),
		IsHidden:   true, // 前端通常折叠这个
	})

	// 记录 Run Step: Tool End
	h.Store.CreateRunStep(&store.RunStep{
		RunID:         run.ID,
		StepType:      "tool_end",
		Name:          toolName,
		Status:        "completed",
		OutputPayload: map[string]interface{}{"stdout": toolOutput},
		LatencyMS:     800,
		FinishedAt:    time.Now(),
	})

	// 3. Assistant 根据工具结果生成最终回复
	finalText := "已查看当前 Git 仓库状态：\n当前位于 **main** 分支，工作区是干净的，没有未提交的更改。"

	h.Store.CreateChatMessage(&store.ChatMessage{
		SessionID: run.SessionID,
		RunID:     run.ID,
		Role:      "assistant",
		Content:   createTextContent(finalText),
	})

	return finalText
}
