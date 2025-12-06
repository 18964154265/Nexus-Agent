package http

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"

	"example.com/agent-server/internal/handler" // 引入你的 handler 包
	"example.com/agent-server/internal/middleware"
)

// RegisterRoutes 注册路由
// h: Hertz 实例
// hdl: 我们初始化好的业务逻辑处理器 (包含数据库和密钥)
// secret: 用于中间件校验的密钥
func RegisterRoutes(h *server.Hertz, hdl *handler.Handler, secret string) {

	// ===========================
	// 1. 公开接口 (Public)
	// ===========================
	h.GET("/health", func(c context.Context, ctx *app.RequestContext) {
		ctx.String(200, "OK")
	})

	// 鉴权
	// 注意：这里变成了 hdl.Register，调用的是实例方法
	h.POST("/api/auth/register", hdl.Register)

	// 注意：Login 和 Refresh 不需要传 secret 了，因为 hdl 内部已经存了
	h.POST("/api/auth/login", hdl.Login)
	h.POST("/api/auth/refresh", hdl.Refresh)

	// ===========================
	// 2. 受保护接口 (Protected)
	// ===========================
	g := h.Group("/api")
	g.Use(middleware.Auth(secret)) // 中间件依然需要 secret

	// --- User & IAM ---
	g.POST("/auth/logout", hdl.Logout)
	g.GET("/auth/me", hdl.Me)

	// 下面的接口你可能还没实现具体的函数，
	// 如果编译报错，请先在 handler 包里创建对应的空函数占位

	g.GET("/api-keys", hdl.ListAPIKeys)
	g.POST("/api-keys", hdl.CreateAPIKey)
	g.DELETE("/api-keys/:id", hdl.RevokeAPIKey)

	// --- Agent Management ---
	g.GET("/agents", hdl.ListAgents)
	g.POST("/agents", hdl.CreateAgent)
	g.GET("/agents/:id", hdl.GetAgent)
	g.PUT("/agents/:id", hdl.UpdateAgent)
	g.DELETE("/agents/:id", hdl.DeleteAgent)

	// --- MCP Ecosystem ---
	// g.GET("/mcp/servers", hdl.ListMCPServers)
	// g.POST("/mcp/servers", hdl.RegisterMCPServer)
	// g.POST("/mcp/servers/:id/sync", hdl.SyncMCPTools)
	// g.GET("/mcp/servers/:id/tools", hdl.ListMCPTools)

	// --- Knowledge Base ---
	// g.GET("/knowledge", hdl.ListKnowledgeBases)
	// g.POST("/knowledge", hdl.CreateKnowledgeBase)
	// g.POST("/knowledge/:id/documents", hdl.UploadDocument)

	// --- Chat & Runtime ---
	// g.GET("/sessions", hdl.ListChatSessions)
	// g.POST("/sessions", hdl.CreateChatSession)
	// g.GET("/sessions/:id", hdl.GetChatSession)
	// g.DELETE("/sessions/:id", hdl.DeleteChatSession)

	// g.GET("/sessions/:id/messages", hdl.ListChatMessages)
	// g.POST("/sessions/:id/chat", hdl.SendChatMessage)

	// --- Observability ---
	// g.GET("/runs", hdl.ListRuns)
	// g.GET("/runs/:id", hdl.GetRunDetail)
	// g.GET("/runs/:id/trace", hdl.GetRunTrace)
	// g.POST("/runs/:id/cancel", hdl.CancelRun)
}
