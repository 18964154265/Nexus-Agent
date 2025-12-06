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
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Type        string   `json:"type"` // user vs system
	Tags        []string `json:"tags"`
	ModelName   string   `json:"model_name"`
	CreatedAt   string   `json:"created_at"` // 格式化后的时间字符串

	// 注意：这里我们故意【不返回】 SystemPrompt，
	// 只有在 GetAgent 详情页才返回，List 列表页不返回以节省流量
}

// 辅助函数：把数据库模型转换为响应模型
func toAgentResp(a *store.Agent) AgentResp {
	return AgentResp{
		ID:          a.ID,
		Name:        a.Name,
		Description: a.Description,
		Type:        a.Type,
		Tags:        a.Tags,
		ModelName:   a.ModelName,
		// 把 time.Time 格式化为前端喜欢的字符串
		CreatedAt: a.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

// ListAgents 获取列表
func (h *Handler) ListAgents(c context.Context, ctx *app.RequestContext) {
	// 1. 从数据库拿原始数据
	agents := h.Store.ListAgents()

	// 2. 【核心修改】转换为 Resp 对象
	respList := make([]AgentResp, 0, len(agents))
	for _, a := range agents {
		respList = append(respList, toAgentResp(a))
	}

	// 3. 发送清洗过的数据给前端
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

	// 调用 Store 创建
	// 注意：需确保 Store 接口已更新支持返回 error
	createdAgent := h.Store.CreateAgent(agent)

	// 返回创建成功的精简信息
	ctx.JSON(http.StatusCreated, map[string]interface{}{
		"id":   createdAgent.ID,
		"data": toAgentResp(createdAgent), // 使用 Resp
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
