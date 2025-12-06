package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"example.com/agent-server/internal/middleware"
	"example.com/agent-server/internal/store"
	"github.com/cloudwego/hertz/pkg/app"
)

// ==========================================
// DTOs
// ==========================================

type CreateAPIKeyReq struct {
	Name string `json:"name" vd:"required"` // 必填，比如 "Macbook CLI"
}

// CreateAPIKeyResp 仅在创建时返回，包含完整的 Key
type CreateAPIKeyResp struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Prefix    string `json:"prefix"`
	Key       string `json:"key"` // <--- 只有这里会返回明文 Key
	CreatedAt string `json:"created_at"`
}

// APIKeyResp 用于列表展示，不包含 Key 明文
type APIKeyResp struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Prefix     string `json:"prefix"`
	LastUsedAt string `json:"last_used_at"`
	CreatedAt  string `json:"created_at"`
}

// ==========================================
// Handlers
// ==========================================

// ListAPIKeys 列出用户的所有 Keys
func (h *Handler) ListAPIKeys(c context.Context, ctx *app.RequestContext) {
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, nil)
		return
	}

	keys := h.Store.ListAPIKeysByUser(userID)

	// 转换为前端友好的 Resp 格式
	res := make([]APIKeyResp, 0, len(keys))
	for _, k := range keys {
		lastUsed := ""
		if !k.LastUsedAt.IsZero() {
			lastUsed = k.LastUsedAt.Format(time.RFC3339)
		}

		res = append(res, APIKeyResp{
			ID:         k.ID,
			Name:       k.Name,
			Prefix:     k.Prefix,
			LastUsedAt: lastUsed,
			CreatedAt:  k.CreatedAt.Format(time.RFC3339),
		})
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"data": res,
	})
}

// CreateAPIKey 创建新的 API Key
func (h *Handler) CreateAPIKey(c context.Context, ctx *app.RequestContext) {
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, nil)
		return
	}

	var req CreateAPIKeyReq
	if err := ctx.BindAndValidate(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// 1. 生成随机 Key (sk-nx-...)
	rawKey, err := generateRandomKey()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate key"})
		return
	}

	// 2. 计算 Hash (存库用)
	// API Key 通常使用 SHA256 即可，速度快且足够安全（因为 Key 本身也是随机的，熵很高）
	hash := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(hash[:])

	// 3. 提取前缀 (展示用)
	// sk-nx-abcdef... -> sk-nx-ab...
	prefix := rawKey[:11] + "..."

	// 4. 存库
	apiKey := &store.APIKey{
		UserID:    userID,
		Name:      req.Name,
		Prefix:    prefix,
		KeyHash:   keyHash,
		CreatedAt: time.Now(),
		// ExpiresAt: ... 可选：设置过期时间
	}
	createdKey := h.Store.CreateAPIKey(apiKey)

	// 5. 返回 (注意：这里必须返回 rawKey，这是用户唯一一次看到它的机会)
	ctx.JSON(http.StatusCreated, CreateAPIKeyResp{
		ID:        createdKey.ID,
		Name:      createdKey.Name,
		Prefix:    createdKey.Prefix,
		Key:       rawKey, // <--- 重要！
		CreatedAt: createdKey.CreatedAt.Format(time.RFC3339),
	})
}

// RevokeAPIKey 删除/吊销 API Key
func (h *Handler) RevokeAPIKey(c context.Context, ctx *app.RequestContext) {
	id := ctx.Param("id")
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, nil)
		return
	}

	// 1. 检查 Key 是否存在且属于当前用户
	targetKey := h.Store.GetAPIKey(id)
	if targetKey == nil {
		ctx.JSON(http.StatusNotFound, map[string]string{"error": "API Key not found"})
		return
	}

	if targetKey.UserID != userID {
		ctx.JSON(http.StatusForbidden, map[string]string{"error": "You do not own this key"})
		return
	}

	// 2. 删除
	if h.Store.DeleteAPIKey(id) {
		ctx.JSON(http.StatusOK, map[string]string{"message": "API Key revoked"})
	} else {
		ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to revoke key"})
	}
}

// ==========================================
// Helper Functions
// ==========================================

// generateRandomKey 生成格式为 sk-nx-<32字节hex> 的随机字符串
func generateRandomKey() (string, error) {
	bytes := make([]byte, 32) // 32 bytes = 256 bits entropy
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return fmt.Sprintf("sk-nx-%s", hex.EncodeToString(bytes)), nil
}
