# Handler DTO 清单

## agents.go

### AgentReq

| 字段 | 类型 | 备注 |
| - | - | - |
| name | string | required |
| description | string |  |
| model_name | string | 默认 gpt-4o |
| system_prompt | string | required |
| temperature | float64 | 范围: >=0,<=2 |
| knowledge_base_ids | []string |  |
| tags | []string |  |
| extra_config | map[string]interface{} |  |
| capabilities | []string |  |

### AgentResp

| 字段 | 类型 | 备注 |
| - | - | - |
| (embedded) | *store.Agent | 嵌入原始实体，不在此展开 |
| created_at | string | 时间格式化输出 |
| updated_at | string | 时间格式化输出 |
| system_prompt | string | 隐藏，不返回（json:"-") |
| extra_config | any | 隐藏，不返回（json:"-") |
| owner_user_id | string | 隐藏，不返回（json:"-") |

## auth.go

### RegisterReq

| 字段 | 类型 | 备注 |
| - | - | - |
| email | string |  |
| password | string |  |
| name | string |  |

### LoginReq

| 字段 | 类型 | 备注 |
| - | - | - |
| email | string |  |
| password | string |  |

### RefreshReq

| 字段 | 类型 | 备注 |
| - | - | - |
| refresh_token | string |  |

### AuthResponse

| 字段 | 类型 | 备注 |
| - | - | - |
| token | string | 访问令牌 |
| refresh_token | string | 刷新令牌 |
| user | *store.User | 嵌入原始实体，不在此展开 |

## apikeys.go

### CreateAPIKeyReq

| 字段 | 类型 | 备注 |
| - | - | - |
| name | string | required |

### CreateAPIKeyResp

| 字段 | 类型 | 备注 |
| - | - | - |
| id | string |  |
| name | string |  |
| prefix | string | 展示前缀 |
| key | string | 仅创建时返回明文 |
| created_at | string | 时间格式化输出 |

### APIKeyResp

| 字段 | 类型 | 备注 |
| - | - | - |
| id | string |  |
| name | string |  |
| prefix | string |  |
| last_used_at | string | 时间格式化输出，可能为空 |
| created_at | string | 时间格式化输出 |

## mcp.go

### RegisterMCPServerReq

| 字段 | 类型 | 备注 |
| - | - | - |
| name | string | required |
| transport_type | string | 允许值: stdio|sse；当前策略仅允许 sse |
| agent_id | string | 可选，绑定目标 Agent |
| connection_config | map[string]interface{} | required |

### MCPServerResp

| 字段 | 类型 | 备注 |
| - | - | - |
| (embedded) | *store.MCPServer | 嵌入原始实体，不在此展开 |
| created_at | string | 时间格式化输出 |
| updated_at | string | 时间格式化输出 |
| tool_count | int | 工具数量 |

### MCPToolResp

| 字段 | 类型 | 备注 |
| - | - | - |
| (embedded) | *store.MCPTool | 嵌入原始实体，不在此展开 |
| created_at | string | 时间格式化输出 |

## sessions.go

### CreateSessionReq

| 字段 | 类型 | 备注 |
| - | - | - |
| agent_id | string | required；外键校验必须存在 |
| title | string | 可选；为空时按 Agent 名构造默认标题 |

### ChatSessionResp

| 字段 | 类型 | 备注 |
| - | - | - |
| (embedded) | *store.ChatSession | 嵌入原始实体，不在此展开 |
| created_at | string | 时间格式化输出 |
| updated_at | string | 时间格式化输出 |
| user_id | string | 隐藏，不返回（json:"-") |
