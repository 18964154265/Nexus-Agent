package handler

import (
	"example.com/agent-server/internal/service/runner"
	"example.com/agent-server/internal/store"
)

type Handler struct {
	Store     *store.MemoryStore
	JWTSecret []byte
	Engine    *runner.AgentEngine
}

// 工厂函数：初始化 Handler
func New(s *store.MemoryStore, secret string) *Handler {
	return &Handler{
		Store:     s,
		JWTSecret: []byte(secret),
		Engine:    runner.NewEngine(s),
	}
}
