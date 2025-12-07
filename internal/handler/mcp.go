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
	"github.com/google/uuid"
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
func (h *Handler) SyncMCPTools(c context.Context, ctx *app.RequestContext) {
	serverID := ctx.Param("id")
	server := h.Store.GetMCPServer(serverID)
	if server == nil {
		response.Error(ctx, http.StatusNotFound, 40400, "MCPServer not found")
		return
	}

	// 2. 【模拟】根据 Server 名字自动生成工具
	// 真实场景：调用mcp_client.Connect(server.Config).ListTools()
	newTools := mockToolsForServer(server.ID, server.Name) // 这里的 name 实际应该取 server.Name

	// 3. 存入数据库
	count := 0
	for _, t := range newTools {
		t.ID = uuid.New().String()
		t.CreatedAt = time.Now()
		t.UpdatedAt = time.Now()
		h.Store.CreateMCPTool(t)
		count++
	}

	response.Success(ctx, map[string]interface{}{
		"message":     "Sync successful",
		"server_name": server.Name,
		"sync_count":  count,
	})
}

// ==========================================
// Mock Helper (模拟 MCP 协议返回)
// ==========================================

func mockToolsForServer(serverID, serverName string) []*store.MCPTool {
	tools := []*store.MCPTool{}
	name := strings.ToLower(serverName)
	//1.模拟Git Server
	if strings.Contains(name, "git") {
		tools = append(tools, &store.MCPTool{
			ServerID:    serverID,
			Name:        "git_status",
			Description: "显示工作目录状态。",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"repo_path": map[string]string{"type": "string", "description": "仓库路径,默认当前目录"},
			},
			},
		})
		tools = append(tools, &store.MCPTool{
			ServerID:    serverID,
			Name:        "git_diff",
			Description: "显示commit,commit和工作树之间的修改。",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"target": map[string]string{"type": "string", "description": "目标commit,默认HEAD"},
				},
			},
		})
		tools = append(tools, &store.MCPTool{
			ServerID:    serverID,
			Name:        "git_commit",
			Description: "提交代码变更。",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"message"},
				"properties": map[string]interface{}{
					"message": map[string]string{"type": "string", "description": "提交信息"},
					"add_all": map[string]string{"type": "boolean", "description": "是否添加所有变更文件"},
				},
			},
		})
		tools = append(tools, &store.MCPTool{
			ServerID:    serverID,
			Name:        "git_log",
			Description: "显示提交日志。",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"max_count": map[string]string{"type": "integer", "description": "最大提交数,默认10"},
				},
			},
		})
	}
	//2.模拟Filesystem Server
	if strings.Contains(name, "filesystem") || strings.Contains(name, "fs") {
		tools = append(tools, &store.MCPTool{
			ServerID:    serverID,
			Name:        "list_directory",
			Description: "列出目录下的文件。",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"path"},
				"properties": map[string]interface{}{
					"path": map[string]string{"type": "string", "description": "目录路径"},
				},
			},
		})
		tools = append(tools, &store.MCPTool{
			ServerID:    serverID,
			Name:        "read_file",
			Description: "读取文件内容。",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"path"},
				"properties": map[string]interface{}{
					"path": map[string]string{"type": "string", "description": "文件路径"},
				},
			},
		})
		tools = append(tools, &store.MCPTool{
			ServerID:    serverID,
			Name:        "write_file",
			Description: "写入文件内容。",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"path", "content"},
				"properties": map[string]interface{}{
					"path":    map[string]string{"type": "string", "description": "文件路径"},
					"content": map[string]string{"type": "string", "description": "文件内容"},
				},
			},
		})
		tools = append(tools, &store.MCPTool{
			ServerID:    serverID,
			Name:        "search_files",
			Description: "搜索文件。",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"path", "pattern"},
				"properties": map[string]interface{}{
					"path":    map[string]string{"type": "string", "description": "文件路径"},
					"pattern": map[string]string{"type": "string", "description": "正则表达式"},
				},
			},
		})
	}
	return tools
}
