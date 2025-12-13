package handler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"example.com/agent-server/internal/middleware"
	"example.com/agent-server/internal/store"
	"example.com/agent-server/pkg/response"
	"github.com/cloudwego/hertz/pkg/app"
)

// ==========================================
// DTOs
// ==========================================

// RegisterMCPServerReq 注册请求
type RegisterMCPServerReq struct {
	Name          string                 `json:"name" vd:"required"`
	TransportType string                 `json:"transport_type" vd:"in(stdio,sse)"` // 只允许 stdio 或 sse
	AgentID       string                 `json:"agent_id"`                          // 可选，如果绑定特定 Agent
	Config        map[string]interface{} `json:"connection_config" vd:"required"`   // {"command": "...", "args": [...]}
}

// MCPServerResp 响应
type MCPServerResp struct {
	*store.MCPServer
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`

	// 统计信息：该 Server 下有多少个工具
	ToolCount int `json:"tool_count"`
}

// MCPToolResp 工具响应
type MCPToolResp struct {
	*store.MCPTool
	CreatedAt string `json:"created_at"`
	// Schema 已经是 map，直接透传即可
}

// ==========================================
// Handlers
// ==========================================

// ListMCPServers 列出 Server
// 支持 Query Param: ?agent_id=xxx (列出全局 + 该 Agent 私有的)
func (h *Handler) ListMCPServers(c context.Context, ctx *app.RequestContext) {
	agentID := ctx.Query("agent_id")

	// 1. 从数据库获取
	// 这里简化逻辑：如果传了 agentID，就查那个 Agent 的；没传就查所有的(仅用于演示)
	// 实际生产中应该是：Global Servers + Private Servers (where owner=me)
	var servers []*store.MCPServer
	if agentID != "" {
		servers = h.Store.ListMCPServersByAgent(agentID)
	} else {
		// 如果没传 ID，暂时列出所有（或者你可以定义一个 ListGlobalMCPServers）
		// 这里为了方便调试，我们简单遍历一下 map (性能较差，仅限 Demo)
		// 实际请在 store 增加 ListAllMCPServers 方法
		// 这里暂且返回空，或者你需要去 store 实现一个 ListAll
		servers = []*store.MCPServer{}
	}

	// 2. 转换 Resp
	res := make([]*MCPServerResp, 0, len(servers))
	for _, s := range servers {
		// 查一下工具有多少个
		tools := h.Store.ListMCPToolsByServer(s.ID)

		res = append(res, &MCPServerResp{
			MCPServer: s,
			CreatedAt: s.CreatedAt.Format(time.RFC3339),
			UpdatedAt: s.UpdatedAt.Format(time.RFC3339),
			ToolCount: len(tools),
		})
	}

	response.Success(ctx, res)
}

// RegisterMCPServer 注册新的 Server
func (h *Handler) RegisterMCPServer(c context.Context, ctx *app.RequestContext) {
	// 权限检查...
	_, ok := middleware.GetUserID(ctx)
	if !ok {
		response.Unauthorized(ctx, "Unauthorized")
		return
	}

	var req RegisterMCPServerReq
	if err := ctx.BindAndValidate(&req); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}
	//用户不得创建本地的mcpserver,只能采用远程连接的方式
	if req.TransportType == "stdio" {
		response.Error(ctx, http.StatusForbidden, 40300, "Security Alert: Users can only register SSE (Remote) servers, not local stdio processes.")
		return
	}

	// 创建 Server 对象
	server := &store.MCPServer{
		Name:             req.Name,
		AgentID:          req.AgentID, // 如果为空，表示全局或者未绑定
		TransportType:    req.TransportType,
		ConnectionConfig: req.Config,
		Status:           "active",
		IsGlobal:         req.AgentID == "", // 没绑 Agent 就是全局的
	}

	createdServer := h.Store.CreateMCPServer(server)

	response.Created(ctx, map[string]interface{}{
		"id": createdServer.ID,
		"data": &MCPServerResp{
			MCPServer: createdServer,
			CreatedAt: createdServer.CreatedAt.Format(time.RFC3339),
			UpdatedAt: createdServer.UpdatedAt.Format(time.RFC3339),
			ToolCount: 0,
		},
	})
}

// ListMCPTools 查看某个 Server 下的工具
func (h *Handler) ListMCPTools(c context.Context, ctx *app.RequestContext) {
	serverID := ctx.Param("id")

	// 校验 Server 是否存在
	// s := h.Store.GetMCPServer(serverID) ...

	tools := h.Store.ListMCPToolsByServer(serverID)

	res := make([]*MCPToolResp, 0, len(tools))
	for _, t := range tools {
		res = append(res, &MCPToolResp{
			MCPTool:   t,
			CreatedAt: t.CreatedAt.Format(time.RFC3339),
		})
	}

	response.Success(ctx, res)
}

// SyncMCPTools [核心] 同步工具
// 真实场景：连接 MCP 进程 -> 发送 list_tools -> 更新 DB
// 当前场景：模拟数据，方便前端开发
// ... 前面的 List 和 Register 保持不变 ...

// SyncMCPTools [核心] 同步工具
func (h *Handler) SyncMCPTools(c context.Context, ctx *app.RequestContext) {
	serverID := ctx.Param("id")

	// 调用 Service 层逻辑
	count, err := h.Svc.MCP.SyncTools(c, serverID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			response.Error(ctx, http.StatusNotFound, 40400, err.Error())
		} else {
			response.Error(ctx, http.StatusInternalServerError, 50000, err.Error())
		}
		return
	}

	response.Success(ctx, map[string]interface{}{
		"message":    "Sync successful",
		"sync_count": count,
	})
}
