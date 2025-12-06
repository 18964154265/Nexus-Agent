package middleware

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/google/uuid"
	"github.com/hertz-contrib/requestid"
)

// RequestID 中间件
// 作用：为每个请求生成唯一的 Trace ID，并注入到 Response Header 中
func RequestID() app.HandlerFunc {
	return requestid.New(
		requestid.WithGenerator(func(ctx context.Context, c *app.RequestContext) string {
			return uuid.New().String()
		}),
		requestid.WithCustomHeaderStrKey("X-Request-ID"),
	)
}

// ==========================================
// 辅助函数 (Helper)
// ==========================================

// GetRequestID 从上下文中获取当前的 Request ID
// 供 Logger 或 Handler 使用
func GetRequestID(ctx *app.RequestContext) string {
	// hertz-contrib/requestid 中间件会将 ID 写入 Response Header
	return ctx.Response.Header.Get("X-Request-ID")
}
