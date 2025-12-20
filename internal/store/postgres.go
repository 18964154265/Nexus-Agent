package store

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type PostgresStore struct {
	db *gorm.DB
}

func NewPostgresStore(dsn string) (*PostgresStore, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// 自动迁移模式，确保表结构存在
	// 注意：生产环境建议使用专门的迁移工具 (golang-migrate)
	// 这里为了方便开发过渡，先列出所有模型
	err = db.AutoMigrate(
		&User{},
		&RefreshToken{},
		&APIKey{},
		&UserIntegration{},
		&KnowledgeBase{},
		&Agent{},
		&MCPServer{},
		&MCPTool{},
		&ChatSession{},
		&Run{},
		&RunStep{},
		&ChatMessage{},
	)
	if err != nil {
		return nil, fmt.Errorf("auto migrate failed: %w", err)
	}

	return &PostgresStore{db: db}, nil
}

// ==========================================
// User Implementation
// ==========================================

func (s *PostgresStore) RandToken() string {
	return uuid.New().String()
}

func (s *PostgresStore) CreateUser(u *User) (*User, bool) {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	if u.CreatedAt.IsZero() {
		u.CreatedAt = time.Now()
	}
	// 检查邮箱是否存在
	var count int64
	s.db.Model(&User{}).Where("email = ?", u.Email).Count(&count)
	if count > 0 {
		return nil, false
	}

	if err := s.db.Create(u).Error; err != nil {
		return nil, false
	}
	return u, true
}

func (s *PostgresStore) FindUserByEmail(email string) *User {
	var u User
	if err := s.db.Where("email = ?", email).First(&u).Error; err != nil {
		return nil
	}
	return &u
}

func (s *PostgresStore) FindUserByID(id string) *User {
	var u User
	if err := s.db.Where("id = ?", id).First(&u).Error; err != nil {
		return nil
	}
	return &u
}

// ==========================================
// Token Implementation
// ==========================================

func (s *PostgresStore) SaveRefresh(rt *RefreshToken) {
	// Upsert: On conflict update
	// GORM Clause: OnConflict
	// 简单起见，这里假设 Token 是主键或唯一键
	// 根据 memory.go 定义，Token 是 string，在 db 里应该是 unique index
	// 我们用 Save (Upsert)
	s.db.Save(rt)
}

func (s *PostgresStore) GetRefresh(token string) *RefreshToken {
	var rt RefreshToken
	if err := s.db.Where("token = ?", token).First(&rt).Error; err != nil {
		return nil
	}
	return &rt
}

func (s *PostgresStore) RevokeRefresh(token string) {
	s.db.Model(&RefreshToken{}).Where("token = ?", token).Update("revoked", true)
}

// ==========================================
// APIKey Implementation
// ==========================================

func (s *PostgresStore) CreateAPIKey(k *APIKey) *APIKey {
	if k.ID == "" {
		k.ID = uuid.New().String()
	}
	if k.CreatedAt.IsZero() {
		k.CreatedAt = time.Now()
	}
	s.db.Create(k)
	return k
}

func (s *PostgresStore) ListAPIKeysByUser(userID string) []*APIKey {
	var keys []*APIKey
	s.db.Where("user_id = ?", userID).Find(&keys)
	return keys
}

func (s *PostgresStore) GetAPIKey(id string) *APIKey {
	var key APIKey
	if err := s.db.Where("id = ?", id).First(&key).Error; err != nil {
		return nil
	}
	return &key
}

func (s *PostgresStore) DeleteAPIKey(id string) bool {
	res := s.db.Where("id = ?", id).Delete(&APIKey{})
	return res.Error == nil && res.RowsAffected > 0
}

// ==========================================
// Chat Session Implementation
// ==========================================

func (s *PostgresStore) CreateChatSession(cs *ChatSession) *ChatSession {
	if cs.ID == "" {
		cs.ID = uuid.New().String()
	}
	if cs.CreatedAt.IsZero() {
		cs.CreatedAt = time.Now()
	}
	// UpdatedAt should also be set on create
	if cs.UpdatedAt.IsZero() {
		cs.UpdatedAt = time.Now()
	}
	s.db.Create(cs)
	return cs
}

func (s *PostgresStore) ListChatSessionsByUser(userID string) []*ChatSession {
	var sessions []*ChatSession
	s.db.Where("user_id = ?", userID).Order("updated_at desc").Find(&sessions)
	return sessions
}

func (s *PostgresStore) GetChatSession(id string) *ChatSession {
	var cs ChatSession
	if err := s.db.Where("id = ?", id).First(&cs).Error; err != nil {
		return nil
	}
	return &cs
}

func (s *PostgresStore) DeleteChatSession(id string) bool {
	// GORM 默认是软删除，如果 ChatSession 结构体里没有 DeletedAt 字段，就是硬删除
	// 这里 memory store 是硬删除
	res := s.db.Where("id = ?", id).Delete(&ChatSession{})
	return res.Error == nil && res.RowsAffected > 0
}

// ==========================================
// Run Implementation
// ==========================================

func (s *PostgresStore) CreateRun(r *Run) (*Run, error) {
	// 外键检查 Agent 是否存在
	var count int64
	s.db.Model(&Agent{}).Where("id = ?", r.AgentID).Count(&count)
	if count == 0 {
		return nil, fmt.Errorf("agent not found: %s", r.AgentID)
	}

	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	if r.StartedAt.IsZero() {
		r.StartedAt = time.Now()
	}
	// JSON 字段的空值处理交给 GORM 序列化
	if r.InputPayload == nil {
		r.InputPayload = make(map[string]interface{})
	}
	if r.OutputPayload == nil {
		r.OutputPayload = make(map[string]interface{})
	}

	if err := s.db.Create(r).Error; err != nil {
		return nil, err
	}
	return r, nil
}

func (s *PostgresStore) FinishRun(id string, output map[string]interface{}, status string) bool {
	// 只更新部分字段
	res := s.db.Model(&Run{}).Where("id = ?", id).Updates(map[string]interface{}{
		"output_payload": output,
		"status":         status,
		"finished_at":    time.Now(),
	})
	return res.Error == nil && res.RowsAffected > 0
}

func (s *PostgresStore) ListRunsBySession(sessionID string) []*Run {
	var runs []*Run
	s.db.Where("session_id = ?", sessionID).Order("started_at desc").Find(&runs)
	return runs
}

func (s *PostgresStore) ListRunsByUser(userID string) []*Run {
	var runs []*Run
	s.db.Where("user_id = ?", userID).Order("started_at desc").Find(&runs)
	return runs
}

func (s *PostgresStore) GetRun(runID string) *Run {
	var run Run
	if err := s.db.Where("id = ?", runID).First(&run).Error; err != nil {
		return nil
	}
	return &run
}

// ==========================================
// Run Step Implementation
// ==========================================

func (s *PostgresStore) CreateRunStep(rs *RunStep) *RunStep {
	if rs.ID == "" {
		rs.ID = uuid.New().String()
	}
	if rs.StartedAt.IsZero() {
		rs.StartedAt = time.Now()
	}
	s.db.Create(rs)
	return rs
}

func (s *PostgresStore) FinishRunStep(id string, out map[string]interface{}, status string, latency int, errMsg string) bool {
	res := s.db.Model(&RunStep{}).Where("id = ?", id).Updates(map[string]interface{}{
		"output_payload": out,
		"status":         status,
		"latency_ms":     latency,
		"error_message":  errMsg,
		"finished_at":    time.Now(),
	})
	return res.Error == nil && res.RowsAffected > 0
}

func (s *PostgresStore) ListRunStepsByRun(runID string) []*RunStep {
	var steps []*RunStep
	// Trace 通常按开始时间正序排列
	s.db.Where("run_id = ?", runID).Order("started_at asc").Find(&steps)
	return steps
}

func (s *PostgresStore) GetRunStep(id string) *RunStep {
	var step RunStep
	if err := s.db.Where("id = ?", id).First(&step).Error; err != nil {
		return nil
	}
	return &step
}

// ==========================================
// Chat Message Implementation
// ==========================================

func (s *PostgresStore) CreateChatMessage(cm *ChatMessage) *ChatMessage {
	if cm.ID == "" {
		cm.ID = uuid.New().String()
	}
	if cm.CreatedAt.IsZero() {
		cm.CreatedAt = time.Now()
	}
	if cm.Content == nil {
		cm.Content = make(map[string]interface{})
	}
	s.db.Create(cm)
	return cm
}

func (s *PostgresStore) ListChatMessagesBySession(sessionID string) []*ChatMessage {
	var msgs []*ChatMessage
	// 聊天记录按时间正序
	s.db.Where("session_id = ?", sessionID).Order("created_at asc").Find(&msgs)
	return msgs
}

func (s *PostgresStore) ListChatMessagesByRun(runID string) []*ChatMessage {
	var msgs []*ChatMessage
	s.db.Where("run_id = ?", runID).Order("created_at asc").Find(&msgs)
	return msgs
}

// ==========================================
// Agent Implementation
// ==========================================

func (s *PostgresStore) CreateAgent(a *Agent) *Agent {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	now := time.Now()
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}
	if a.UpdatedAt.IsZero() {
		a.UpdatedAt = now
	}
	// JSON fields init
	if a.Meta == nil {
		a.Meta = make(map[string]interface{})
	}
	if a.ExtraConfig == nil {
		a.ExtraConfig = make(map[string]interface{})
	}

	s.db.Create(a)
	return a
}

func (s *PostgresStore) UpdateAgent(id string, f func(*Agent)) bool {
	var a Agent
	if err := s.db.Where("id = ?", id).First(&a).Error; err != nil {
		return false
	}

	// Apply modifications
	f(&a)
	a.UpdatedAt = time.Now()

	// Update columns
	// 注意：GORM Updates 默认忽略零值，如果要更新为零值需使用 Select 或 Map
	// 这里假设 f 修改后的 a 是全量数据，或者我们只更新非零值
	// 保险起见，我们Save整个对象
	if err := s.db.Save(&a).Error; err != nil {
		return false
	}
	return true
}

func (s *PostgresStore) DeleteAgent(id string) bool {
	res := s.db.Where("id = ?", id).Delete(&Agent{})
	return res.Error == nil && res.RowsAffected > 0
}

func (s *PostgresStore) GetAgent(id string) *Agent {
	var a Agent
	if err := s.db.Where("id = ?", id).First(&a).Error; err != nil {
		return nil
	}
	return &a
}

func (s *PostgresStore) ListAgents() []*Agent {
	var agents []*Agent
	s.db.Find(&agents)
	return agents
}

// ==========================================
// User Integration Implementation
// ==========================================

func (s *PostgresStore) CreateIntegration(in *UserIntegration) *UserIntegration {
	if in.ID == "" {
		in.ID = uuid.New().String()
	}
	now := time.Now()
	if in.CreatedAt.IsZero() {
		in.CreatedAt = now
	}
	if in.UpdatedAt.IsZero() {
		in.UpdatedAt = now
	}
	s.db.Create(in)
	return in
}

func (s *PostgresStore) ListIntegrationsByUser(userID string) []*UserIntegration {
	var ins []*UserIntegration
	s.db.Where("user_id = ?", userID).Find(&ins)
	return ins
}

// ==========================================
// KnowledgeBase Implementation
// ==========================================

func (s *PostgresStore) CreateKnowledgeBase(kb *KnowledgeBase) *KnowledgeBase {
	if kb.ID == "" {
		kb.ID = uuid.New().String()
	}
	now := time.Now()
	if kb.CreatedAt.IsZero() {
		kb.CreatedAt = now
	}
	if kb.UpdatedAt.IsZero() {
		kb.UpdatedAt = now
	}
	if kb.MetaInfo == nil {
		kb.MetaInfo = make(map[string]interface{})
	}
	s.db.Create(kb)
	return kb
}

func (s *PostgresStore) GetKnowledgeBase(id string) *KnowledgeBase {
	var kb KnowledgeBase
	if err := s.db.Where("id = ?", id).First(&kb).Error; err != nil {
		return nil
	}
	return &kb
}

func (s *PostgresStore) ListKnowledgeBasesByUser(userID string) []*KnowledgeBase {
	var kbs []*KnowledgeBase
	s.db.Where("user_id = ?", userID).Find(&kbs)
	return kbs
}

func (s *PostgresStore) UpdateKnowledgeBase(id string, f func(*KnowledgeBase)) bool {
	var kb KnowledgeBase
	if err := s.db.Where("id = ?", id).First(&kb).Error; err != nil {
		return false
	}

	f(&kb)
	kb.UpdatedAt = time.Now()

	if err := s.db.Save(&kb).Error; err != nil {
		return false
	}
	return true
}

// ==========================================
// MCP Implementation
// ==========================================

func (s *PostgresStore) CreateMCPServer(ms *MCPServer) *MCPServer {
	if ms.ID == "" {
		ms.ID = uuid.New().String()
	}
	now := time.Now()
	if ms.CreatedAt.IsZero() {
		ms.CreatedAt = now
	}
	if ms.UpdatedAt.IsZero() {
		ms.UpdatedAt = now
	}
	if ms.ConnectionConfig == nil {
		ms.ConnectionConfig = make(map[string]interface{})
	}
	s.db.Create(ms)
	return ms
}

func (s *PostgresStore) ListMCPServersByAgent(agentID string) []*MCPServer {
	var servers []*MCPServer
	s.db.Where("agent_id = ?", agentID).Find(&servers)
	return servers
}

func (s *PostgresStore) GetMCPServer(id string) *MCPServer {
	var ms MCPServer
	if err := s.db.Where("id = ?", id).First(&ms).Error; err != nil {
		return nil
	}
	return &ms
}

func (s *PostgresStore) ListAllMCPServers() []*MCPServer {
	var servers []*MCPServer
	s.db.Find(&servers)
	return servers
}

func (s *PostgresStore) ListGlobalMCPServers() []*MCPServer {
	var servers []*MCPServer
	s.db.Where("is_global = ?", true).Find(&servers)
	return servers
}

func (s *PostgresStore) FindMCPServerByName(agentID string, name string) *MCPServer {
	var ms MCPServer
	// 优先找私有的，再找全局的
	// 这里用 OR 查询
	_ = s.db.Where("(agent_id = ? AND name = ?) OR (is_global = ? AND name = ?)", agentID, name, true, name).
		// 排序让 Agent 自己的优先 (假设 ID 排序或者 CreatedAt 排序)
		// 更好的做法是分别查，或者 accept first match
		// 这里简单处理：查到就返回。如果有两个同名（一个私有一个全局），这可能返回任意一个。
		// 为了严谨，我们先查 Agent 的
		First(&ms).Error

	// 严谨逻辑：先查 Agent 专属
	if err := s.db.Where("agent_id = ? AND name = ?", agentID, name).First(&ms).Error; err == nil {
		return &ms
	}
	// 再查全局
	if err := s.db.Where("is_global = ? AND name = ?", true, name).First(&ms).Error; err == nil {
		return &ms
	}

	return nil
}

func (s *PostgresStore) CreateMCPTool(t *MCPTool) *MCPTool {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	now := time.Now()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	if t.UpdatedAt.IsZero() {
		t.UpdatedAt = now
	}
	if t.InputSchema == nil {
		t.InputSchema = make(map[string]interface{})
	}
	s.db.Create(t)
	return t
}

func (s *PostgresStore) ListMCPToolsByServer(serverID string) []*MCPTool {
	var tools []*MCPTool
	s.db.Where("server_id = ?", serverID).Find(&tools)
	return tools
}

func (s *PostgresStore) ListMCPToolsByAgent(agentID string) []*MCPTool {
	// GORM Join Query
	// SELECT t.* FROM mcp_tools t JOIN mcp_servers s ON t.server_id = s.id WHERE s.agent_id = ?
	var tools []*MCPTool
	s.db.Table("mcp_tools").
		Joins("JOIN mcp_servers ON mcp_servers.id = mcp_tools.server_id").
		Where("mcp_servers.agent_id = ?", agentID).
		Find(&tools)
	return tools
}

func (s *PostgresStore) ListGlobalMCPTools() []*MCPTool {
	var tools []*MCPTool
	s.db.Table("mcp_tools").
		Joins("JOIN mcp_servers ON mcp_servers.id = mcp_tools.server_id").
		Where("mcp_servers.is_global = ?", true).
		Find(&tools)
	return tools
}

func (s *PostgresStore) FindMCPToolByName(name string) *MCPTool {
	var t MCPTool
	if err := s.db.Where("name = ?", name).First(&t).Error; err != nil {
		return nil
	}
	return &t
}
