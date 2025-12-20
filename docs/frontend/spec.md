# 前端交互设计（Next.js）

## 总览
- 技术栈：Next.js（App Router）、TypeScript、Tailwind 或 Ant Design（均可）、SWR/React Query 管理请求状态
- 后端基础地址：`http://127.0.0.1:8888`
- 认证：登录成功后将 `token` 存储于 `localStorage`，所有受保护接口在 Header 加 `Authorization: Bearer <token>`

## 路由与页面
- `/login`：登录页（含注册入口）
- `/dashboard`：主控制台，包含以下模块：API Keys、Agents、MCP、Knowledge、Sessions/Chat
- `/sessions/[id]`：会话详情与聊天面板

## 请求封装
- `apiClient.ts`：基础封装
  - 自动附带 `Authorization`（从 `localStorage` 读取）
  - `GET/POST/PUT/DELETE` 方法
  - 错误统一弹出（根据 `code/message`）

## 按钮、事件与后端映射

### Auth（/login）
- 按钮：`登录`
  - 事件：`handleLogin`
  - 方法/路由：`POST /api/auth/login`
  - Body：`{ email, password }`
- 按钮：`注册`
  - 事件：`handleRegister`
  - 方法/路由：`POST /api/auth/register`
  - Body：`{ email, password, name }`
- 按钮：`获取我的信息`
  - 事件：`fetchMe`
  - 方法/路由：`GET /api/auth/me`
- 按钮：`注销`
  - 事件：`handleLogout`
  - 方法/路由：`POST /api/auth/logout`
  - Body（可选）：`{ refresh_token }`

### API Keys（/dashboard）
- 按钮：`刷新列表`
  - 事件：`listAPIKeys`
  - 方法/路由：`GET /api/api-keys`
- 按钮：`创建Key`
  - 事件：`createAPIKey`
  - 方法/路由：`POST /api/api-keys`
  - Body：`{ name }`
- 按钮：`吊销`
  - 事件：`revokeAPIKey(id)`
  - 方法/路由：`DELETE /api/api-keys/:id`

### Agents（/dashboard）
- 按钮：`刷新列表`
  - 事件：`listAgents`
  - 方法/路由：`GET /api/agents`
- 按钮：`新建Agent`
  - 事件：`createAgent`
  - 方法/路由：`POST /api/agents`
  - Body：`{ name, description, model_name, system_prompt, temperature, knowledge_base_ids, tags, extra_config, capabilities }`
- 按钮：`查看`
  - 事件：`getAgent(id)`
  - 方法/路由：`GET /api/agents/:id`
- 按钮：`保存`
  - 事件：`updateAgent(id)`
  - 方法/路由：`PUT /api/agents/:id`
  - Body：同上（部分字段）
- 按钮：`删除`
  - 事件：`deleteAgent(id)`
  - 方法/路由：`DELETE /api/agents/:id`

### MCP（/dashboard）
- 按钮：`列出Server`
  - 事件：`listMCPServers(agentId?)`
  - 方法/路由：`GET /api/mcp/servers?agent_id=<id>`
- 按钮：`注册Server`
  - 事件：`registerMCPServer`
  - 方法/路由：`POST /api/mcp/servers`
  - Body：`{ name, transport_type: "sse", agent_id, connection_config }`
- 按钮：`同步Tools`
  - 事件：`syncMCPTools(serverId)`
  - 方法/路由：`POST /api/mcp/servers/:id/sync`
- 按钮：`查看Tools`
  - 事件：`listMCPTools(serverId)`
  - 方法/路由：`GET /api/mcp/servers/:id/tools`

### Knowledge（/dashboard）
- 按钮：`刷新列表`
  - 事件：`listKnowledge`
  - 方法/路由：`GET /api/knowledge`
- 按钮：`新建KB`
  - 事件：`createKnowledge`
  - 方法/路由：`POST /api/knowledge`
  - Body：`{ name, description, is_public, meta_info }`
- 按钮：`上传文档`
  - 事件：`uploadDocument(kbId, file)`
  - 方法/路由：`POST /api/knowledge/:id/documents`
  - 表单：`FormData { file }`

### Sessions & Chat（/dashboard, /sessions/[id]）
- 按钮：`刷新会话`
  - 事件：`listSessions`
  - 方法/路由：`GET /api/sessions`
- 按钮：`新建会话`
  - 事件：`createSession`
  - 方法/路由：`POST /api/sessions`
  - Body：`{ agent_id, title }`
- 按钮：`打开会话`
  - 事件：`openSession(id)` → 跳转 `/sessions/[id]`
- 按钮：`删除会话`
  - 事件：`deleteSession(id)`
  - 方法/路由：`DELETE /api/sessions/:id`
- 按钮：`刷新消息`
  - 事件：`listMessages(sessionId)`
  - 方法/路由：`GET /api/sessions/:id/messages`
- 按钮：`发送`
  - 事件：`sendMessage(sessionId, content)`
  - 方法/路由：`POST /api/sessions/:id/chat`
  - Body：`{ content }`

## 组件结构（建议）
- `AuthForm`：登录/注册表单，暴露 `onLogin/onRegister` 事件
- `Dashboard`
  - `APIKeysPanel`
  - `AgentsPanel`
  - `MCPPanel`
  - `KnowledgePanel`
  - `SessionsPanel`
- `ChatView`
  - 消息列表（区分 `role: user|assistant|tool`）
  - 输入框与发送按钮

## 交互细节
- 成功与错误提示：统一读取响应的 `code/message`；`code=0` 表示成功
- 授权上下文：登录后将 `token` 写入 `localStorage`，在 `apiClient` 的请求拦截中附带到 Header
- 文件上传：使用 `fetch`/`axios` 的 `multipart/form-data`，后端键名为 `file`
- 列表刷新：创建/删除后调用对应 `list*` 方法以更新 UI

## 示例事件到路由映射速查
- `handleLogin` → `POST /api/auth/login`
- `handleRegister` → `POST /api/auth/register`
- `fetchMe` → `GET /api/auth/me`
- `handleLogout` → `POST /api/auth/logout`
- `listAPIKeys` → `GET /api/api-keys`
- `createAPIKey` → `POST /api/api-keys`
- `revokeAPIKey` → `DELETE /api/api-keys/:id`
- `listAgents` → `GET /api/agents`
- `createAgent` → `POST /api/agents`
- `getAgent` → `GET /api/agents/:id`
- `updateAgent` → `PUT /api/agents/:id`
- `deleteAgent` → `DELETE /api/agents/:id`
- `listMCPServers` → `GET /api/mcp/servers?agent_id=`
- `registerMCPServer` → `POST /api/mcp/servers`
- `syncMCPTools` → `POST /api/mcp/servers/:id/sync`
- `listMCPTools` → `GET /api/mcp/servers/:id/tools`
- `listKnowledge` → `GET /api/knowledge`
- `createKnowledge` → `POST /api/knowledge`
- `uploadDocument` → `POST /api/knowledge/:id/documents`
- `listSessions` → `GET /api/sessions`
- `createSession` → `POST /api/sessions`
- `openSession` → 路由跳转 `/sessions/[id]`
- `deleteSession` → `DELETE /api/sessions/:id`
- `listMessages` → `GET /api/sessions/:id/messages`
- `sendMessage` → `POST /api/sessions/:id/chat`
