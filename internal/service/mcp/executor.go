package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"example.com/agent-server/internal/store"
)

// Executor 负责执行具体的工具逻辑
type Executor struct {
	Store store.Store
}

func NewExecutor(s store.Store) *Executor {
	return &Executor{Store: s}
}

// ExecuteTool 执行工具
// server: 目标 Server
// toolName: 工具名 (e.g. "git_status")
// argsJSON: LLM 生成的 JSON 参数字符串 (e.g. '{"path": "."}')
func (e *Executor) ExecuteTool(ctx context.Context, server *store.MCPServer, toolName string, argsJSON string) (string, error) {
	log.Printf("[MCP Executor] Executing %s on server %s. Args: %s", toolName, server.Name, argsJSON)

	// 1. 解析参数
	var args map[string]interface{}
	// 允许空参数的情况
	if argsJSON != "" && argsJSON != "{}" {
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return "", fmt.Errorf("invalid json args: %v", err)
		}
	} else {
		args = make(map[string]interface{})
	}

	// 2. 路由分发 (Routing)
	// 根据 Server 的名字或类型，分发到不同的处理函数
	// 注意：在真实 MCP 中，这里应该是通过 stdio/sse 发送 JSON-RPC 请求
	// 但在 MVP 中，我们直接调用内部实现的 Go 函数
	serverName := strings.ToLower(server.Name)

	switch {
	case strings.Contains(serverName, "git"):
		return e.handleGit(ctx, toolName, args)

	case strings.Contains(serverName, "filesystem") || strings.Contains(serverName, "fs"):
		return e.handleFilesystem(ctx, toolName, args)

	// 未来扩展：
	// case server.TransportType == "sse":
	//     return e.callRemoteSSE(ctx, server, toolName, args)

	default:
		return "", fmt.Errorf("server implementation not found for: %s", serverName)
	}
}

// =============================================================================
// Module 1: Git Implementation
// =============================================================================

func (e *Executor) handleGit(ctx context.Context, tool string, args map[string]interface{}) (string, error) {
	switch tool {
	case "git_status":
		return runCommand("git", "status")

	case "git_diff":
		target, _ := args["target"].(string)
		if target == "" {
			target = "HEAD"
		}
		return runCommand("git", "diff", target)

	case "git_commit":
		msg, ok := args["message"].(string)
		if !ok || msg == "" {
			return "", fmt.Errorf("git_commit requires 'message'")
		}
		// 简单的 commit，实际可能需要处理 add_all
		addAll, _ := args["add_all"].(bool)
		if addAll {
			runCommand("git", "add", ".")
		}
		return runCommand("git", "commit", "-m", msg)

	case "git_log":
		return runCommand("git", "log", "-n", "10", "--oneline")

	default:
		return "", fmt.Errorf("unknown git tool: %s", tool)
	}
}

// =============================================================================
// Module 2: Filesystem Implementation
// =============================================================================

func (e *Executor) handleFilesystem(ctx context.Context, tool string, args map[string]interface{}) (string, error) {
	// 获取当前工作目录，作为根目录
	cwd, _ := os.Getwd()

	switch tool {
	case "list_directory":
		path, _ := args["path"].(string)
		if path == "" {
			path = "."
		}

		targetPath := filepath.Join(cwd, path)
		entries, err := os.ReadDir(targetPath)
		if err != nil {
			return "", fmt.Errorf("ls error: %v", err)
		}

		var names []string
		for _, ent := range entries {
			suffix := ""
			if ent.IsDir() {
				suffix = "/"
			}
			names = append(names, ent.Name()+suffix)
		}
		return strings.Join(names, "\n"), nil

	case "read_file":
		path, ok := args["path"].(string)
		if !ok {
			return "", fmt.Errorf("missing path")
		}

		targetPath := filepath.Join(cwd, path)
		content, err := os.ReadFile(targetPath)
		if err != nil {
			return "", fmt.Errorf("read error: %v", err)
		}
		return string(content), nil

	case "write_file":
		path, ok := args["path"].(string)
		content, ok2 := args["content"].(string)
		if !ok || !ok2 {
			return "", fmt.Errorf("missing path or content")
		}

		targetPath := filepath.Join(cwd, path)
		// 0644 权限写入
		if err := os.WriteFile(targetPath, []byte(content), 0644); err != nil {
			return "", fmt.Errorf("write error: %v", err)
		}
		return fmt.Sprintf("Successfully wrote to %s", path), nil

	case "search_files":
		// 简单的 Grep 实现
		pattern, ok := args["pattern"].(string)
		if !ok {
			return "", fmt.Errorf("missing pattern")
		}
		return runCommand("grep", "-r", pattern, ".")

	default:
		return "", fmt.Errorf("unknown fs tool: %s", tool)
	}
}

// =============================================================================
// Helper: Command Runner
// =============================================================================

func runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	// 在生产环境中，这里应该设置 cmd.Dir 到用户的 Workspace 目录
	// cmd.Dir = "/tmp/workspace/user_123"

	output, err := cmd.CombinedOutput()
	result := string(output)

	if err != nil {
		// 注意：即使报错（比如 git status 报 fatal），也应该返回 output 给 LLM，
		// 因为 output 里包含了错误原因，LLM 可以据此自我修正。
		return fmt.Sprintf("Command failed: %v\nOutput:\n%s", err, result), nil
	}

	return result, nil
}
