package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"example.com/agent-server/internal/store"
	"github.com/sashabaranov/go-openai"
)

// executeToolCall 分发并执行工具
func (e *AgentEngine) executeToolCall(ctx context.Context, call openai.ToolCall) (string, error) {
	name := call.Function.Name
	argsJSON := call.Function.Arguments

	// 解析参数
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return fmt.Sprintf("Error parsing arguments: %v", err), nil
	}

	// === 核心：工具路由 ===
	// 这里我们“硬编码”实现 seed.go 里初始化的那些工具
	// 以后你可以把这里改成真正的 MCP Client RPC 调用
	switch name {
	// --- Git Tools ---
	case "git_status":
		return runCmd("git", "status")
	case "git_log":
		return runCmd("git", "log", "--oneline", "-n", "5")
	case "git_diff":
		return runCmd("git", "diff")

	// --- Filesystem Tools ---
	case "list_directory":
		path := "."
		if p, ok := args["path"].(string); ok {
			path = p
		}
		return runCmd("ls", "-F", path)
	case "read_file":
		path, _ := args["path"].(string)
		return runCmd("cat", path)

	// --- Special Tools ---
	case "delegate_to_specialist":
		// 这里是父子 Agent 调度的入口（以后实现）
		return "Mock: Task delegated to specialist.", nil

	default:
		return fmt.Sprintf("Error: Tool '%s' not found or not implemented in runner.", name), nil
	}
}

// runCmd 辅助函数：在服务器本地执行 Shell 命令
// 注意：生产环境必须做安全沙箱限制！这里仅供演示。
func runCmd(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Command failed: %s\nOutput: %s", err, string(out)), nil
	}
	return string(out), nil
}

// saveToolOutput 辅助：保存工具执行结果
func (e *AgentEngine) saveToolOutput(run *store.Run, toolCallID string, output string) {
	e.Store.CreateChatMessage(&store.ChatMessage{
		SessionID:  run.SessionID,
		RunID:      run.ID,
		Role:       "tool",     // 必须是 tool
		ToolCallID: toolCallID, // 必须对应
		Content:    map[string]interface{}{"type": "text", "text": output},
		CreatedAt:  time.Now(),
	})
}
