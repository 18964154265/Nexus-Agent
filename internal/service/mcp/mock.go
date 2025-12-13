package mcp

import (
	"strings"

	"example.com/agent-server/internal/store"
)

// MockToolsForServer 生成模拟数据
// 这是一个纯函数，不依赖外部状态
func MockToolsForServer(serverID, serverName string) []*store.MCPTool {
	tools := []*store.MCPTool{}
	name := strings.ToLower(serverName)

	// 1. 模拟 Git Server
	if strings.Contains(name, "git") {
		tools = append(tools, &store.MCPTool{
			ServerID:    serverID,
			Name:        "git_status",
			Description: "显示工作目录状态。",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"repo_path": map[string]string{"type": "string", "description": "仓库路径,默认当前目录"},
				},
			},
		})
		// ... 其他 git 工具 ...
	}

	// 2. 模拟 Filesystem Server
	if strings.Contains(name, "filesystem") || strings.Contains(name, "fs") {
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
		// ... 其他 fs 工具 ...
	}

	return tools
}
