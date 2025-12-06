package handler

import (
	"context"
	"net/http"

	"example.com/agent-server/internal/store"
	"github.com/cloudwego/hertz/pkg/app"
)

// ==========================================
// DTOs (请求参数定义)
// ==========================================

type AgentReq struct {
	Name             string                 `json:"name" vd:"required"` // 必填
	Description      string                 `json:"description"`
	ModelName        string                 `json:"model_name"` // 默认 gpt-4o
	SystemPrompt     string                 `json:"system_prompt" vd:"required"`
	Temperature      float64                `json:"temperature" vd:">=0,<=2"`
	KnowledgeBaseIDs []string               `json:"knowledge_base_ids"`
	Tags             []string               `json:"tags"`
	ExtraConfig      map[string]interface{} `json:"extra_config"`
	// Capabilities 定义该 Agent 能干什么，比如 ["code", "review"]
	Capabilities []string `json:"capabilities"`
}

type AgentResp struct {
	*store.Agent // 嵌入原始结构

	// 1. 覆盖字段 (Shadowing)
	// 这个 string 类型的 CreatedAt 会覆盖 store.Agent 里的 CreatedAt
	// 前端收到的 json key 也是 "created_at"
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`

	// 2. 屏蔽敏感字段
	// 显式忽略 SystemPrompt 等
	SystemPrompt string `json:"-"`
	ExtraConfig  any    `json:"-"`
	OwnerUserID  string `json:"-"`
}

func toAgentResp(a *store.Agent) *AgentResp {
	return &AgentResp{
		Agent:     a,                                         // 直接把 store 对象塞进去
		CreatedAt: a.CreatedAt.Format("2006-01-02 15:04:05"), // 格式化时间
		UpdatedAt: a.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

// ListAgents 获取列表
func (h *Handler) ListAgents(c context.Context, ctx *app.RequestContext) {
	agents := h.Store.ListAgents()

	respList := make([]*AgentResp, 0, len(agents))
	for _, a := range agents {
		respList = append(respList, toAgentResp(a))
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"data": respList,
	})
}

// ==========================================
// Handlers
// ==========================================

// CreateAgent 创建一个新的自定义 Agent
func (h *Handler) CreateAgent(c context.Context, ctx *app.RequestContext) {
	userID, ok := GetUserIDFromCtx(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		return
	}

	var req AgentReq
	if err := ctx.BindAndValidate(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// 设置默认值
	if req.ModelName == "" {
		req.ModelName = "gpt-4o"
	}

	agent := &store.Agent{
		OwnerUserID:      userID,
		Name:             req.Name,
		Description:      req.Description,
		ModelName:        req.ModelName,
		SystemPrompt:     req.SystemPrompt,
		Temperature:      req.Temperature,
		KnowledgeBaseIDs: req.KnowledgeBaseIDs,
		Tags:             req.Tags,
		ExtraConfig:      req.ExtraConfig,
		Capabilities:     req.Capabilities,
		Status:           "active",
		Type:             "user", // 用户创建的标记为 user
	}

	createdAgent := h.Store.CreateAgent(agent)

	ctx.JSON(http.StatusCreated, map[string]interface{}{
		"id":   createdAgent.ID,
		"data": toAgentResp(createdAgent),
	})
}

// GetAgent 获取详情 (详情页可能需要 SystemPrompt，可以定义另一个 DetailResp)
func (h *Handler) GetAgent(c context.Context, ctx *app.RequestContext) {
	id := ctx.Param("id")
	agent := h.Store.GetAgent(id)
	if agent == nil {
		ctx.JSON(http.StatusNotFound, map[string]string{"error": "Agent not found"})
		return
	}

	// 这里为了简单，我们还是直接返回 agent 或者构造一个包含 prompt 的 Resp
	// 假设详情页允许看 prompt
	ctx.JSON(http.StatusOK, map[string]interface{}{
		"data": agent, // 详情页直接透传数据库模型通常是可以接受的
	})
}

// UpdateAgent 更新 Agent
func (h *Handler) UpdateAgent(c context.Context, ctx *app.RequestContext) {
	id := ctx.Param("id")
	userID, ok := GetUserIDFromCtx(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, nil)
		return
	}

	// 1. 检查权限
	existing := h.Store.GetAgent(id)
	if existing == nil {
		ctx.JSON(http.StatusNotFound, nil)
		return
	}
	// 只有拥有者或系统管理员(这里简化)能修改
	if existing.Type != "system" && existing.OwnerUserID != userID {
		ctx.JSON(http.StatusForbidden, map[string]string{"error": "No permission"})
		return
	}

	// 2. 绑定参数
	var req AgentReq
	if err := ctx.BindAndValidate(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// 3. 执行更新 (使用闭包回调)
	updated := h.Store.UpdateAgent(id, func(a *store.Agent) {
		a.Name = req.Name
		a.Description = req.Description
		a.SystemPrompt = req.SystemPrompt
		a.Temperature = req.Temperature
		a.Tags = req.Tags
		a.ModelName = req.ModelName
		// ... 其他字段
	})

	if !updated {
		ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update"})
		return
	}

	ctx.JSON(http.StatusOK, map[string]string{"message": "Agent updated"})
}

// DeleteAgent 删除 Agent
func (h *Handler) DeleteAgent(c context.Context, ctx *app.RequestContext) {
	id := ctx.Param("id")
	userID, ok := GetUserIDFromCtx(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, nil)
		return
	}

	existing := h.Store.GetAgent(id)
	if existing == nil {
		ctx.JSON(http.StatusNotFound, nil)
		return
	}
	if existing.Type == "system" {
		ctx.JSON(http.StatusForbidden, map[string]string{"error": "Cannot delete system agent"})
		return
	}
	if existing.OwnerUserID != userID {
		ctx.JSON(http.StatusForbidden, nil)
		return
	}

	if h.Store.DeleteAgent(id) {
		ctx.JSON(http.StatusOK, map[string]string{"message": "Deleted"})
	} else {
		ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete"})
	}
}
