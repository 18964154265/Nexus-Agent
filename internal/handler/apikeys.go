package handler

import (
	"context"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
)

// 注意：这是 (h *Handler) 的方法
func (h *Handler) ListAPIKeys(c context.Context, ctx *app.RequestContext) {
	// 1. 从中间件获取当前登录用户 ID (假设中间件存了 "user_id")
	// 如果还没做中间件，这里暂时写死测试，比如 userID := "some-uuid"
	userIDInterface, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		return
	}
	userID := userIDInterface.(string)

	// 2. 调用 Store 层获取数据
	keys := h.Store.ListAPIKeysByUser(userID)

	// 3. 返回 JSON
	ctx.JSON(http.StatusOK, map[string]interface{}{
		"data": keys,
	})
}

func (h *Handler) CreateAPIKey(c context.Context, ctx *app.RequestContext) {
	// 逻辑类似：Bind参数 -> h.Store.CreateAPIKey -> Return JSON
	var req struct {
		Name string `json:"name"`
	}
	if err := ctx.BindAndValidate(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	// 1. 从中间件获取当前登录用户 ID (假设中间件存了 "user_id")
	// 如果还没做中间件，这里暂时写死测试，比如 userID := "some-uuid"
	userIDInterface, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		return
	}
	userID := userIDInterface.(string)
	// 2. 调用 Store 层创建 API Key
	key := h.Store.CreateAPIKey(userID, req.Name)
	// 3. 返回 JSON
	ctx.JSON(http.StatusOK, map[string]interface{}{
		"data": key,
	})
}
