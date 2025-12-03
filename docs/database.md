# 数据库与模型设计

## 总览
- 目标：支撑用户账号、任务与 Agent 的关系，并为 RAG 与 MCP 能力预留可扩展的结构与接口
- 推荐：PostgreSQL（生产），SQLite（本地开发）。使用 `UUID` 主键、`timestamptz` 时间、`JSONB` 存储灵活结构；RAG 向量检索可选 `pgvector`
- 命名约定：表名复数；审计字段统一：`created_at`、`updated_at`

## 表结构
- `users`
  - `id uuid pk`
  - `email text unique not null`
  - `password_hash text not null`
  - `name text`
  - `roles text[] default '{"user"}'`
  - `status text default 'active'`（active|disabled）
  - `last_login_at timestamptz`
  - `created_at timestamptz not null`
  - `updated_at timestamptz not null`
  - 索引：`(email)` 唯一，`(status)`

- `refresh_tokens`
  - `id uuid pk`
  - `user_id uuid not null references users(id) on delete cascade`
  - `token_hash text unique not null`（只存哈希）
  - `expires_at timestamptz not null`
  - `revoked boolean default false`
  - `created_at timestamptz not null`
  - 索引：`(user_id)`、`(expires_at)`

- `agents`
  - `id uuid pk`
  - `owner_user_id uuid references users(id) on delete set null`
  - `name text not null`
  - `type text not null`（builtin|external|rag|mcp）
  - `capabilities text[] not null`（如 `{"chat","search","rag.query","mcp.call"}`）
  - `concurrency int default 1`
  - `tags text[]`
  - `token_hash text`（不可回读）
  - `config jsonb default '{}'`（RAG/MCP 专用配置）
  - `meta jsonb default '{}'`
  - `last_seen_at timestamptz`
  - `created_at timestamptz not null`
  - `updated_at timestamptz not null`
  - 索引：`(owner_user_id)`、`(type)`、`(capabilities)` GIN、`(tags)` GIN

- `tasks`
  - `id uuid pk`
  - `user_id uuid not null references users(id) on delete cascade`
  - `agent_id uuid references agents(id) on delete set null`
  - `type text not null`（chat|search|rag.query|mcp.call|...）
  - `input jsonb not null`（任务输入载荷）
  - `priority int default 0`
  - `status text not null`（queued|running|done|failed|canceled）
  - `result jsonb`、`error text`
  - `dedupe_key text`（可选去重键，唯一）
  - `parent_task_id uuid references tasks(id) on delete set null`
  - `created_at timestamptz not null`
  - `updated_at timestamptz not null`
  - `started_at timestamptz`、`finished_at timestamptz`
  - 索引：`(user_id)`、`(agent_id)`、`(status)`、`(type)`、`(created_at)`、`(dedupe_key)` 唯一、`(input)` GIN（必要时）

- `task_events`
  - `id uuid pk`
  - `task_id uuid not null references tasks(id) on delete cascade`
  - `event_type text not null`（accept|progress|result|error|cancel）
  - `payload jsonb not null`
  - `ts timestamptz not null`
  - 索引：`(task_id)`、`(ts)`、`(event_type)`

### RAG 相关
- `rag_datasets`
  - `id uuid pk`
  - `agent_id uuid references agents(id) on delete set null`
  - `owner_user_id uuid references users(id) on delete cascade`
  - `name text not null`
  - `description text`
  - `created_at timestamptz not null`
  - `updated_at timestamptz not null`
  - 约束：`unique(owner_user_id, name)`

- `rag_documents`
  - `id uuid pk`
  - `dataset_id uuid not null references rag_datasets(id) on delete cascade`
  - `source_uri text`（文件URL/路径）
  - `mime_type text`
  - `status text not null`（ingested|chunked|indexed）
  - `created_at timestamptz not null`
  - `updated_at timestamptz not null`
  - 索引：`(dataset_id)`、`(status)`

- `rag_chunks`
  - `id uuid pk`
  - `dataset_id uuid not null references rag_datasets(id) on delete cascade`
  - `document_id uuid not null references rag_documents(id) on delete cascade`
  - `content text not null`
  - `metadata jsonb default '{}'`
  - `embedding vector(1536)`（需 `pgvector`，维度按模型调整；SQLite 可替代列为 `blob`）
  - `chunk_index int not null`
  - `created_at timestamptz not null`
  - 索引：`(dataset_id)`、`(document_id)`、`(chunk_index)`、`embedding` 向量索引（IVFFLAT/HNSW）

- `rag_indexes`
  - `id uuid pk`
  - `dataset_id uuid not null references rag_datasets(id) on delete cascade`
  - `engine text not null`（pgvector|milvus|weaviate|qdrant）
  - `status text not null`（building|ready|failed）
  - `config jsonb not null`
  - `created_at timestamptz not null`
  - `updated_at timestamptz not null`

- `rag_queries`
  - `id uuid pk`
  - `dataset_id uuid not null references rag_datasets(id) on delete cascade`
  - `user_id uuid references users(id) on delete set null`
  - `query_text text not null`
  - `top_k int default 5`
  - `filters jsonb default '{}'`
  - `results jsonb`（检索到的 chunk 摘要与分数）
  - `created_at timestamptz not null`
  - 索引：`(dataset_id)`、`(user_id)`、`(created_at)`

### MCP 相关
- `mcp_providers`
  - `id uuid pk`
  - `agent_id uuid references agents(id) on delete cascade`
  - `name text not null`
  - `base_url text not null`
  - `auth_config jsonb default '{}'`（令牌/密钥/OAuth）
  - `status text not null`（active|disabled）
  - `created_at timestamptz not null`
  - `updated_at timestamptz not null`
  - 约束：`unique(agent_id, name)`

- `mcp_tools`
  - `id uuid pk`
  - `provider_id uuid not null references mcp_providers(id) on delete cascade`
  - `name text not null`
  - `description text`
  - `input_schema jsonb not null`
  - `output_schema jsonb not null`
  - `enabled boolean default true`
  - 约束：`unique(provider_id, name)`

- `mcp_sessions`
  - `id uuid pk`
  - `provider_id uuid not null references mcp_providers(id) on delete cascade`
  - `user_id uuid references users(id) on delete set null`
  - `state jsonb default '{}'`
  - `created_at timestamptz not null`
  - `expires_at timestamptz`
  - 索引：`(provider_id)`、`(user_id)`、`(expires_at)`

- `mcp_calls`
  - `id uuid pk`
  - `session_id uuid not null references mcp_sessions(id) on delete cascade`
  - `tool_id uuid not null references mcp_tools(id) on delete cascade`
  - `input jsonb not null`
  - `output jsonb`
  - `status text not null`（queued|running|done|failed|canceled）
  - `error text`
  - `created_at timestamptz not null`
  - `finished_at timestamptz`
  - 索引：`(session_id)`、`(tool_id)`、`(status)`、`(created_at)`

## 关系与约束
- 用户 1..N 任务：`tasks.user_id -> users.id`
- 任务 N..1 Agent：`tasks.agent_id -> agents.id`
- Agent 1..N 能力：使用 `capabilities text[]` 或扩展为 `agent_capabilities(agent_id, capability)` 以便细粒度查询
- RAG：数据集 1..N 文档，文档 1..N chunk，数据集 1..N 索引，数据集 1..N 查询日志
- MCP：Agent 1..N Provider，Provider 1..N Tool，Tool 1..N 调用（经 Session 关联用户）

## 模型（Go 结构体示例）
- Users
  - `ID uuid.UUID`
  - `Email string`、`PasswordHash string`、`Name string`
  - `Roles []string`、`Status string`
  - 时间字段 `time.Time`
- Agents
  - `ID uuid.UUID`、`OwnerUserID *uuid.UUID`
  - `Name string`、`Type string`、`Capabilities []string`
  - `Concurrency int`、`Tags []string`
  - `TokenHash string`、`Config map[string]any`、`Meta map[string]any`
- Tasks
  - `ID uuid.UUID`、`UserID uuid.UUID`、`AgentID *uuid.UUID`
  - `Type string`、`Input map[string]any`、`Priority int`、`Status string`
  - `Result map[string]any`、`Error string`
  - 审计与时间字段
- RAG
  - Dataset/Document/Chunk/Index/Query 对应结构体；`Chunk.Embedding []float32`（或 pgvector 映射）
- MCP
  - Provider/Tool/Session/Call 对应结构体；`Schema map[string]any`

## 接口设计（预留）
- RAG
  - `POST /api/rag/datasets` 创建数据集（name, description, agent_id）
  - `GET /api/rag/datasets` 列表；`GET /api/rag/datasets/:id` 详情
  - `POST /api/rag/datasets/:id/documents` 上传/登记文档（支持文件或 URI）
  - `POST /api/rag/datasets/:id/chunks` 自定义分片（可选）
  - `POST /api/rag/datasets/:id/index` 构建索引（engine, config）
  - `POST /api/rag/query` 检索（dataset_id, query_text, top_k, filters）→ 返回 chunk 摘要与分数

- MCP
  - `POST /api/mcp/providers` 创建 Provider（agent_id, name, base_url, auth_config）
  - `GET /api/mcp/providers` 列表；`GET /api/mcp/providers/:id` 详情
  - `POST /api/mcp/providers/:id/tools/sync` 同步工具定义（从 MCP 拉取）
  - `POST /api/mcp/sessions` 创建会话（provider_id, user_id）
  - `POST /api/mcp/call` 调用工具（session_id, tool_id, input）→ 返回 output / 异步任务ID

- 任务类型与输入约定
  - `rag.query`：`{ dataset_id, query_text, top_k, filters }`
  - `mcp.call`：`{ provider_id, tool_id, session_id?, input }`

## 索引与性能建议
- `JSONB` 字段使用 GIN 索引按需加速过滤
- 向量检索使用 `pgvector` 并建立 `HNSW`/`IVFFLAT` 索引；维度与模型一致
- 高频查询字段建立 B-Tree 索引：`status`、`created_at`、`agent_id`、`user_id`

## 安全与合规
- 敏感信息只存哈希：`token_hash`、`password_hash`
- 所有外键 `on delete` 策略明确：任务随用户删除而清理，Agent 删除后任务可置空关联
- 审计：关键操作记录到 `task_events` 与 RAG/MCP 日志表

## 迁移与初始化
- PostgreSQL 扩展：`CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`、`CREATE EXTENSION IF NOT EXISTS vector;`
- 迁移工具：`golang-migrate`，迁移文件位于 `configs/migrations`
- 开发环境：SQLite 可去掉向量列或用 `blob` 占位；生产切换到 Postgres + pgvector

## 审核点
- 关系是否满足“用户-任务-Agent”主线且能扩展 RAG/MCP
- 约束与索引是否可支撑常见查询与高并发
- 接口是否覆盖创建/登记/检索/调用的关键路径
