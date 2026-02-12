package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"example.com/agent-server/internal/middleware"
	"example.com/agent-server/internal/service/runner"
	"example.com/agent-server/internal/store"
	"example.com/agent-server/pkg/response"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/google/uuid"
	"github.com/hertz-contrib/sse"
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
	Content    map[string]interface{} `json:"content"`          // JSON 结构
	RunID      string                 `json:"run_id,omitempty"` // 关联的 Run ID
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
		RunID:      m.RunID,
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

// SendChatMessageStream 流式发送消息 (SSE)
func (h *Handler) SendChatMessageStream(c context.Context, ctx *app.RequestContext) {
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		ctx.SetStatusCode(http.StatusUnauthorized)
		ctx.WriteString("Unauthorized")
		return
	}

	sessionID := ctx.Param("id")
	session := h.Store.GetChatSession(sessionID)
	if session == nil || session.UserID != userID {
		ctx.SetStatusCode(http.StatusNotFound)
		ctx.WriteString("Session not found")
		return
	}

	var req SendChatReq
	if err := ctx.BindAndValidate(&req); err != nil {
		ctx.SetStatusCode(http.StatusBadRequest)
		ctx.WriteString(err.Error())
		return
	}

	// 保存用户消息
	userMsg := &store.ChatMessage{
		SessionID: sessionID,
		Role:      "user",
		Content:   createTextContent(req.Content),
	}
	h.Store.CreateChatMessage(userMsg)

	// 创建 Run
	run := &store.Run{
		SessionID:    sessionID,
		UserID:       userID,
		AgentID:      session.AgentID,
		TraceID:      uuid.New().String(),
		Status:       "running",
		InputPayload: map[string]interface{}{"content": req.Content},
	}
	h.Store.CreateRun(run)

	// 设置 SSE 响应头
	ctx.SetStatusCode(http.StatusOK)
	ctx.Response.Header.Set("X-Accel-Buffering", "no") // 禁用 nginx 缓冲

	// 创建 SSE Stream
	stream := sse.NewStream(ctx)

	// 创建事件通道
	eventChan := make(chan runner.RunStreamEvent, 10)

	// 异步执行 Agent
	go h.Engine.ExecuteRunStream(run.ID, eventChan)

	// 流式推送给前端
	for event := range eventChan {
		data, _ := json.Marshal(event)
		err := stream.Publish(&sse.Event{
			Event: event.Type,
			Data:  data,
		})
		if err != nil {
			// 客户端断开连接
			return
		}
	}
}
