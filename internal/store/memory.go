package store

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// JSONMap 定义通用的 JSONB 类型，并实现 Scanner/Valuer 接口 (如果 GORM 默认不支持 map[string]interface{})
// 不过 GORM 对 map[string]interface{} + gorm:"type:jsonb" 支持良好，
// 这里为了类型安全和统一，可以定义一个别名，或者直接用 map[string]interface{}
type JSONMap map[string]interface{}

// GORM 需要实现 Valuer 和 Scanner 接口才能正确处理自定义 map 类型
// 简单起见，我们先用 map[string]interface{}，但在 struct tag 里必须指明 type:jsonb
// 注意：Postgres driver (pgx) 对 map 的支持可能需要显式声明 Valuer/Scanner
// 为了修复 "unsupported data type: &map[]" 错误，我们需要让它符合 sql.Scanner 和 driver.Valuer
// 或者直接使用 datatypes.JSON (来自 gorm.io/datatypes)
// 但为了少引入依赖，我们定义一个包装类型

func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	return json.Marshal(m)
}

func (m *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*m = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSONB value: %v", value)
	}
	result := make(JSONMap)
	err := json.Unmarshal(bytes, &result)
	*m = result
	return err
}

type User struct {
	ID        string         `json:"id"`
	Email     string         `json:"email"`
	Name      string         `json:"name"`
	Password  string         `json:"password"`
	Roles     pq.StringArray `json:"roles" gorm:"type:text[]"`
	CreatedAt time.Time      `json:"created_at"`
}

type RefreshToken struct {
	Token   string    `json:"token"`
	UserID  string    `json:"user_id"`
	Expire  time.Time `json:"expire"`
	Revoked bool      `json:"revoked"`
}

type APIKey struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	Name       string    `json:"name"`
	Prefix     string    `json:"prefix"`
	KeyHash    string    `json:"key_hash"`
	LastUsedAt time.Time `json:"last_used_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
}

type UserIntegration struct {
	ID                   string    `json:"id"`
	UserID               string    `json:"user_id"`
	Provider             string    `json:"provider"`
	EncryptedCredentials string    `json:"encrypted_credentials"`
	DisplayLabel         string    `json:"display_label"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type KnowledgeBase struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsPublic    bool      `json:"is_public"`
	MetaInfo    JSONMap   `json:"meta_info" gorm:"type:jsonb"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Agent struct {
	ID               string         `json:"id"`
	OwnerUserID      string         `json:"owner_user_id"`
	Name             string         `json:"name"`
	Description      string         `json:"description"`
	ModelName        string         `json:"model_name"`
	SystemPrompt     string         `json:"system_prompt"`
	Temperature      float64        `json:"temperature"`
	KnowledgeBaseIDs pq.StringArray `json:"knowledge_base_ids" gorm:"type:text[]"`
	Status           string         `json:"status"`
	ExtraConfig      JSONMap        `json:"extra_config" gorm:"type:jsonb"`
	Type             string         `json:"type"`
	Capabilities     pq.StringArray `json:"capabilities" gorm:"type:text[]"`
	Concurrency      int            `json:"concurrency"`
	Tags             pq.StringArray `json:"tags" gorm:"type:text[]"`
	Meta             JSONMap        `json:"meta" gorm:"type:jsonb"`
	Token            string         `json:"token"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

type MCPServer struct {
	ID               string    `json:"id"`
	AgentID          string    `json:"agent_id"`
	Name             string    `json:"name"`
	TransportType    string    `json:"transport_type"`
	ConnectionConfig JSONMap   `json:"connection_config" gorm:"type:jsonb"`
	IsGlobal         bool      `json:"is_global"` //这是平台预先定义好的几个常用server
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type MCPTool struct {
	ID          string    `json:"id"`
	ServerID    string    `json:"server_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	InputSchema JSONMap   `json:"input_schema" gorm:"type:jsonb"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ChatSession struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	AgentID   string    `json:"agent_id"`
	Title     string    `json:"title"`
	Summary   string    `json:"summary"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Run struct {
	ID            string    `json:"id"`
	SessionID     string    `json:"session_id"`
	UserID        string    `json:"user_id"`
	AgentID       string    `json:"agent_id"`
	ParentRunID   string    `json:"parent_run_id"`
	TraceID       string    `json:"trace_id"`
	Status        string    `json:"status"`
	InputPayload  JSONMap   `json:"input_payload" gorm:"type:jsonb"`
	OutputPayload JSONMap   `json:"output_payload" gorm:"type:jsonb"`
	UsageMetadata JSONMap   `json:"usage_metadata" gorm:"type:jsonb"`
	StartedAt     time.Time `json:"started_at"`
	FinishedAt    time.Time `json:"finished_at"`
}

type RunStep struct {
	ID            string    `json:"id"`
	RunID         string    `json:"run_id"`
	StepType      string    `json:"step_type"`
	Name          string    `json:"name"`
	InputPayload  JSONMap   `json:"input_payload" gorm:"type:jsonb"`
	OutputPayload JSONMap   `json:"output_payload" gorm:"type:jsonb"`
	Status        string    `json:"status"`
	ErrorMessage  string    `json:"error_message"`
	LatencyMS     int       `json:"latency_ms"`
	StartedAt     time.Time `json:"started_at"`
	FinishedAt    time.Time `json:"finished_at"`
}

type ChatMessage struct {
	ID         string    `json:"id"`
	SessionID  string    `json:"session_id"`
	RunID      string    `json:"run_id"`
	Role       string    `json:"role"`
	Content    JSONMap   `json:"content" gorm:"type:jsonb"`
	ToolCallID string    `json:"tool_call_id"`
	TokenCount int       `json:"token_count"`
	IsHidden   bool      `json:"is_hidden"`
	CreatedAt  time.Time `json:"created_at"`
}

type MemoryStore struct {
	mu           sync.RWMutex
	users        map[string]*User
	usersByE     map[string]*User
	refresh      map[string]*RefreshToken
	agents       map[string]*Agent
	apiKeys      map[string]*APIKey
	integrations map[string]*UserIntegration
	kbs          map[string]*KnowledgeBase
	mcpServers   map[string]*MCPServer
	mcpTools     map[string]*MCPTool
	sessions     map[string]*ChatSession
	runs         map[string]*Run
	runSteps     map[string]*RunStep
	messages     map[string]*ChatMessage
}

func init() {
	Set(&MemoryStore{users: map[string]*User{}, usersByE: map[string]*User{}, refresh: map[string]*RefreshToken{}, agents: map[string]*Agent{}, apiKeys: map[string]*APIKey{}, integrations: map[string]*UserIntegration{}, kbs: map[string]*KnowledgeBase{}, mcpServers: map[string]*MCPServer{}, mcpTools: map[string]*MCPTool{}, sessions: map[string]*ChatSession{}, runs: map[string]*Run{}, runSteps: map[string]*RunStep{}, messages: map[string]*ChatMessage{}})
}

func randID() string {
	return uuid.New().String()
}

func (m *MemoryStore) RandToken() string { return randID() }

func (m *MemoryStore) CreateUser(u *User) (*User, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.usersByE[u.Email]; ok {
		return nil, false
	}
	u.ID = randID()
	u.CreatedAt = time.Now()
	m.users[u.ID] = u
	m.usersByE[u.Email] = u
	return u, true
}

func (m *MemoryStore) FindUserByEmail(email string) *User {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.usersByE[email]
}

func (m *MemoryStore) FindUserByID(id string) *User {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.users[id]
}

func (m *MemoryStore) SaveRefresh(rt *RefreshToken) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.refresh[rt.Token] = rt
}

func (m *MemoryStore) GetRefresh(token string) *RefreshToken {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.refresh[token]
}

func (m *MemoryStore) RevokeRefresh(token string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r := m.refresh[token]; r != nil {
		r.Revoked = true
	}
}

func (m *MemoryStore) CreateAgent(a *Agent) *Agent {
	m.mu.Lock()
	defer m.mu.Unlock()
	a.ID = randID()
	now := time.Now()
	a.CreatedAt = now
	a.UpdatedAt = now
	if a.Meta == nil {
		a.Meta = map[string]interface{}{}
	}
	if a.ExtraConfig == nil {
		a.ExtraConfig = map[string]interface{}{}
	}
	m.agents[a.ID] = a
	return a
}

func (m *MemoryStore) UpdateAgent(id string, f func(*Agent)) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if a, ok := m.agents[id]; ok {
		f(a)
		a.UpdatedAt = time.Now()
		return true
	}
	return false
}

func (m *MemoryStore) DeleteAgent(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.agents[id]; ok {
		delete(m.agents, id)
		return true
	}
	return false
}

func (m *MemoryStore) GetAgent(id string) *Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.agents[id]
}

func (m *MemoryStore) ListAgents() []*Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*Agent, 0, len(m.agents))
	for _, a := range m.agents {
		res = append(res, a)
	}
	return res
}

func (m *MemoryStore) CreateAPIKey(k *APIKey) *APIKey {
	m.mu.Lock()
	defer m.mu.Unlock()
	k.ID = randID()
	k.CreatedAt = time.Now()
	m.apiKeys[k.ID] = k
	return k
}

func (m *MemoryStore) ListAPIKeysByUser(userID string) []*APIKey {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := []*APIKey{}
	for _, k := range m.apiKeys {
		if k.UserID == userID {
			res = append(res, k)
		}
	}
	return res
}

// internal/store/store.go

func (m *MemoryStore) GetAPIKey(id string) *APIKey {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.apiKeys[id]
}

func (m *MemoryStore) DeleteAPIKey(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.apiKeys[id]; ok {
		delete(m.apiKeys, id)
		return true
	}
	return false
}

func (m *MemoryStore) CreateIntegration(in *UserIntegration) *UserIntegration {
	m.mu.Lock()
	defer m.mu.Unlock()
	in.ID = randID()
	now := time.Now()
	in.CreatedAt = now
	in.UpdatedAt = now
	m.integrations[in.ID] = in
	return in
}

func (m *MemoryStore) ListIntegrationsByUser(userID string) []*UserIntegration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := []*UserIntegration{}
	for _, in := range m.integrations {
		if in.UserID == userID {
			res = append(res, in)
		}
	}
	return res
}

func (m *MemoryStore) CreateKnowledgeBase(kb *KnowledgeBase) *KnowledgeBase {
	m.mu.Lock()
	defer m.mu.Unlock()
	kb.ID = randID()
	now := time.Now()
	kb.CreatedAt = now
	kb.UpdatedAt = now
	if kb.MetaInfo == nil {
		kb.MetaInfo = map[string]interface{}{}
	}
	m.kbs[kb.ID] = kb
	return kb
}

func (m *MemoryStore) GetKnowledgeBase(id string) *KnowledgeBase {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.kbs[id]
}

func (m *MemoryStore) ListKnowledgeBasesByUser(userID string) []*KnowledgeBase {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := []*KnowledgeBase{}
	for _, k := range m.kbs {
		if k.UserID == userID {
			res = append(res, k)
		}
	}
	return res
}

// UpdateKnowledgeBase 更新知识库元数据
func (m *MemoryStore) UpdateKnowledgeBase(id string, f func(*KnowledgeBase)) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if kb, ok := m.kbs[id]; ok {
		f(kb)
		kb.UpdatedAt = time.Now()
		return true
	}
	return false
}

func (m *MemoryStore) CreateMCPServer(s *MCPServer) *MCPServer {
	m.mu.Lock()
	defer m.mu.Unlock()
	s.ID = randID()
	now := time.Now()
	s.CreatedAt = now
	s.UpdatedAt = now
	if s.ConnectionConfig == nil {
		s.ConnectionConfig = map[string]interface{}{}
	}
	m.mcpServers[s.ID] = s
	return s
}

func (m *MemoryStore) ListMCPServersByAgent(agentID string) []*MCPServer {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := []*MCPServer{}
	for _, s := range m.mcpServers {
		if s.AgentID == agentID {
			res = append(res, s)
		}
	}
	return res
}

func (m *MemoryStore) GetMCPServer(id string) *MCPServer {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.mcpServers[id]
}

// ListAllMCPServers (可选，用于 List 时不传参的情况)
func (m *MemoryStore) ListAllMCPServers() []*MCPServer {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := []*MCPServer{}
	for _, s := range m.mcpServers {
		res = append(res, s)
	}
	return res
}

func (m *MemoryStore) ListGlobalMCPServers() []*MCPServer {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := []*MCPServer{}
	for _, s := range m.mcpServers {
		if s.IsGlobal {
			res = append(res, s)
		}
	}
	return res
}

func (m *MemoryStore) FindMCPServerByName(agentID string, name string) *MCPServer {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 1. 优先找私有的
	for _, s := range m.mcpServers {
		if s.AgentID == agentID && s.Name == name {
			return s
		}
	}

	// 2. 找不到，找全局的
	for _, s := range m.mcpServers {
		if s.IsGlobal && s.Name == name {
			return s
		}
	}

	return nil
}

func (m *MemoryStore) CreateMCPTool(t *MCPTool) *MCPTool {
	m.mu.Lock()
	defer m.mu.Unlock()
	t.ID = randID()
	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now
	if t.InputSchema == nil {
		t.InputSchema = map[string]interface{}{}
	}
	m.mcpTools[t.ID] = t
	return t
}

func (m *MemoryStore) ListMCPToolsByServer(serverID string) []*MCPTool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := []*MCPTool{}
	for _, t := range m.mcpTools {
		if t.ServerID == serverID {
			res = append(res, t)
		}
	}
	return res
}

func (m *MemoryStore) ListMCPToolsByAgent(agentID string) []*MCPTool {
	//必须是显式绑定的server，globalserver不能直接加上来
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := []*MCPTool{}
	for _, s := range m.ListMCPServersByAgent(agentID) {
		res = append(res, m.ListMCPToolsByServer(s.ID)...)
	}
	return res
}

func (m *MemoryStore) ListGlobalMCPTools() []*MCPTool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	globalMCPServers := m.ListGlobalMCPServers()
	res := []*MCPTool{}
	for _, s := range globalMCPServers {
		res = append(res, m.ListMCPToolsByServer(s.ID)...)
	}
	return res
}

// internal/store/store.go

// FindMCPToolByName 根据工具名查找工具定义 (MVP 简化版，假设工具名全局唯一)
func (m *MemoryStore) FindMCPToolByName(name string) *MCPTool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, t := range m.mcpTools {
		if t.Name == name {
			return t
		}
	}
	return nil
}

func (m *MemoryStore) CreateChatSession(s *ChatSession) *ChatSession {
	m.mu.Lock()
	defer m.mu.Unlock()
	s.ID = randID()
	now := time.Now()
	s.CreatedAt = now
	s.UpdatedAt = now
	m.sessions[s.ID] = s
	return s
}

func (m *MemoryStore) ListChatSessionsByUser(userID string) []*ChatSession {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := []*ChatSession{}
	for _, s := range m.sessions {
		if s.UserID == userID {
			res = append(res, s)
		}
	}
	return res
}

func (m *MemoryStore) GetChatSession(id string) *ChatSession {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[id]
}

func (m *MemoryStore) DeleteChatSession(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.sessions[id]; ok {
		delete(m.sessions, id)
		return true
	}
	return false
}

func (m *MemoryStore) CreateRun(r *Run) (*Run, error) {
	//添加外键检查

	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.agents[r.AgentID]; !ok {
		return nil, fmt.Errorf("agent not found: %s", r.AgentID)
	}
	r.ID = randID()
	r.StartedAt = time.Now()
	if r.InputPayload == nil {
		r.InputPayload = make(map[string]interface{})
	}
	if r.OutputPayload == nil {
		r.OutputPayload = make(map[string]interface{})
	}
	m.runs[r.ID] = r
	return r, nil
}

func (m *MemoryStore) FinishRun(id string, output map[string]interface{}, status string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r, ok := m.runs[id]; ok {
		r.OutputPayload = output
		r.Status = status
		r.FinishedAt = time.Now()
		return true
	}
	return false
}

func (m *MemoryStore) ListRunsBySession(sessionID string) []*Run {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := []*Run{}
	for _, r := range m.runs {
		if r.SessionID == sessionID {
			res = append(res, r)
		}
	}
	return res
}

func (m *MemoryStore) ListRunsByUser(userID string) []*Run {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := []*Run{}
	for _, r := range m.runs {
		if r.UserID == userID {
			res = append(res, r)
		}
	}
	// 按开始时间倒序排列
	sort.Slice(res, func(i, j int) bool {
		return res[i].StartedAt.After(res[j].StartedAt)
	})
	return res
}

func (m *MemoryStore) GetRun(runID string) *Run {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if _, ok := m.runs[runID]; !ok {
		return nil
	}
	return m.runs[runID]
}

func (m *MemoryStore) GetRunStep(id string) *RunStep {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.runSteps[id]
}

func (m *MemoryStore) CreateRunStep(rs *RunStep) *RunStep {
	m.mu.Lock()
	defer m.mu.Unlock()
	rs.ID = randID()
	rs.StartedAt = time.Now()
	m.runSteps[rs.ID] = rs
	return rs
}

func (m *MemoryStore) FinishRunStep(id string, out map[string]interface{}, status string, latency int, errMsg string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.runSteps[id]; ok {
		s.OutputPayload = out
		s.Status = status
		s.LatencyMS = latency
		s.ErrorMessage = errMsg
		s.FinishedAt = time.Now()
		return true
	}
	return false
}

func (m *MemoryStore) ListRunStepsByRun(runID string) []*RunStep {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := []*RunStep{}
	for _, s := range m.runSteps {
		if s.RunID == runID {
			copyStep := *s
			res = append(res, &copyStep)
		}
	}
	// 核心修复：Trace 必须按时间排序
	sort.Slice(res, func(i, j int) bool {
		return res[i].StartedAt.Before(res[j].StartedAt)
	})
	return res
}

func (m *MemoryStore) CreateChatMessage(cm *ChatMessage) *ChatMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	cm.ID = randID()
	cm.CreatedAt = time.Now()
	if cm.Content == nil {
		cm.Content = map[string]interface{}{}
	}
	m.messages[cm.ID] = cm
	return cm
}

func (m *MemoryStore) ListChatMessagesBySession(sessionID string) []*ChatMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 预分配容量，微小性能优化
	res := make([]*ChatMessage, 0, 20)
	for _, c := range m.messages {
		if c.SessionID == sessionID {
			// 这里建议返回 Copy，防止外部修改内部状态
			copyMsg := *c
			res = append(res, &copyMsg)
		}
	}

	// 核心修复：按时间正序排列
	sort.Slice(res, func(i, j int) bool {
		return res[i].CreatedAt.Before(res[j].CreatedAt)
	})
	return res
}

func (m *MemoryStore) ListChatMessagesByRun(runID string) []*ChatMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := []*ChatMessage{}
	for _, c := range m.messages {
		if c.RunID == runID {
			res = append(res, c)
		}
	}
	return res
}
