package http

import (
    "context"

    "example.com/agent-server/internal/store"
    "example.com/agent-server/pkg/types"
    "github.com/cloudwego/hertz/pkg/app"
)

type agentCreateReq struct {
    Name         string                 `json:"name"`
    Type         string                 `json:"type"`
    Capabilities []string               `json:"capabilities"`
    Concurrency  int                    `json:"concurrency"`
    Tags         []string               `json:"tags"`
    Meta         map[string]interface{} `json:"meta"`
    Token        string                 `json:"token"`
}

func CreateAgent(ctx context.Context, c *app.RequestContext) {
    var req agentCreateReq
    if err := c.Bind(&req); err != nil || req.Name == "" {
        types.Write(c, 1001, "invalid_params", nil)
        return
    }
    a := &store.Agent{Name: req.Name, Type: req.Type, Capabilities: req.Capabilities, Concurrency: req.Concurrency, Tags: req.Tags, Meta: req.Meta, Token: req.Token}
    created := store.Store().CreateAgent(a)
    types.Write(c, 0, "OK", map[string]interface{}{"id": created.ID})
}

func ListAgents(ctx context.Context, c *app.RequestContext) {
    list := store.Store().ListAgents()
    types.Write(c, 0, "OK", list)
}

func GetAgent(ctx context.Context, c *app.RequestContext) {
    id := c.Param("id")
    a := store.Store().GetAgent(id)
    if a == nil {
        types.Write(c, 2001, "not_found", nil)
        return
    }
    types.Write(c, 0, "OK", a)
}

type agentUpdateReq struct {
    Name         *string   `json:"name"`
    Capabilities *[]string `json:"capabilities"`
    Concurrency  *int      `json:"concurrency"`
}

func UpdateAgent(ctx context.Context, c *app.RequestContext) {
    id := c.Param("id")
    var req agentUpdateReq
    if err := c.Bind(&req); err != nil {
        types.Write(c, 1001, "invalid_params", nil)
        return
    }
    ok := store.Store().UpdateAgent(id, func(a *store.Agent) {
        if req.Name != nil {
            a.Name = *req.Name
        }
        if req.Capabilities != nil {
            a.Capabilities = *req.Capabilities
        }
        if req.Concurrency != nil {
            a.Concurrency = *req.Concurrency
        }
    })
    if !ok {
        types.Write(c, 2001, "not_found", nil)
        return
    }
    types.Write(c, 0, "OK", map[string]interface{}{"updated": true})
}

func DeleteAgent(ctx context.Context, c *app.RequestContext) {
    id := c.Param("id")
    ok := store.Store().DeleteAgent(id)
    if !ok {
        types.Write(c, 2001, "not_found", nil)
        return
    }
    types.Write(c, 0, "OK", map[string]interface{}{"deleted": true})
}

