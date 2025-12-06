package main

import (
	"github.com/cloudwego/hertz/pkg/app/server"

	"example.com/agent-server/internal/bootstrap"
	"example.com/agent-server/internal/handler"
	myhttp "example.com/agent-server/internal/http"
	"example.com/agent-server/internal/middleware"
	"example.com/agent-server/internal/store"
)

func main() {
	// 1. 【核心修改】使用构造函数初始化 DB，确保所有 map 都已 make
	db := store.NewMemoryStore()

	// 2. 初始化预设的 Agent 团队 (Manager, Coder 等)
	bootstrap.SeedDevOpsTeam(db)
	bootstrap.SeedMCPServers(db)

	// 3. 初始化 Handler (注入 db)
	// 注意：jwt-secret 应该从环境变量读取，这里为了演示写死
	h := handler.New(db, "your-secure-jwt-secret-key")

	// 4. 初始化 Hertz Server
	srv := server.Default()

	//5.注册中间件
	srv.Use(middleware.Recovery())
	srv.Use(middleware.RequestID())
	srv.Use(middleware.AccessLog())
	srv.Use(middleware.Cors())

	// 5. 注册路由
	myhttp.RegisterRoutes(srv, h, "your-secure-jwt-secret-key")

	// 6. 启动服务
	srv.Spin()
}
