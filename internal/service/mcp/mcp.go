package mcp

import (
	"context"
	"fmt"
	"time"

	"example.com/agent-server/internal/store"
	"github.com/google/uuid"
)

type MCPService struct {
	Store    *store.MemoryStore
	Executor *Executor
}

func NewMCPService(s *store.MemoryStore) *MCPService {
	return &MCPService{Store: s, Executor: NewExecutor(s)}
}

// SyncTools 负责同步一个 Server 下的所有工具
func (s *MCPService) SyncTools(ctx context.Context, serverID string) (int, error) {
	// 1. 查库
	server := s.Store.GetMCPServer(serverID)
	if server == nil {
		return 0, fmt.Errorf("server not found: %s", serverID)
	}

	// 2. 获取工具列表 (Discovery)
	// 在 MVP 阶段，我们在这里调用 Mock 逻辑
	// 在真实阶段，这里会调用 JSON-RPC Client 去连远程
	tools, err := s.fetchToolsFromSource(ctx, server)
	if err != nil {
		return 0, err
	}

	// 3. 更新数据库 (全量覆盖或增量更新)
	// 简单起见，我们直接 Create 新的 (注意：store层最好加个 Upsert 或去重)
	count := 0
	for _, t := range tools {
		// 补全 ID 和 时间
		t.ID = uuid.New().String()
		t.CreatedAt = time.Now()
		t.UpdatedAt = time.Now()

		// 存库
		s.Store.CreateMCPTool(t)
		count++
	}

	return count, nil
}

// fetchToolsFromSource 抽象了工具来源（Mock vs Real）
func (s *MCPService) fetchToolsFromSource(ctx context.Context, server *store.MCPServer) ([]*store.MCPTool, error) {
	// 如果是 demo 阶段，走 Mock
	// 实际代码中可以用一个 flag 或者 server.TransportType 来判断
	return MockToolsForServer(server.ID, server.Name), nil
}
