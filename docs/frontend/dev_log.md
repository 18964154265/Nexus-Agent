# 前端开发日志

## 项目概述
Nexus Agent 前端采用 Next.js (App Router) + TypeScript + Tailwind CSS 构建，提供现代化的 DevOps 多智能体管理界面。

## 技术栈
- **框架**: Next.js 16.1.0 (App Router)
- **语言**: TypeScript
- **样式**: Tailwind CSS 4
- **状态管理**: SWR (数据获取)
- **HTTP 客户端**: Axios
- **表单处理**: React Hook Form
- **图标**: Lucide React
- **代码编辑器**: react-simple-code-editor + Prism.js (JSON 语法高亮)
- **工具库**: clsx, tailwind-merge

## 开发进度

### 1. 项目初始化 ✅
- 基于 Next.js 16.1.0 创建项目
- 配置 TypeScript 和 Tailwind CSS
- 安装核心依赖包

### 2. 基础架构 ✅

#### 2.1 API 客户端封装 (`src/lib/apiClient.ts`)
- 创建 Axios 实例，配置基础 URL
- 实现请求拦截器：自动注入 JWT Token
- 实现响应拦截器：
  - 统一返回 `response.data`（后端标准格式：`{ code, message, data }`）
  - 处理 401 未授权错误，自动跳转登录页
  - 统一错误格式

#### 2.2 类型定义 (`src/types/index.ts`)
- 定义统一的 `ApiResponse<T>` 响应结构
- 定义所有业务实体类型：
  - `User`, `LoginResp` (认证相关)
  - `APIKey` (API 密钥)
  - `Agent` (智能体)
  - `MCPServer`, `MCPTool` (MCP 服务)
  - `KnowledgeBase` (知识库)
  - `ChatSession`, `ChatMessage` (聊天会话)
  - `Run`, `RunStep` (任务执行)

### 3. 页面开发 ✅

#### 3.1 登录页面 (`/login`)
- 实现登录/注册切换功能
- 表单验证（邮箱、密码）
- 错误提示显示
- 登录成功后保存 Token 并跳转 Dashboard
- 使用翡翠绿（Emerald）作为主题色

#### 3.2 Dashboard 布局 (`/dashboard`)
- **侧边栏导航**：
  - 概览、会话、Agents、知识库、MCP 服务、API Keys
  - 退出登录按钮
- **顶部栏**：用户信息显示
- **内容区**：可滚动的主内容区域
- **首页**：快速入口卡片

#### 3.3 Agents 管理模块 (`/dashboard/agents`)

**列表页** (`/dashboard/agents`):
- 使用 SWR 获取 Agent 列表
- 卡片式布局展示（响应式网格）
- 显示 Agent 名称、模型、描述、创建时间
- 删除功能（带确认）

**创建页** (`/dashboard/agents/new`):
- **左右分栏布局**（60% 配置 + 40% 预览）
- **配置区（Tabs）**：
  - **基础信息**：名称、描述、标签（支持回车添加、点击删除）
  - **模型设定**：
    - 模型选择（GPT-4o, Claude 3.5 Sonnet, GPT-3.5 Turbo）
    - 温度滑块（0-2）
    - 系统提示词编辑器（GitHub 风格 JSON 编辑器，白色背景）
  - **能力装配 (MCP)**：MCP 服务器多选（待完善工具列表显示）
  - **知识库 (RAG)**：知识库多选
- **预览区**：迷你聊天窗口，用于测试 Prompt 效果（Mock 模式）

#### 3.4 MCP Servers 管理页面 (`/dashboard/mcp`)
- **页面头部**：标题、副标题、注册按钮
- **过滤标签**：All / System / Private
- **服务器卡片网格**（响应式：移动 1 列，平板 2 列，桌面 3 列）
- **MCPServerCard 组件**：
  - 动态图标（Git、Database、Terminal）
  - System/Private 徽章
  - 状态指示器（绿色/红色点）
  - 传输类型显示（STDIO/SSE）
  - 工具数量显示
  - 连接信息（代码块样式）
  - 操作按钮：同步、查看工具、删除
- **注册模态框**：
  - 支持 STDIO 和 SSE 两种传输类型
  - STDIO：Command + Args 输入
  - SSE：URL 输入
- **工具抽屉**：侧边滑出，显示工具列表和输入 Schema（可展开查看 JSON）

#### 3.5 API Keys 管理页面 (`/dashboard/apikeys`)
- **页面头部**：标题、副标题、生成按钮
- **表格展示**：
  - 列：Name、Token（前缀徽章）、Status/Usage、Created、Actions
  - 相对时间显示（"2 hours ago"、"Never used"）
  - Revoke 按钮（带确认提示）
- **两步骤生成模态框**：
  - **第一步**：输入名称，点击 Generate
  - **第二步**：
    - 显示完整 API Key（只读输入框）
    - Copy 按钮（复制后显示 "Copied"）
    - 黄色警告提示：提醒用户保存密钥（无法再次查看）
    - Done 按钮关闭

### 4. UI/UX 优化 ✅
- **主题色**：从蓝色改为翡翠绿（Emerald），更柔和护眼
- **响应式设计**：所有页面支持移动端、平板、桌面
- **交互反馈**：加载状态、成功/错误提示、悬停效果

### 5. 待完成功能 ⏳
- [ ] 会话管理页面 (`/dashboard/sessions`)
- [ ] 聊天界面 (`/sessions/[id]`)
- [ ] 知识库管理页面 (`/dashboard/knowledge`)
- [ ] 真实 API 集成（替换 Mock 数据）
- [ ] 错误处理和 Toast 通知系统
- [ ] 用户信息获取和显示
- [ ] 路由保护（未登录自动跳转）

## 文件结构
```
frontend/
├── src/
│   ├── app/
│   │   ├── layout.tsx              # 根布局
│   │   ├── page.tsx                # 首页（重定向到 /login）
│   │   ├── login/
│   │   │   └── page.tsx           # 登录页
│   │   └── dashboard/
│   │       ├── layout.tsx         # Dashboard 布局（侧边栏+顶部栏）
│   │       ├── page.tsx            # Dashboard 首页
│   │       ├── agents/
│   │       │   ├── page.tsx       # Agents 列表页
│   │       │   └── new/
│   │       │       └── page.tsx   # 创建 Agent 页
│   │       ├── mcp/
│   │       │   └── page.tsx       # MCP Servers 管理页
│   │       └── apikeys/
│   │           └── page.tsx       # API Keys 管理页
│   ├── lib/
│   │   └── apiClient.ts           # API 客户端封装
│   └── types/
│       └── index.ts                # TypeScript 类型定义
```

## 开发注意事项

### API 调用规范
- 所有 API 调用使用 `apiClient`，自动注入 Token
- 响应格式统一为 `ApiResponse<T>`：`{ code: number, message: string, data: T }`
- `code === 0` 表示成功，否则为业务错误

### 类型安全
- 所有 API 响应使用 `ApiResponse<T>` 类型
- 业务实体使用 `@/types` 中定义的类型
- 避免使用 `any`，必要时使用类型断言

### 状态管理
- 使用 SWR 进行数据获取和缓存
- 表单状态使用 React Hook Form
- 本地 UI 状态使用 `useState`

### 样式规范
- 使用 Tailwind CSS 工具类
- 主题色：翡翠绿（emerald-600, emerald-500 等）
- 响应式断点：`md:` (768px), `lg:` (1024px)

## 已知问题
1. 部分页面仍使用 Mock 数据，需要替换为真实 API 调用
2. 错误提示使用 `alert()`，后续应集成 Toast 通知系统
3. MCP Tools 列表显示功能待完善（需要后端接口支持）

## 下一步计划
1. 实现会话管理页面和聊天界面
2. 实现知识库管理页面
3. 集成真实 API，移除 Mock 数据
4. 添加 Toast 通知系统
5. 完善错误处理和加载状态
6. 添加路由保护中间件

