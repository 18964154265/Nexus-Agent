package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"example.com/agent-server/internal/middleware"
	"example.com/agent-server/internal/store"
	"example.com/agent-server/pkg/response"
	"github.com/cloudwego/hertz/pkg/app"
)

// ==========================================
// DTOs
// ==========================================

type CreateSessionReq struct {
	AgentID string `json:"agent_id" vd:"required"` // 必填
	Title   string `json:"title"`                  // 选填
}

type ChatSessionResp struct {
	*store.ChatSession
	// 覆盖时间字段，格式化为 string
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`

	// 屏蔽字段 (可选)
	UserID string `json:"-"` // 前端知道是自己的，没必要返
}

func toSessionResp(s *store.ChatSession) *ChatSessionResp {
	return &ChatSessionResp{
		ChatSession: s,
		CreatedAt:   s.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   s.UpdatedAt.Format(time.RFC3339),
	}
}

// ==========================================
// Handlers
// ==========================================

// ListChatSessions 获取会话列表
func (h *Handler) ListChatSessions(c context.Context, ctx *app.RequestContext) {
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		response.Unauthorized(ctx, "Unauthorized")
		return
	}

	items := h.Store.ListChatSessionsByUser(userID)

	res := make([]*ChatSessionResp, 0, len(items))
	for _, s := range items {
		res = append(res, toSessionResp(s))
	}
	response.Success(ctx, res)
}

// CreateChatSession 创建会话
func (h *Handler) CreateChatSession(c context.Context, ctx *app.RequestContext) {
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		response.Unauthorized(ctx, "Unauthorized")
		return
	}

	var req CreateSessionReq
	if err := ctx.BindAndValidate(&req); err != nil {
		response.BadRequest(ctx, "参数错误: "+err.Error())
		return
	}

	// 添加了外键检查
	agent := h.Store.GetAgent(req.AgentID)
	if agent == nil {
		response.BadRequest(ctx, "Agent 不存在")
		return
	}

	// 【优化 2】: 处理默认标题
	if req.Title == "" {
		// 例如: "Chat with DevOps Manager"
		req.Title = fmt.Sprintf("Chat with %s", agent.Name)
	}

	s := &store.ChatSession{
		UserID:  userID,
		AgentID: req.AgentID,
		Title:   req.Title,
		// ID, CreatedAt 由 Store 处理
	}

	created := h.Store.CreateChatSession(s)
	response.Created(ctx, toSessionResp(created))
}

// GetChatSession 会话详情
func (h *Handler) GetChatSession(c context.Context, ctx *app.RequestContext) {
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		response.Unauthorized(ctx, "Unauthorized")
		return
	}

	id := ctx.Param("id")
	s := h.Store.GetChatSession(id)

	// 检查是否存在以及是否属于当前用户
	if s == nil || s.UserID != userID {
		response.Error(ctx, http.StatusNotFound, 40400, "Session not found")
		return
	}

	response.Success(ctx, toSessionResp(s))
}

// DeleteChatSession 删除会话
func (h *Handler) DeleteChatSession(c context.Context, ctx *app.RequestContext) {
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		response.Unauthorized(ctx, "Unauthorized")
		return
	}

	id := ctx.Param("id")
	s := h.Store.GetChatSession(id)

	// 权限检查
	if s == nil || s.UserID != userID {
		response.Error(ctx, http.StatusNotFound, 40400, "Session not found")
		return
	}

	// 执行删除
	if h.Store.DeleteChatSession(id) {
		response.Success(ctx, map[string]string{"message": "Deleted"})
	} else {
		response.ServerError(ctx, fmt.Errorf("failed to delete session"))
	}
}
