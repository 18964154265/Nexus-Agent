// Common Response Structure
export interface ApiResponse<T = any> {
  code: number;
  message: string;
  data: T;
}

// User & Auth
export interface User {
  id: string;
  email: string;
  name: string;
  roles: string[];
  created_at: string;
  
}

export interface LoginResp {
  access_token: string;
  refresh_token: string;
  user: User;
}

// API Key
export interface APIKey {
  id: string;
  name: string;
  prefix: string;
  created_at: string;
  last_used_at?: string;
}

// Agent
export interface Agent {
  id: string;
  name: string;
  description: string;
  model_name: string;
  system_prompt?: string;
  temperature: number;
  knowledge_base_ids?: string[];
  capabilities?: string[];
  extra_config?: Record<string, any>;
  tags?: string[];
  type: 'user'|'system';
  status: string;
  created_at: string;
}

// MCP
export interface MCPServer {
  id: string;
  name: string;
  status: string;
  transport_type: string;
  is_global: boolean;
  connection_config: Record<string, any>; 
  tool_count: number;
  created_at: string;
}

export interface MCPTool {
  id: string;
  server_id: string;
  name: string;
  description: string;
  input_schema: Record<string, any>;
}

// Knowledge Base
export interface KnowledgeBase {
  id: string;
  name: string;
  description: string;
  is_public: boolean;
  document_count: number;
  created_at: string;
  updated_at: string;
}

// Chat Session
export interface ChatSession {
  id: string;
  title: string;
  agent_id: string;
  created_at: string;
  updated_at: string;
}

export interface ChatMessage {
  id: string;
  role: 'user' | 'assistant' | 'tool' | 'system';
  
  // Content 在数据库是 JSONB，我们在 Go 里存的是 map
  content: {
    text?: string;       
    type?: string;       
    tool_calls?: Array<{
      id: string;
      type: 'function';
      function: {
        name: string;
        arguments: string; // JSON string
      }
    }>;
    
    [key: string]: any;
  };
  
  run_id?: string; // 关联的 Run ID（仅当 role === 'assistant' 时可能有值）
  tool_call_id?: string; // 仅当 role === 'tool' 时有值
  created_at: string;
}

export interface Run {
  id: string;
  session_id: string;
  agent_id: string;
  trace_id: string;
  parent_run_id?: string; // 如果是子任务
  
  status: 'queued' | 'running' | 'succeeded' | 'failed' | 'cancelled';
  
  // 列表页通常不返回巨大的 payload，只返回摘要
  usage_metadata?: {
    total_tokens: number;
    input_tokens: number;
    output_tokens: number;
  };
  
  created_at: string;
  finished_at?: string;
}

// Run Step (Trace)
export interface RunStep {
  id: string;
  run_id: string;
  step_type: string;
  name: string;
  status: string;
  input_payload: any;
  output_payload: any;
  latency_ms: number;
  error_message?: string;
  started_at: string;
  finished_at: string;
}

