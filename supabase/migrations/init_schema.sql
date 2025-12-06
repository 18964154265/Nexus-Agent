-- =============================================================================
-- Nexus-Agent Platform Database Schema
-- Target: PostgreSQL 14+
-- Author: Nexus Architect
-- Description: Supports Auth, Multi-Agent Orchestration, MCP, RAG, and Observability
-- =============================================================================

-- 1. Enable Extensions
-- 用于生成 UUID 主键
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto"; 
-- 用于 JSONB 高效索引
CREATE EXTENSION IF NOT EXISTS "btree_gin";

-- 2. Common Functions
-- 自动更新 updated_at 的触发器函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- =============================================================================
-- A. Identity & Access Management (IAM)
-- =============================================================================

-- 用户表
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL, -- 使用 bcrypt/argon2
    name TEXT,
    roles TEXT[] DEFAULT '{"user"}', -- RBAC: admin, developer, user
    status TEXT DEFAULT 'active' CHECK (status IN ('active', 'disabled', 'banned')),
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 开发者 API Keys (用于 CLI 或 CI/CD 集成)
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL, -- e.g. "Macbook Pro CLI"
    prefix TEXT NOT NULL, -- e.g. "sk-nx..." 用于前端展示
    key_hash TEXT NOT NULL UNIQUE, -- 存 Hash，不存明文
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 刷新令牌 (用于 JWT 续期)
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT UNIQUE NOT NULL,
    revoked BOOLEAN DEFAULT FALSE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 专门用来存用户填入的外部平台 Token (GitHub, DockerHub, AWS 等)
CREATE TABLE user_integrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    provider TEXT NOT NULL, -- 'github', 'docker_hub', 'linear', 'slack'
    
    -- 核心：这里存的是加密后的密文！不是明文，也不是 Hash！
    -- Go 后端读取时，用 AES-GCM 解密
    encrypted_credentials TEXT NOT NULL, 
    
    -- 存一下 Key 的指纹或者最后四位，方便前端展示 "已绑定 (xv...890)"
    display_label TEXT, 
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(user_id, provider)
);

-- =============================================================================
-- B. Agent & RAG Core
-- =============================================================================

-- 知识库元数据 (RAG)
-- 实际向量存储在 ChromaDB，这里管理集合的权限和描述
CREATE TABLE knowledge_bases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL, -- 对应 ChromaDB Collection Name
    description TEXT,
    is_public BOOLEAN DEFAULT FALSE,
    meta_info JSONB DEFAULT '{}', -- 存 embedding 模型版本等
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Agent 定义 (核心大脑)
CREATE TABLE agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    description TEXT,
    
    -- 模型配置
    model_name TEXT NOT NULL DEFAULT 'gpt-4o',
    system_prompt TEXT,
    temperature FLOAT DEFAULT 0.7,
    
    -- 关联的知识库 IDs (Array of UUIDs)
    knowledge_base_ids UUID[], 
    
    -- 状态与元数据
    status TEXT DEFAULT 'active' CHECK (status IN ('active', 'draft', 'archived')),
    extra_config JSONB DEFAULT '{}', -- 预留扩展配置
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =============================================================================
-- C. MCP Ecosystem (Model Context Protocol)
-- =============================================================================

-- MCP Servers (工具箱)
CREATE TABLE mcp_servers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID REFERENCES agents(id) ON DELETE CASCADE, -- 私有 Server
    name TEXT NOT NULL,
    
    -- 传输类型：决定了 Client 如何连接
    transport_type TEXT NOT NULL CHECK (transport_type IN ('stdio', 'sse', 'http')),
    
    -- 连接配置
    connection_config JSONB NOT NULL, 
    
    is_global BOOLEAN DEFAULT FALSE, -- 是否为平台公共服务
    status TEXT DEFAULT 'active' CHECK (status IN ('active', 'maintenance', 'disabled')),
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- MCP Tools (工具缓存)
-- 用于在不连接 Server 的情况下快速展示可用工具
CREATE TABLE mcp_tools (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES mcp_servers(id) ON DELETE CASCADE,
    name TEXT NOT NULL, -- e.g. "git_commit"
    description TEXT,
    input_schema JSONB NOT NULL, -- JSON Schema
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(server_id, name)
);

-- =============================================================================
-- D. Conversation & Observability (Tables reordered for FK dependencies)
-- =============================================================================

-- 1. 会话窗口 (最顶层容器)
CREATE TABLE chat_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    
    title TEXT,
    summary TEXT, -- 长对话压缩后的摘要 (Memory)
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Runs (必须在 chat_messages 之前创建，因为 message 依赖 run)
CREATE TABLE runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID REFERENCES chat_sessions(id) ON DELETE SET NULL,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    agent_id UUID REFERENCES agents(id),
    
    -- 核心：树状结构与链路追踪
    parent_run_id UUID REFERENCES runs(id) ON DELETE SET NULL, -- 创建这个agent的父节点
    trace_id UUID NOT NULL, -- 全链路共享ID
    
    status TEXT NOT NULL CHECK (status IN ('queued', 'running', 'succeeded', 'failed', 'cancelled')),
    
    -- 输入输出与成本
    input_payload JSONB,  -- 父agent传进来的参数
    output_payload JSONB, -- 子agent返回给父agent的结果
    usage_metadata JSONB DEFAULT '{}', -- 【新增建议】存Token消耗 {"total": 500}
    
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ
);

-- 3. Run Steps (详细的执行链路)
CREATE TABLE run_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    
    step_type TEXT NOT NULL CHECK (step_type IN ('thought', 'tool_start', 'tool_end', 'llm_process')),
    name TEXT, 
    
    -- 输入输出快照
    input_payload JSONB,
    output_payload JSONB,
    
    status TEXT DEFAULT 'completed',
    error_message TEXT,
    latency_ms INT, 
    
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ
);

-- 4. 消息记录 (最后创建，因为依赖 session 和 run)
CREATE TABLE chat_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    run_id UUID REFERENCES runs(id) ON DELETE SET NULL,
    
    role TEXT NOT NULL CHECK (role IN ('user', 'assistant', 'system', 'tool')),
    content JSONB NOT NULL, -- 支持多模态结构 [{"type": "text", ...}]
    
    tool_call_id TEXT, 
    token_count INT DEFAULT 0,
    
    is_hidden BOOLEAN DEFAULT FALSE, 
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =============================================================================
-- F. Indexes & Triggers
-- =============================================================================

-- 1. 自动更新时间的 Trigger 绑定
CREATE TRIGGER update_users_modtime BEFORE UPDATE ON users FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();
CREATE TRIGGER update_agents_modtime BEFORE UPDATE ON agents FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();
CREATE TRIGGER update_mcp_servers_modtime BEFORE UPDATE ON mcp_servers FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();
CREATE TRIGGER update_chat_sessions_modtime BEFORE UPDATE ON chat_sessions FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();

-- 2. 性能索引
-- Users
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);

-- Agents & MCP
CREATE INDEX idx_agents_owner ON agents(owner_user_id);
CREATE INDEX idx_mcp_tools_server ON mcp_tools(server_id);

-- Chat History (高频查询)
CREATE INDEX idx_chat_sessions_user ON chat_sessions(user_id, updated_at DESC);
CREATE INDEX idx_chat_messages_session ON chat_messages(session_id, created_at ASC);

-- Observability (核心查询)
CREATE INDEX idx_runs_user_status ON runs(user_id, status);
CREATE INDEX idx_run_steps_run_id ON run_steps(run_id, created_at ASC);
CREATE INDEX idx_runs_parent ON runs(parent_run_id);
CREATE INDEX idx_runs_trace ON runs(trace_id);

-- JSONB GIN 索引
CREATE INDEX idx_run_steps_input ON run_steps USING gin(input_payload);
CREATE INDEX idx_run_steps_output ON run_steps USING gin(output_payload);
CREATE INDEX idx_agents_extra_config ON agents USING gin(extra_config);
CREATE INDEX idx_chat_messages_content ON chat_messages USING gin(content); -- 【新增】偶尔可能需要搜索消息内容

-- =============================================================================
-- End of Schema
-- =============================================================================