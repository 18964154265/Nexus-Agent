package handler

import (
	"example.com/agent-server/internal/middleware" // 引用定义常量的地方
	"github.com/cloudwego/hertz/pkg/app"
)

// GetUserIDFromCtx 安全地从上下文获取 UserID
func GetUserIDFromCtx(ctx *app.RequestContext) (string, bool) {
	val, exists := ctx.Get(middleware.CtxKeyUserID)
	if !exists {
		return "", false
	}
	id, ok := val.(string)
	return id, ok
}
