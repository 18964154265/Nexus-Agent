package response

import (
	"log"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
)

// Response 标准响应结构
type Response struct {
	Code      int         `json:"code"`       // 业务码：0成功，非0失败
	Message   string      `json:"message"`    // 提示信息
	Data      interface{} `json:"data"`       // 数据 payload
	RequestID string      `json:"request_id"` // 链路 ID
}

// ==========================================
// 核心封装
// ==========================================

// Success 成功响应 (HTTP 200 OK)
func Success(ctx *app.RequestContext, data interface{}) {
	// 获取 RequestID (从 Response Header 里拿，因为中间件已经写入了)
	rid := ctx.Response.Header.Get("X-Request-ID")

	ctx.JSON(http.StatusOK, &Response{
		Code:      0, // 0 代表业务成功
		Message:   "success",
		Data:      data,
		RequestID: rid,
	})
}

// Created 创建成功 (HTTP 201 Created)
func Created(ctx *app.RequestContext, data interface{}) {
	rid := ctx.Response.Header.Get("X-Request-ID")
	ctx.JSON(http.StatusCreated, &Response{
		Code:      0,
		Message:   "created",
		Data:      data,
		RequestID: rid,
	})
}

// Error 错误响应 (自定义 HTTP 状态码)
func Error(ctx *app.RequestContext, httpStatus int, bizCode int, message string) {
	rid := ctx.Response.Header.Get("X-Request-ID")

	// 确保 bizCode 不为 0，否则前端会以为成功了
	if bizCode == 0 {
		bizCode = -1
	}

	ctx.JSON(httpStatus, &Response{
		Code:      bizCode,
		Message:   message,
		Data:      nil,
		RequestID: rid,
	})
}

// ServerError 快捷方式：服务器内部错误 (500)
func ServerError(ctx *app.RequestContext, err error) {
	Error(ctx, http.StatusInternalServerError, 50000, "Internal Server Error")
	log.Printf("[ServerError] err=%v", err)
}

// BadRequest 快捷方式：参数错误 (400)
func BadRequest(ctx *app.RequestContext, message string) {
	Error(ctx, http.StatusBadRequest, 40000, message)
}

// Unauthorized 快捷方式：未授权 (401)
func Unauthorized(ctx *app.RequestContext, message string) {
	Error(ctx, http.StatusUnauthorized, 40100, message)
}
