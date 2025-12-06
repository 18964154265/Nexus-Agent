package handler

import (
	"context"

	"example.com/agent-server/pkg/types"
	"github.com/cloudwego/hertz/pkg/app"
)

func ListMCPServers(ctx context.Context, c *app.RequestContext) {
	types.Write(c, 0, "OK", map[string]interface{}{"items": []interface{}{}, "total": 0})
}
func RegisterMCPServer(ctx context.Context, c *app.RequestContext) {
	types.Write(c, 0, "OK", map[string]interface{}{"registered": true})
}
func SyncMCPTools(ctx context.Context, c *app.RequestContext) {
	types.Write(c, 0, "OK", map[string]interface{}{"synced": true})
}
func ListMCPTools(ctx context.Context, c *app.RequestContext) {
	types.Write(c, 0, "OK", map[string]interface{}{"items": []interface{}{}, "total": 0})
}
