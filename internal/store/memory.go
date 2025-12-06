package store

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID       string
	Email    string
	Name     string
	Password string
	Roles    []string
	Created  time.Time
}

type RefreshToken struct {
	Token   string
	UserID  string
	Expire  time.Time
	Revoked bool
}

type APIKey struct {
	ID         string
	UserID     string
	Name       string
	Prefix     string
	KeyHash    string
	LastUsedAt time.Time
	ExpiresAt  time.Time
	CreatedAt  time.Time
}

type UserIntegration struct {
	ID                   string
	UserID               string
	Provider             string
	EncryptedCredentials string
	DisplayLabel         string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type KnowledgeBase struct {
	ID          string
	UserID      string
	Name        string
	Description string
	IsPublic    bool
	MetaInfo    map[string]interface{}
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Agent struct {
	ID               string
	OwnerUserID      string
	Name             string
	Description      string
	ModelName        string
	SystemPrompt     string
	Temperature      float64
	KnowledgeBaseIDs []string
	Status           string
	ExtraConfig      map[string]interface{}
	Type             string
	Capabilities     []string
	Concurrency      int
	Tags             []string
	Meta             map[string]interface{}
	Token            string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type MCPServer struct {
	ID               string
	AgentID          string
	Name             string
	TransportType    string
	ConnectionConfig map[string]interface{}
	IsGlobal         bool
	Status           string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type MCPTool struct {
	ID          string
	ServerID    string
	Name        string
	Description string
	InputSchema map[string]interface{}
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ChatSession struct {
	ID        string
	UserID    string
	AgentID   string
	Title     string
	Summary   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Run struct {
	ID            string
	SessionID     string
	UserID        string
	AgentID       string
	ParentRunID   string
	TraceID       string
	Status        string
	InputPayload  map[string]interface{}
	OutputPayload map[string]interface{}
	UsageMetadata map[string]interface{}
	StartedAt     time.Time
	FinishedAt    time.Time
}

type RunStep struct {
	ID            string
	RunID         string
	StepType      string
	Name          string
	InputPayload  map[string]interface{}
	OutputPayload map[string]interface{}
	Status        string
	ErrorMessage  string
	LatencyMS     int
	StartedAt     time.Time
	FinishedAt    time.Time
}

type ChatMessage struct {
	ID         string
	SessionID  string
	RunID      string
	Role       string
	Content    map[string]interface{}
	ToolCallID string
	TokenCount int
	IsHidden   bool
	CreatedAt  time.Time
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
	u.Created = time.Now()
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
