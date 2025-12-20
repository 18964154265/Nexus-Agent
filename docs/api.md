# Nexus-Agent API 文档（Postman风格）

## 说明
- 基础路径：`/`
- 鉴权：受保护接口需在 Header 中携带 `Authorization: Bearer <access_token>`（登录获取）
- 响应封装：统一为
  - 成功：`{ code: 0, message: "success|created", data: <payload>, request_id: <id> }`
  - 失败：`{ code: <biz_code>, message: <error>, data: null, request_id: <id> }`

---

## 公共接口（Public）

### Health Check
- Method: `GET`
- URL: `/health`
- Auth: 无
- Response 示例：
```
OK
```

### Register
- Method: `POST`
- URL: `/api/auth/register`
- Body(JSON):
```
{
  "email": "user@example.com",
  "password": "secret",
  "name": "Alice"
}
```
- Success Response:
```
{
  "code": 0,
  "message": "created",
  "data": { "message": "User registered successfully", "user_id": "<uuid>" },
  "request_id": "..."
}
```

### Login
- Method: `POST`
- URL: `/api/auth/login`
- Body(JSON):
```
{ "email": "user@example.com", "password": "secret" }
```
- Success Response:
```
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "<access_token>",
    "refresh_token": "<refresh_token>",
    "user": { "id": "<uuid>", "email": "user@example.com", "name": "Alice", "roles": ["user"], "created_at": "..." }
  },
  "request_id": "..."
}
```

### Refresh Token
- Method: `POST`
- URL: `/api/auth/refresh`
- Body(JSON):
```
{ "refresh_token": "<refresh_token>" }
```
- Success Response:
```
{ "code": 0, "message": "success", "data": { "access_token": "<token>" }, "request_id": "..." }
```

---

## 受保护接口（Protected）
所有请求需携带 Header：`Authorization: Bearer <access_token>`

### Logout
- Method: `POST`
- URL: `/api/auth/logout`
- Body(JSON，可选):
```
{ "refresh_token": "<refresh_token>" }
```
- Success Response：`{ code: 0, message: "success", data: { "message": "Logged out" } }`

### Me
- Method: `GET`
- URL: `/api/auth/me`
- Success Response：
```
{ "code": 0, "message": "success", "data": { "id": "<uuid>", "email": "user@example.com", "name": "Alice", "roles": ["user"], "created_at": "..." } }
```

---

## API Keys

### List API Keys
- Method: `GET`
- URL: `/api/api-keys`
- Success Response：
```
{ "code": 0, "message": "success", "data": [ { "id": "...", "name": "Macbook CLI", "prefix": "sk-nx-ab...", "last_used_at": "...", "created_at": "..." } ] }
```

### Create API Key
- Method: `POST`
- URL: `/api/api-keys`
- Body(JSON):
```
{ "name": "Macbook CLI" }
```
- Success Response（仅创建时返回明文key）：
```
{ "code": 0, "message": "created", "data": { "id": "...", "name": "Macbook CLI", "prefix": "sk-nx-ab...", "key": "sk-nx-<hex>", "created_at": "..." } }
```

### Revoke API Key
- Method: `DELETE`
- URL: `/api/api-keys/:id`
- Success Response：`{ "code": 0, "message": "success", "data": { "message": "API Key revoked" } }`

---

## Agents

### List Agents
- Method: `GET`
- URL: `/api/agents`
- Success Response：`{ "code": 0, "message": "success", "data": [<Agent>] }`

### Create Agent
- Method: `POST`
- URL: `/api/agents`
- Body(JSON):
```
{
  "name": "Dev Architect",
  "description": "System design",
  "model_name": "gpt-4o",
  "system_prompt": "You are an architect...",
  "temperature": 0.7,
  "knowledge_base_ids": ["<kb-uuid>"],
  "tags": ["arch"],
  "extra_config": {"domain":"infra"},
  "capabilities": ["code","review"]
}
```
- Success Response：`{ "code": 0, "message": "created", "data": { <Agent> } }`

### Get Agent
- Method: `GET`
- URL: `/api/agents/:id`
- Success Response：`{ "code": 0, "message": "success", "data": { <Agent> } }`

### Update Agent
- Method: `PUT`
- URL: `/api/agents/:id`
- Body(JSON)：同 Create，支持更新字段
- Success Response：`{ "code": 0, "message": "success", "data": { "message": "Agent updated" } }`

### Delete Agent
- Method: `DELETE`
- URL: `/api/agents/:id`
- Success Response：`{ "code": 0, "message": "success", "data": { "message": "Deleted" } }`

---

## MCP Ecosystem

### List MCP Servers
- Method: `GET`
- URL: `/api/mcp/servers`
- Query（可选）：`agent_id=<uuid>`
- Success Response：`{ "code": 0, "message": "success", "data": [ { <MCPServer>, "created_at":"...","updated_at":"...","tool_count":0 } ] }`

### Register MCP Server
- Method: `POST`
- URL: `/api/mcp/servers`
- Body(JSON):
```
{
  "name": "filesystem",
  "transport_type": "sse",
  "agent_id": "<uuid>",
  "connection_config": {"url":"https://mcp.example.com/endpoint"}
}
```
- Success Response：`{ "code": 0, "message": "created", "data": { "id": "<uuid>", "data": { <MCPServerResp> } } }`

### Sync MCP Tools
- Method: `POST`
- URL: `/api/mcp/servers/:id/sync`
- Success Response：`{ "code": 0, "message": "success", "data": { "message":"Sync successful", "server_name":"...", "sync_count": 4 } }`

### List MCP Tools
- Method: `GET`
- URL: `/api/mcp/servers/:id/tools`
- Success Response：`{ "code": 0, "message": "success", "data": [ { <MCPTool>, "created_at":"..." } ] }`

---

## Knowledge Base

### List Knowledge Bases
- Method: `GET`
- URL: `/api/knowledge`
- Success Response：`{ "code": 0, "message": "success", "data": [ { <KnowledgeBase>, "created_at":"...", "updated_at":"...", "document_count": 0 } ] }`

### Create Knowledge Base
- Method: `POST`
- URL: `/api/knowledge`
- Body(JSON):
```
{
  "name": "project-docs",
  "description": "Project documents",
  "is_public": false,
  "meta_info": {"domain":"docs"}
}
```
- Success Response：`{ "code": 0, "message": "created", "data": { <KBResp> } }`

### Upload Document
- Method: `POST`
- URL: `/api/knowledge/:id/documents`
- Headers：`Content-Type: multipart/form-data`
- FormData：`file=<选择文件>`
- Success Response：
```
{ "code": 0, "message": "success", "data": { "filename":"...", "size":12345, "chunks_created": 12, "status":"indexed" } }
```

---

## Chat & Runtime

### List Chat Sessions
- Method: `GET`
- URL: `/api/sessions`
- Success Response：`{ "code": 0, "message": "success", "data": [ { <ChatSession>, "created_at":"...", "updated_at":"..." } ] }`

### Create Chat Session
- Method: `POST`
- URL: `/api/sessions`
- Body(JSON):
```
{ "agent_id": "<uuid>", "title": "Chat with Dev Architect" }
```
- Success Response：`{ "code": 0, "message": "created", "data": { <ChatSessionResp> } }`

### Get Chat Session
- Method: `GET`
- URL: `/api/sessions/:id`
- Success Response：`{ "code": 0, "message": "success", "data": { <ChatSessionResp> } }`

### Delete Chat Session
- Method: `DELETE`
- URL: `/api/sessions/:id`
- Success Response：`{ "code": 0, "message": "success", "data": { "message":"Deleted" } }`

### List Chat Messages 
- Method: `GET`
- URL: `/api/sessions/:id/messages`
- Success Response：`{ "code": 0, "message": "success", "data": [ { "id":"...","role":"user|assistant|tool","content":{...},"tool_call_id":"...","created_at":"..." } ] }`

### Send Chat Message
- Method: `POST`
- URL: `/api/sessions/:id/chat`
- Body(JSON):
```
{ "content": "查看当前目录有什么文件？" }
```
- Success Response：`{ "code": 0, "message": "success", "data": { "id":"...","role":"assistant","content":{ "type":"text","text":"..." }, "created_at":"..." } }`

---

## 备注
- `<Agent>`、`<MCPServer>`、`<MCPTool>`、`<KnowledgeBase>`、`<ChatSession>` 等实体字段请参考 `docs/datatransferobj.md`（响应型 DTO 嵌入的实体不在本文件展开）。
- 错误响应的 `code` 为业务码（如 40400/40300/50000），`message` 为人类可读文案。请在客户端基于 `HTTP Status` 与 `code` 均衡处理。 
