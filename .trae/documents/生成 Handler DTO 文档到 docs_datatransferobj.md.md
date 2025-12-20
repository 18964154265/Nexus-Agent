## 目标

* 汇总当前所有 handler 下的 DTO 结构体，以表格形式呈现“类型名、字段及类型”。

* 结果输出到 `docs/datatransferobj.md`，并按文件分组（同一文件内的结构体相邻排列）。

## 数据来源（已研读）

* `internal/handler/agents.go`

  * `AgentReq`

  * `AgentResp`

* `internal/handler/auth.go`

  * `RegisterReq`

  * `LoginReq`

  * `RefreshReq`

  * `AuthResponse`

* `internal/handler/apikeys.go`

  * `CreateAPIKeyReq`

  * `CreateAPIKeyResp`

  * `APIKeyResp`

* `internal/handler/mcp.go`

  * `RegisterMCPServerReq`

  * `MCPServerResp`

  * `MCPToolResp`

* `internal/handler/sessions.go`

  * `CreateSessionReq`

  * `ChatSessionResp`

## 输出格式

* 使用 Markdown 表格；每个结构体一个小节，包含：

  * 结构体名（小节标题）

  * 表格列：`字段` | `类型` | `备注（如必填/校验/说明）`

* 同一 handler 文件内的结构体按代码出现顺序排列，文件之间按上面“数据来源”顺序组织。

## 字段采集规则

* 类型来自 Go 源码（例如：`string`、`float64`、`[]string`、`map[string]interface{}`）。

* 对于响应类型中嵌入的实体（例如 `*store.Agent`、`*store.MCPServer`、`*store.MCPTool`、`*store.ChatSession`），在“备注”中标注“嵌入原始实体，不在此文档展开字段”。

* 若结构体字段带有校验标签（如 `vd:"required"`、范围限制），在“备注”列记录。

* 若字段通过 `json:"-"` 屏蔽，备注标注“隐藏，不返回”。

## 文档结构示例

* 顶部添加目录索引（文件分组），便于跳转。

* 每个文件作为一级小节（如 `### agents.go`），每个 DTO 作为二级小节（如 `#### AgentReq`）。

* 表格示例：

  * `字段 | 类型 | 备注`

  * `name | string | required`

  * `temperature | float64 | 范围: >=0,<=2`

  * `extra_config | map[string]interface{} | 可选`

## 具体编写要点

* `AgentResp`、`MCPServerResp`、`MCPToolResp`、`ChatSessionResp` 中的时间覆盖字段（如 `CreatedAt`/`UpdatedAt`）以 `string` 标注并备注“格式化 RFC3339/或具体格式”。

* `AuthResponse` 的 `User` 字段标注为“嵌入 `*store.User`”。

* `RegisterMCPServerReq` 的 `transport_type` 备注允许值：`stdio|sse`（并按当前策略仅允许 `sse`）。

* `CreateAPIKeyResp.key` 备注“仅创建时返回明文”。

## 交付

* 生成 `docs/datatransferobj.md`，包含上述分组与表格内容。

* 完成后我会回传完成说明，供你审阅并提出修订意见（如字段增删、命名统一、校验规则补充）。

