package store

import (
    "crypto/rand"
    "encoding/hex"
    "sync"
    "time"
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

type Agent struct {
    ID          string
    Name        string
    Type        string
    Capabilities []string
    Concurrency int
    Tags        []string
    Meta        map[string]interface{}
    Token       string
    Created     time.Time
    Updated     time.Time
}

type Task struct {
    ID        string
    AgentID   string
    Type      string
    Input     map[string]interface{}
    Priority  int
    Status    string
    Result    map[string]interface{}
    Error     string
    Created   time.Time
    Updated   time.Time
    UserID    string
}

type MemoryStore struct {
    mu       sync.RWMutex
    users    map[string]*User
    usersByE map[string]*User
    refresh  map[string]*RefreshToken
    agents   map[string]*Agent
    tasks    map[string]*Task
}

var global *MemoryStore

func init() {
    global = &MemoryStore{users: map[string]*User{}, usersByE: map[string]*User{}, refresh: map[string]*RefreshToken{}, agents: map[string]*Agent{}, tasks: map[string]*Task{}}
}

func Store() *MemoryStore { return global }

func randID(n int) string {
    b := make([]byte, n)
    rand.Read(b)
    s := make([]byte, hex.EncodedLen(len(b)))
    hex.Encode(s, b)
    return string(s)
}

func (m *MemoryStore) RandToken() string { return randID(24) }

func (m *MemoryStore) CreateUser(u *User) (*User, bool) {
    m.mu.Lock()
    defer m.mu.Unlock()
    if _, ok := m.usersByE[u.Email]; ok {
        return nil, false
    }
    u.ID = randID(16)
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
    a.ID = randID(16)
    now := time.Now()
    a.Created = now
    a.Updated = now
    if a.Meta == nil {
        a.Meta = map[string]interface{}{}
    }
    m.agents[a.ID] = a
    return a
}

func (m *MemoryStore) UpdateAgent(id string, f func(*Agent)) bool {
    m.mu.Lock()
    defer m.mu.Unlock()
    if a, ok := m.agents[id]; ok {
        f(a)
        a.Updated = time.Now()
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

func (m *MemoryStore) CreateTask(t *Task) *Task {
    m.mu.Lock()
    defer m.mu.Unlock()
    t.ID = randID(16)
    now := time.Now()
    t.Created = now
    t.Updated = now
    m.tasks[t.ID] = t
    return t
}

func (m *MemoryStore) GetTask(id string) *Task {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.tasks[id]
}

func (m *MemoryStore) ListTasks() []*Task {
    m.mu.RLock()
    defer m.mu.RUnlock()
    res := make([]*Task, 0, len(m.tasks))
    for _, t := range m.tasks {
        res = append(res, t)
    }
    return res
}
