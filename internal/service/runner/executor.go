package runner

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"example.com/agent-server/internal/service/mcp"
	"example.com/agent-server/internal/store"
	"github.com/sashabaranov/go-openai"
)

// executeToolCall 分发并执行工具
func (e *AgentEngine) executeToolCall(ctx context.Context, call openai.ToolCall) (string, error) {

	// 为了代码跑通，我们临时在这里 new 一个 executor
	// 最佳实践是在 NewEngine 里初始化它
	executor := mcp.NewExecutor(e.Store)

	// 1. 获取工具名和参数
	// 现在 call 是具体类型，编译器能识别 Function 字段了
	toolName := call.Function.Name
	argsJSON := call.Function.Arguments

	// 2. 反查 Tool 对应的 Server
	// 我们需要先找到 tool 属于哪个 server
	toolDef := e.Store.FindMCPToolByName(toolName)
	if toolDef == nil {
		return "", fmt.Errorf("tool definition not found: %s", toolName)
	}

	server := e.Store.GetMCPServer(toolDef.ServerID)
	if server == nil {
		return "", fmt.Errorf("server not found for tool: %s", toolName)
	}

	// 3. 执行
	return executor.ExecuteTool(ctx, server, toolName, argsJSON)
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
func (e *AgentEngine) saveToolOutput(run *store.Run, toolCallID, output string) {
	e.Store.CreateChatMessage(&store.ChatMessage{
		SessionID:  run.SessionID,
		RunID:      run.ID,
		Role:       "tool", // 角色必须是 tool
		Content:    map[string]interface{}{"type": "text", "text": output},
		ToolCallID: toolCallID, // 必须填！跟 Assistant 的 tool_calls[i].id 对应
		CreatedAt:  time.Now(),
		IsHidden:   true, // 前端通常不展示大段的工具日志
	})
}
