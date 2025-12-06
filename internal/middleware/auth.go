package middleware

import (
	"context"
	"net/http"
	"strings"

	"example.com/agent-server/internal/auth" // 引用你之前写好的 jwt 工具包
	"github.com/cloudwego/hertz/pkg/app"
)

// ==========================================
// 1. 常量定义 (Context Keys)
// ==========================================
// 使用常量避免代码里出现魔法字符串，防止拼写错误
const (
	CtxKeyUserID = "userID"
	CtxKeyEmail  = "userEmail"
	CtxKeyRoles  = "userRoles"
)

// ==========================================
// 2. 中间件核心逻辑
// ==========================================

// Auth 鉴权中间件
// secret: 用于验证 JWT 签名的密钥
func Auth(secret string) app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		// 1. 获取 Authorization Header
		authHeader := string(ctx.Request.Header.Get("Authorization"))

		// 2. 基础校验：判空
		if authHeader == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, map[string]string{
				"error": "Missing Authorization header",
			})
			return
		}

		// 3. 格式校验：Bearer <token>
		// SplitN 将字符串分为两部分
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, map[string]string{
				"error": "Invalid Authorization format. Expected 'Bearer <token>'",
			})
			return
		}

		tokenString := parts[1]

		// 4. 解析并验证 Token
		// 调用你 internal/auth/jwt.go 里写的 Parse 函数
		claims, err := auth.Parse(secret, tokenString)
		if err != nil {
			// 这里可以根据 err 类型细分，比如是过期了还是签名错误
			// 简单起见统一返回 401
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, map[string]string{
				"error": "Invalid or expired token",
			})
			return
		}

		// 5. 将关键信息注入上下文 (Context)
		// 这样后续的 Handler 就能直接拿到当前是谁在操作
		ctx.Set(CtxKeyUserID, claims.Sub) // Subject 对应 UserID
		ctx.Set(CtxKeyEmail, claims.Email)
		ctx.Set(CtxKeyRoles, claims.Roles)

		// 6. 放行，进入下一个 Handler
		ctx.Next(c)
	}
}

// ==========================================
// 3. 辅助函数 (Helpers)
// ==========================================
// 这些函数供 Handler 层调用，避免手动类型断言

// GetUserID 从上下文中安全获取 UserID
func GetUserID(ctx *app.RequestContext) (string, bool) {
	val, exists := ctx.Get(CtxKeyUserID)
	if !exists {
		return "", false
	}
	id, ok := val.(string)
	return id, ok
}

// GetUserEmail 从上下文中获取 Email
func GetUserEmail(ctx *app.RequestContext) string {
	val, exists := ctx.Get(CtxKeyEmail)
	if !exists {
		return ""
	}
	email, ok := val.(string)
	if !ok {
		return ""
	}
	return email
}

// IsAdmin 检查当前用户是否是管理员 (示例)
func IsAdmin(ctx *app.RequestContext) bool {
	val, exists := ctx.Get(CtxKeyRoles)
	if !exists {
		return false
	}
	roles, ok := val.([]string)
	if !ok {
		return false
	}
	for _, r := range roles {
		if r == "admin" {
			return true
		}
	}
	return false
}
