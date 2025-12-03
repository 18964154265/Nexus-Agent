# Hertz Agent 后端项目文档

## 概述
- 目标：提供网页端 Agent 的后端能力，涵盖 `HTTP` 基本接口、`用户鉴权`、`WebSocket` 双向通信、`任务调度`、`Agent 配置管理`。
- 框架：`cloudwego/hertz`，强调高性能、可扩展中间件。
- 架构：`REST API` + `WebSocket`，任务通过调度器路由到在线 Agent；所有状态以事件流或查询接口暴露。

## 环境与依赖
- 语言与运行时：`Go >= 1.22`，操作系统：`windows`、`linux`、`macOS`。
- 主框架：`github.com/cloudwego/hertz`。
- 数据库：开发可用 `SQLite`，生产建议 `PostgreSQL`。ORM 建议 `gorm.io/gorm`（驱动：`gorm.io/driver/sqlite` / `gorm.io/driver/postgres`）。
- 缓存与队列：`Redis >= 6`（任务队列、连接状态、限流）。
- JWT：`github.com/golang-jwt/jwt/v5`（生成与校验令牌）。
- WebSocket：`nhooyr.io/websocket`（与 Hertz 兼容，API 简洁）。
- 调度器：`github.com/go-co-op/gocron`（定时任务），或自研基于 Redis 的延时队列。
- 迁移：`github.com/golang-migrate/migrate/v4`（数据库迁移，推荐）。
- 观测：`github.com/hertz-contrib/pprof`、`prometheus`（可选）。

## 目录结构（建议）
- `cmd/server`：主服务入口 `main.go`。
- `internal/http`：路由、控制器、请求响应模型。
- `internal/middleware`：鉴权、日志、恢复、CORS、限流。
- `internal/ws`：WebSocket 会话管理、协议编解码。
- `internal/scheduler`：任务调度、队列、状态维护。
- `internal/agent`：Agent 配置与注册、心跳管理。
- `internal/store`：数据库与缓存（`gorm`、`redis` 适配）。
- `pkg/types`：公共类型与错误码。
- `configs`：配置文件模板与示例。
- `docs`：项目文档与 API 说明。

## 配置
- 支持 `env` + `yaml`。
- 必要环境变量：
  - `APP_ENV=dev|prod`
  - `HTTP_ADDR=:8080`
  - `JWT_SECRET=<随机 32+ 字节>`
  - `DB_DSN=postgres://user:pass@host:5432/dbname?sslmode=disable` 或 `sqlite://./data/app.db`
  - `REDIS_ADDR=127.0.0.1:6379`
  - `REDIS_PASSWORD=`
- 配置文件示例：`configs/app.yaml`

## 中间件
- 日志：结构化日志（请求 ID、耗时、状态码）。
- 恢复：panic 保护与统一错误响应。
- CORS：`Origin` 白名单与凭证策略。
- 鉴权：`JWT` 校验，`Bearer` 头或 `cookie`。
- 限流：基于 `Redis` 的漏桶/令牌桶（按 IP/用户）。
- 请求 ID：统一 `X-Request-ID`。

## 响应规范
- 统一结构：`{ "code": 0, "message": "OK", "data": {...}, "request_id": "..." }`
- 错误码：
  - `0` 成功
  - `1001` 参数错误
  - `1002` 未认证 / 令牌无效
  - `1003` 权限不足
  - `2001` 资源不存在
  - `3001` 任务不可执行
  - `5000` 服务端错误

## HTTP 接口
- `POST /api/auth/register`
  - 用途：用户注册（仅开发或管理员允许）。
  - 请求：`{ "email": "string", "password": "string", "name": "string" }`
  - 响应：`{ "user_id": "string" }`
- `POST /api/auth/login`
  - 用途：用户名/邮箱登录，发放 `JWT`。
  - 请求：`{ "email": "string", "password": "string" }`
  - 响应：`{ "access_token": "string", "expires_in": 3600, "refresh_token": "string" }`
- `POST /api/auth/refresh`
  - 用途：刷新令牌。
  - 请求：`{ "refresh_token": "string" }`
  - 响应：`{ "access_token": "string", "expires_in": 3600 }`
- `POST /api/auth/logout`
  - 用途：注销（服务端黑名单 refresh / 访问令牌）。
  - 请求：`{}`（需 `Authorization: Bearer <token>`）
  - 响应：`{}`
- `GET /api/auth/me`
  - 用途：获取当前用户信息与角色。
  - 响应：`{ "id": "string", "email": "string", "name": "string", "roles": ["admin","user"] }`

- `GET /api/agents`
  - 用途：列出 Agent 及在线状态。
  - 响应：`[{ "id": "string", "name": "string", "type": "string", "online": true, "tags": ["nlp"], "capabilities": ["chat","search"], "concurrency": 4 }...]`
- `POST /api/agents`
  - 用途：创建 Agent 配置。
  - 请求：`{ "name": "string", "type": "string", "capabilities": ["string"], "concurrency": 4, "token": "string", "tags": ["string"], "meta": {}}`
  - 响应：`{ "id": "string" }`
- `GET /api/agents/:id`
  - 用途：查询单个 Agent。
  - 响应：`{ ... }`
- `PUT /api/agents/:id`
  - 用途：更新 Agent 配置。
  - 请求：`{ "name": "string?", "capabilities": ["string"]?, "concurrency": 8? }`
  - 响应：`{ "updated": true }`
- `DELETE /api/agents/:id`
  - 用途：删除 Agent。
  - 响应：`{ "deleted": true }`

- `POST /api/tasks`
  - 用途：提交任务。
  - 请求：`{ "agent_id": "string", "type": "string", "input": {}, "priority": 0 }`
  - 响应：`{ "task_id": "string", "status": "queued" }`
- `GET /api/tasks/:id`
  - 用途：查询任务详情与当前状态。
  - 响应：`{ "task_id": "string", "status": "running|queued|done|failed", "result": {}, "error": "" }`
- `GET /api/tasks`
  - 用途：任务列表与过滤。
  - 查询参数：`status`、`agent_id`、`type`、`page`、`page_size`。
  - 响应：`{ "items": [...], "total": 123 }`
- `POST /api/tasks/:id/cancel`
  - 用途：取消任务。
  - 响应：`{ "canceled": true }`

## WebSocket 接口
- 路径：`/ws`
- 鉴权：
  - 方式一：请求头 `Authorization: Bearer <access_token>`。
  - 方式二：查询串 `?token=<access_token>`（只用于非浏览器或受限环境）。
- 连接角色：
  - `role=agent`：Agent 进程接入，需提供 `agent_token` 与 `agent_id`。
  - `role=client`：前端页面接入，使用用户的 `access_token`。
- 握手参数：`role`、`agent_id`、`agent_token`、`client_id`。
- 消息包结构：`{ "id": "uuid", "type": "string", "ts": 1733000000, "payload": {} }`
- 消息类型：
  - `auth/ok`、`auth/error`
  - `heartbeat`（服务端下发心跳间隔，客户端回传）
  - `agent/status`（在线/离线、资源占用）
  - `task/submit`（客户端→服务端→Agent）
  - `task/accept`（Agent 接收）
  - `task/progress`（流式）
  - `task/result`（完成）
  - `task/error`（失败）
- 断线与重连：服务端维护连接映射（`agent_id`→`conn`）；支持指数退避重连；未完成任务自动回收为 `queued`。

## 任务调度
- 队列模型：`Redis` 列表或有序集合，支持优先级与延迟。
- 路由策略：按 `agent_id` 指定或按 `capabilities` 匹配可用 Agent；支持 `concurrency` 上限与负载均衡。
- 状态管理：`queued`→`running`→`done|failed|canceled`，状态变更事件通过 `WebSocket` 推送。
- 定时任务：使用 `gocron` 或 `Redis` 延时队列实现 `cron`，在后端创建/管理。
- 幂等：`task_id` 唯一；重复提交策略可配置（去重键或强制新建）。

## Agent 配置管理
- 字段建议：`id`、`name`、`type`、`capabilities[]`、`concurrency`、`tags[]`、`meta{}`、`token`、`created_at`、`updated_at`。
- 安全：`token` 仅可写不可读（写入时加密存储，读取不返回原文）。
- 在线状态：由 `ws` 心跳汇报与服务端检测组合判定；缓存于 `Redis`。

## 权限模型
- 角色：`admin`、`user`、`agent`（WebSocket 侧）。
- 规则：
  - `admin`：管理用户与 Agent、所有任务。
  - `user`：只能操作自身任务与查看公开 Agent。
  - `agent`：只允许与调度器交互的消息通道。

## 安全与合规
- 密码：`bcrypt` 哈希，禁止明文存储。
- 令牌：`JWT` 使用强随机 `JWT_SECRET`，设置合理 `exp` 与 `aud`。
- 传输：启用 `TLS` 与 `HSTS`（生产）；`CORS` 白名单控制。
- 输入校验：所有 `POST/PUT` 进行严格校验与长度限制。
- 速率限制：对登录、任务提交等敏感接口限速。
- 审计：记录关键操作与任务状态流转日志。

## 本地开发与运行
- 初始化：
  - `go mod init example.com/agent-server`
  - `go get github.com/cloudwego/hertz`
  - `go get gorm.io/gorm gorm.io/driver/sqlite`
  - `go get github.com/golang-jwt/jwt/v5`
  - `go get nhooyr.io/websocket`
  - 可选：`go get github.com/go-redis/redis/v9`
- 启动：
  - 设置环境变量，例如：
    - `set APP_ENV=dev`
    - `set HTTP_ADDR=:8080`
    - `set JWT_SECRET=YOUR_SECRET`
    - `set DB_DSN=sqlite://./data/app.db`
  - 运行：`go run ./cmd/server`
- 迁移：
  - 使用 `migrate` 或初始化 `gorm` 自动建表；迁移文件位于 `configs/migrations`。

## 示例请求与响应
- 登录请求：`POST /api/auth/login`
  - 示例：`{"email":"a@b.com","password":"p@ss"}`
  - 响应：`{"access_token":"...","expires_in":3600,"refresh_token":"..."}`
- WebSocket 握手：`GET /ws?role=client&token=<ACCESS_TOKEN>`
  - 首帧：`{"id":"uuid","type":"heartbeat","ts":1733000000,"payload":{"interval":30}}`

## 里程碑与实现顺序（后端）
- 第 1 步：搭建 Hertz 与基础中间件（日志、恢复、CORS、请求 ID）。
- 第 2 步：用户模型与 `JWT` 登录、刷新、`/me`。
- 第 3 步：WebSocket 服务端实现、认证、心跳与会话管理。
- 第 4 步：任务模型、队列与调度、状态事件。
- 第 5 步：Agent 配置 CRUD 与在线状态管理。
- 第 6 步：限流、审计与基本监控指标。

## 审核要点
- 接口是否覆盖需求并保持最小可用。
- 鉴权与安全边界是否清晰与可验证。
- 调度策略与失败恢复是否明确。
- 配置管理与敏感信息存储是否安全。
- 文档与环境是否足够支撑开发启动。

