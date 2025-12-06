package handler

import (
	"example.com/agent-server/internal/store"
)

type Handler struct {
	Store     *store.MemoryStore
	JWTSecret []byte // 新增：保存 JWT 密钥
}

// 工厂函数：初始化 Handler
func New(s *store.MemoryStore, secret string) *Handler {
	return &Handler{
		Store:     s,
		JWTSecret: []byte(secret),
	}
}
