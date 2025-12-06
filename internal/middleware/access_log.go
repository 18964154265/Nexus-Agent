package middleware

import (
	"context"
	"log"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
)

// AccessLog 记录每个请求的简要信息
func AccessLog() app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		start := time.Now()

		// 继续处理后续逻辑
		ctx.Next(c)

		end := time.Now()
		latency := end.Sub(start)

		// 获取 RequestID (由上面的中间件生成)
		reqID := ctx.Response.Header.Get("X-Request-ID")

		// 获取状态码
		statusCode := ctx.Response.StatusCode()

		// 获取请求方法和路径
		method := string(ctx.Request.Method())
		path := string(ctx.Request.URI().Path())
		clientIP := ctx.ClientIP()

		// 打印日志 (生产环境建议使用 zap 或 logrus)
		// 格式: [200] GET /api/agents (12.3ms) | ip=127.0.0.1 | req_id=...
		log.Printf("[%d] %s %s (%v) | ip=%s | req_id=%s",
			statusCode,
			method,
			path,
			latency,
			clientIP,
			reqID,
		)
	}
}
