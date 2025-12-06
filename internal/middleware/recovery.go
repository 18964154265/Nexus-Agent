package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/cloudwego/hertz/pkg/app"
)

func Recovery() app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		defer func() {
			if err := recover(); err != nil {
				// 1. 打印 Panic 堆栈信息到日志，方便排查
				log.Printf("[Panic Recover] err=%v\nstack=%s", err, debug.Stack())

				// 2. 返回 500 Internal Server Error
				ctx.JSON(http.StatusInternalServerError, map[string]string{
					"error": fmt.Sprintf("Internal Server Error: %v", err),
				})

				// 3. 终止后续处理
				ctx.Abort()
			}
		}()
		ctx.Next(c)
	}
}
