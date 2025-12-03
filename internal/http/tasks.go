package http

import (
    "context"

    "example.com/agent-server/internal/store"
    "example.com/agent-server/pkg/types"
    "github.com/cloudwego/hertz/pkg/app"
)

type taskCreateReq struct {
    AgentID  string                 `json:"agent_id"`
    Type     string                 `json:"type"`
    Input    map[string]interface{} `json:"input"`
    Priority int                    `json:"priority"`
}

func CreateTask(ctx context.Context, c *app.RequestContext) {
    var req taskCreateReq
    if err := c.Bind(&req); err != nil || req.Type == "" {
        types.Write(c, 1001, "invalid_params", nil)
        return
    }
    uid, _ := c.Get("user_id")
    t := &store.Task{AgentID: req.AgentID, Type: req.Type, Input: req.Input, Priority: req.Priority, Status: "queued", UserID: uid.(string)}
    created := store.Store().CreateTask(t)
    types.Write(c, 0, "OK", map[string]interface{}{"task_id": created.ID, "status": created.Status})
}

func GetTask(ctx context.Context, c *app.RequestContext) {
    id := c.Param("id")
    t := store.Store().GetTask(id)
    if t == nil {
        types.Write(c, 2001, "not_found", nil)
        return
    }
    types.Write(c, 0, "OK", t)
}

func ListTasks(ctx context.Context, c *app.RequestContext) {
    list := store.Store().ListTasks()
    types.Write(c, 0, "OK", map[string]interface{}{"items": list, "total": len(list)})
}

