package service

import (
	"example.com/agent-server/internal/service/llm"
	"example.com/agent-server/internal/service/mcp"
	"example.com/agent-server/internal/store"
)

type Service struct {
	Store *store.MemoryStore
	LLM   *llm.Client
	MCP   *mcp.MCPService // <--- 新增
}

func NewService(s *store.MemoryStore, l *llm.Client) *Service {
	return &Service{
		Store: s,
		LLM:   l,
		MCP:   mcp.NewMCPService(s), // <--- 初始化
	}
}
