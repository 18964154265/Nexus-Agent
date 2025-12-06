package store

type Store interface {
	RandToken() string

	CreateUser(u *User) (*User, bool)
	FindUserByEmail(email string) *User
	FindUserByID(id string) *User

	SaveRefresh(rt *RefreshToken)
	GetRefresh(token string) *RefreshToken
	RevokeRefresh(token string)

	CreateAgent(a *Agent) *Agent
	UpdateAgent(id string, f func(*Agent)) bool
	DeleteAgent(id string) bool
	GetAgent(id string) *Agent
	ListAgents() []*Agent

	CreateAPIKey(k *APIKey) *APIKey
	ListAPIKeysByUser(userID string) []*APIKey
	DeleteAPIKey(id string) bool
	GetAPIKey(id string) *APIKey

	CreateIntegration(in *UserIntegration) *UserIntegration
	ListIntegrationsByUser(userID string) []*UserIntegration

	CreateKnowledgeBase(kb *KnowledgeBase) *KnowledgeBase
	GetKnowledgeBase(id string) *KnowledgeBase
	ListKnowledgeBasesByUser(userID string) []*KnowledgeBase

	CreateMCPServer(s *MCPServer) *MCPServer
	ListMCPServersByAgent(agentID string) []*MCPServer

	CreateMCPTool(t *MCPTool) *MCPTool
	ListMCPToolsByServer(serverID string) []*MCPTool

	CreateChatSession(s *ChatSession) *ChatSession
	ListChatSessionsByUser(userID string) []*ChatSession

	CreateRun(r *Run) (*Run, error)
	FinishRun(id string, output map[string]interface{}, status string) bool
	ListRunsBySession(sessionID string) []*Run

	CreateRunStep(rs *RunStep) *RunStep
	FinishRunStep(id string, out map[string]interface{}, status string, latency int, errMsg string) bool
	ListRunStepsByRun(runID string) []*RunStep

	CreateChatMessage(cm *ChatMessage) *ChatMessage
	ListChatMessagesBySession(sessionID string) []*ChatMessage
	ListChatMessagesByRun(runID string) []*ChatMessage
}

var current Store

func Set(s Store) { current = s }

// ... 之前的 import 和 struct 定义 ...

// NewMemoryStore 构造函数：初始化所有 map，防止 nil panic
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		users:        make(map[string]*User),
		usersByE:     make(map[string]*User),
		refresh:      make(map[string]*RefreshToken),
		apiKeys:      make(map[string]*APIKey),
		integrations: make(map[string]*UserIntegration),
		kbs:          make(map[string]*KnowledgeBase),
		agents:       make(map[string]*Agent),
		mcpServers:   make(map[string]*MCPServer),
		mcpTools:     make(map[string]*MCPTool),
		sessions:     make(map[string]*ChatSession),
		runs:         make(map[string]*Run),
		runSteps:     make(map[string]*RunStep),
		messages:     make(map[string]*ChatMessage),
	}
}

func GetStore() Store { return current }
