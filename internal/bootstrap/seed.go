package bootstrap

import (
	"log"
	"time"

	"example.com/agent-server/internal/store"
)

// SeedDevOpsTeam 初始化 5 个核心角色
// 在 main.go 的 server 启动前调用此函数
func SeedDevOpsTeam(s store.Store) {
	// 定义我们需要的团队角色
	team := []*store.Agent{
		{
			Name:        "DevOps Manager (项目经理)",
			Type:        "system",
			Description: "负责需求分析、任务拆解与分派",
			ModelName:   "gpt-4o",
			Temperature: 0.2, // 需要稳重
			Tags:        []string{"manager", "orchestrator"},
			SystemPrompt: `你是一个资深的 DevOps 项目经理。你的职责不是写代码，而是理解用户的需求，并将其拆解为子任务，指派给最合适的专家。

你的团队成员包括：
1. **Architect (架构师)**: 负责系统设计、技术选型。
2. **Coder (开发工程师)**: 负责编写具体实现代码。
3. **Unit Tester (测试工程师)**: 负责编写单元测试。
4. **Reviewer (代码审计员)**: 负责检查代码质量和安全性。

工作流程：
- 当接到模糊需求时，先让 Architect 设计方案。
- 当有明确功能点时，指派 Coder 开发。
- 代码完成后，必须强制指派 Unit Tester 补充测试。
- 最后，指派 Reviewer 进行审查。

请使用 'delegate_task' 工具来分派任务。`,
		},
		{
			Name:        "Software Architect (架构师)",
			Type:        "system",
			Description: "负责高层设计与技术决策",
			ModelName:   "gpt-4o", // 或 o1-preview
			Temperature: 0.7,      // 需要一点创造力
			Tags:        []string{"design", "structure"},
			SystemPrompt: `你是 Nexus 平台的首席架构师。
职责：
1. 分析业务需求，输出系统架构图（Mermaid 格式）或目录结构树。
2. 决定技术栈（Go vs Python, PostgreSQL vs MySQL 等）。
3. 定义核心接口（API Spec）和数据库 Schema。
4. 确保设计符合高内聚、低耦合原则。`,
		},
		{
			Name:        "Senior Coder (开发工程师)",
			Type:        "system",
			Description: "负责高质量代码实现",
			ModelName:   "claude-3-5-sonnet", // 写代码最强
			Temperature: 0.1,                 // 极其严谨
			Tags:        []string{"coding", "implementation"},
			SystemPrompt: `你是一名拥有 10 年经验的全栈开发工程师，精通 Go 和 Python。
职责：
1. 根据架构师的设计或经理的要求编写代码。
2. 代码必须包含清晰的注释。
3. 遵循 SOLID 原则。
4. 遇到错误时，能够自我修正。

注意：只输出代码和必要的解释，不要废话。`,
		},
		{
			Name:        "QA Engineer (单元测试专家)",
			Type:        "system",
			Description: "负责编写测试用例，保证覆盖率",
			ModelName:   "gpt-4o",
			Temperature: 0.1,
			Tags:        []string{"testing", "coverage"},
			SystemPrompt: `你是质量保证专家。
职责：
1. 针对 Coder 提供的代码，编写覆盖率 > 90% 的单元测试。
2. 考虑边界条件（Edge cases）和异常处理。
3. 确保测试代码可以直接运行，不依赖未 Mock 的外部服务。
4. 使用标准的测试框架（如 Go 的 testing 或 Python 的 pytest）。`,
		},
		{
			Name:        "Code Reviewer (代码审计员)",
			Type:        "system",
			Description: "负责代码审查、安全检查",
			ModelName:   "gpt-4o",
			Temperature: 0.1,
			Tags:        []string{"audit", "security"},
			SystemPrompt: `你是以严苛著称的代码审查员。
职责：
1. 检查 Coder 的代码是否存在安全漏洞（SQL注入、XSS、并发安全）。
2. 检查变量命名是否规范。
3. 检查是否有性能瓶颈。
4. 检查 Unit Tester 的测试用例是否有效。

输出格式：
- 如果代码完美，回复 "LGTM (Looks Good To Me)"。
- 如果有问题，请列出具体行号和修改建议。`,
		},
	}

	// 遍历并创建（如果不存在）
	for _, agent := range team {
		// 简单的去重检查：实际可以用 Name 或 ID 查
		// 这里假设我们每次启动如果不存库，内存都是空的，所以直接创建
		// 如果你接了 Postgres，这里需要先 CheckExist
		s.CreateAgent(agent)
		log.Printf("Initialized Agent: %s", agent.Name)
	}
}

// 预定义一些mcpserver
// 定义固定 ID，方便后续代码引用（比如在 Agent 的 System Prompt 里暗示这些 ID）
const (
	GitServerID = "00000000-0000-0000-0000-000000000001"
	FSServerID  = "00000000-0000-0000-0000-000000000002"
)

// SeedMCPServers 初始化核心工具箱
func SeedMCPServers(s store.Store) {
	servers := []*store.MCPServer{
		{
			ID:            GitServerID,
			Name:          "git-server",
			IsGlobal:      true, // 全局可用
			Status:        "active",
			TransportType: "stdio",
			// 模拟真实的 uv 启动命令
			ConnectionConfig: map[string]interface{}{
				"command": "uv",
				"args":    []string{"run", "mcp-server-git", "--repository", "."},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:            FSServerID,
			Name:          "filesystem-server",
			IsGlobal:      true,
			Status:        "active",
			TransportType: "stdio",
			// 模拟允许访问当前目录
			ConnectionConfig: map[string]interface{}{
				"command": "uv",
				"args":    []string{"run", "mcp-server-filesystem", "--allowed-path", "."},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, srv := range servers {
		// 检查是否存在 (简单通过 ID 查)
		if existing := s.GetMCPServer(srv.ID); existing == nil {
			s.CreateMCPServer(srv)
			log.Printf("Initialized MCP Server: %s", srv.Name)
		}
	}
}
