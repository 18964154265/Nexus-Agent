package handler

import (
	"example.com/agent-server/internal/service"
	"example.com/agent-server/internal/service/runner"
	"example.com/agent-server/internal/store"
)

type Handler struct {
	Store     store.Store
	JWTSecret []byte
	Engine    *runner.AgentEngine
	Svc       *service.Service
}

// 工厂函数：初始化 Handler
func New(s store.Store, secret string, svc *service.Service) *Handler {
	return &Handler{
		Store:     s,
		JWTSecret: []byte(secret),
		Engine:    runner.NewEngine(s, svc.LLM),
		Svc:       svc,
	}
}
