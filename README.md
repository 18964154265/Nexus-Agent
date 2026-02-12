# Nexus-Agent

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/Next.js-16-black?style=flat&logo=next.js" alt="Next.js">
  <img src="https://img.shields.io/badge/React-19-61DAFB?style=flat&logo=react" alt="React">
  <img src="https://img.shields.io/badge/License-MIT-green?style=flat" alt="License">
</p>

<p align="center">
  <b>一个支持多 Agent 协作、工具调用和过程观测的 AI Agent 平台</b>
</p>

---

## 特性

- **Multi-Agent 协作** - 预置 DevOps 团队（项目经理、架构师、开发、测试、审计），支持 Agent 间智能切换（Handoff）
- **MCP 工具集成** - 支持 Model Context Protocol，可扩展的工具调用能力（Git、文件系统等）
- **流式输出** - 基于 SSE 的实时流式响应，打字机效果即时反馈
- **执行追踪** - 完整的 Run/Step 追踪，可视化 Agent 思考过程和工具调用链
- **用户认证** - JWT Token 认证，支持多用户隔离
- **灵活存储** - 支持内存存储（开发）和 PostgreSQL（生产）

## 技术架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        Frontend (Next.js 16)                     │
│              React 19 + Tailwind CSS + SWR                       │
└─────────────────────────────────────────────────────────────────┘
                                  │
                                  │ REST API + SSE
                                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Backend (Go + Hertz)                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │   Handler   │──│   Runner    │──│      LLM Client         │  │
│  │  (API层)    │  │ (编排引擎)  │  │  (OpenAI Compatible)    │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
│         │                │                                       │
│         ▼                ▼                                       │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                     Store (Interface)                        ││
│  │            Memory Store  |  PostgreSQL Store                 ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
                                  │
                                  │ MCP Protocol (stdio)
                                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                      MCP Tool Servers                            │
│            Git Server  |  Filesystem Server  |  ...              │
└─────────────────────────────────────────────────────────────────┘
```

## 快速开始

### 环境要求

- Go 1.24+
- Node.js 20+
- PostgreSQL 14+ (可选，开发模式可用内存存储)

### 1. 克隆项目

```bash
git clone https://github.com/18964154265/Nexus-Agent.git
cd Nexus-Agent
```

### 2. 配置环境变量

```bash
cp .env.example .env
```

编辑 `.env` 文件：

```env
# LLM 配置 (支持 OpenAI 兼容接口)
LLM_API_KEY=your-api-key
LLM_BASE_URL=https://api.openai.com/v1
LLM_MODEL_NAME=gpt-4o
LLM_TEMPERATURE=0.1

# 服务配置
PORT=8888
JWT_SECRET=your-secure-jwt-secret

# 数据库配置 (可选)
USE_DB=false
DB_DSN=postgres://user:password@localhost:5432/nexus?sslmode=disable
```

### 3. 启动后端

```bash
# 安装依赖
go mod download

# 运行服务
go run ./cmd/server
```

### 4. 启动前端

```bash
cd frontend

# 安装依赖
npm install

# 开发模式
npm run dev
```

访问 http://localhost:3000 开始使用。

## 📖 核心概念

### Agent（代理）

Agent 是具有特定职责和能力的 AI 实体。每个 Agent 有独立的：
- System Prompt（系统提示词）
- 可用工具集
- 模型配置

**预置 DevOps 团队：**

| Agent | 职责 | 特点 |
|-------|------|------|
| DevOps Manager | 需求分析、任务分派 | 协调团队，不直接编码 |
| Architect | 系统设计、技术选型 | 输出架构图和接口定义 |
| Senior Coder | 代码实现 | 遵循 SOLID 原则 |
| QA Engineer | 单元测试 | 保证 90%+ 覆盖率 |
| Code Reviewer | 代码审查 | 安全性和性能检查 |

### Handoff（切换）

Agent 可以智能判断是否需要将对话转交给更合适的 Agent 处理：

```
用户 → Manager → "这个需求需要架构设计"
                         ↓ Handoff
                    Architect → "输出架构方案"
                         ↓ Handoff  
                    Coder → "实现代码"
```

### MCP Server（工具服务）

基于 Model Context Protocol 的工具扩展机制：

- **git-server** - Git 操作（status, diff, commit...）
- **filesystem-server** - 文件操作（read, write, list...）

### Run & RunStep（执行追踪）

- **Run** - 一次完整的 Agent 执行过程
- **RunStep** - Run 中的单个步骤（思考、工具调用、Handoff）

## 项目结构

```
Nexus-Agent/
├── cmd/
│   └── server/
│       └── main.go              # 入口文件
├── internal/
│   ├── auth/                    # JWT 认证
│   ├── bootstrap/               # 初始化预设数据
│   ├── handler/                 # HTTP Handler
│   ├── http/                    # 路由定义
│   ├── middleware/              # 中间件
│   ├── service/
│   │   ├── llm/                 # LLM 客户端
│   │   ├── mcp/                 # MCP 执行器
│   │   └── runner/              # Agent 执行引擎
│   └── store/                   # 数据存储层
├── pkg/
│   └── response/                # 统一响应格式
├── frontend/
│   ├── src/
│   │   ├── app/                 # Next.js App Router
│   │   ├── components/          # React 组件
│   │   ├── lib/                 # 工具函数
│   │   └── types/               # TypeScript 类型
│   └── package.json
├── .env.example
├── go.mod
└── README.md
```

## 🔌 API 概览

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/auth/register` | 用户注册 |
| POST | `/api/auth/login` | 用户登录 |
| GET | `/api/agents` | 获取 Agent 列表 |
| POST | `/api/agents` | 创建 Agent |
| GET | `/api/sessions` | 获取会话列表 |
| POST | `/api/sessions` | 创建会话 |
| POST | `/api/sessions/:id/chat` | 发送消息 |
| POST | `/api/sessions/:id/chat/stream` | 发送消息（流式） |
| GET | `/api/runs/:id/trace` | 获取执行追踪 |
| GET | `/api/mcp/servers` | 获取 MCP 服务器列表 |

## 🛠️ 开发指南

### 添加新的 Agent

编辑 `internal/bootstrap/seed.go`：

```go
{
    ID:          "your-agent-id",
    Name:        "Your Agent Name",
    Description: "Agent description",
    ModelName:   "gpt-4o",
    Temperature: 0.1,
    SystemPrompt: `Your system prompt here...`,
}
```

### 添加新的 MCP Server

1. 实现符合 MCP 协议的 Server
2. 在 `SeedMCPServers` 中注册配置
3. 关联到目标 Agent

### 自定义 LLM Provider

修改 `internal/service/llm/client.go`，支持任意 OpenAI 兼容接口：

```go
llmConfig := llm.LLMConfig{
    ApiKey:    "your-key",
    BaseURL:   "https://your-provider.com/v1",
    ModelName: "your-model",
}
```

## 贡献

欢迎提交 Issue 和 Pull Request！

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 发起 Pull Request

## License

本项目采用 MIT 协议 - 详见 [LICENSE](LICENSE) 文件。

## 🙏 致谢

- [Hertz](https://github.com/cloudwego/hertz) - 高性能 Go HTTP 框架
- [go-openai](https://github.com/sashabaranov/go-openai) - OpenAI Go SDK
- [Next.js](https://nextjs.org/) - React 全栈框架
- [Tailwind CSS](https://tailwindcss.com/) - 原子化 CSS 框架

---

<p align="center">
  Made by <a href="https://github.com/18964154265">18964154265</a>
</p>
