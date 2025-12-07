package handler

import (
	"context"
	"net/http"
	"time"

	"example.com/agent-server/internal/middleware"
	"example.com/agent-server/internal/store"
	"example.com/agent-server/pkg/response"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// ==========================================
// DTOs (数据传输对象) - 用于解析请求参数
// ==========================================

type RegisterReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type LoginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

type AuthResponse struct {
	AccessToken  string      `json:"token"`
	RefreshToken string      `json:"refresh_token"`
	User         *store.User `json:"user"`
}

// ==========================================
// Handlers
// ==========================================

// Register 用户注册
func (h *Handler) Register(c context.Context, ctx *app.RequestContext) {
	var req RegisterReq
	if err := ctx.BindAndValidate(&req); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}

	// 1. 检查邮箱是否已存在
	if u := h.Store.FindUserByEmail(req.Email); u != nil {
		response.Error(ctx, http.StatusConflict, 40900, "Email already exists")
		return
	}

	// 2. 密码加密 (Hash)
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		response.ServerError(ctx, err)
		return
	}

	// 3. 创建用户
	user := &store.User{
		Email:    req.Email,
		Name:     req.Name,
		Password: string(hashedPwd), // 存 Hash，不存明文
		Roles:    []string{"user"},
	}

	createdUser, ok := h.Store.CreateUser(user)
	if !ok {
		response.Error(ctx, http.StatusInternalServerError, 50000, "Failed to create user")
		return
	}

	// 注册成功，返回用户信息（为了安全，通常不直接返回 Token，让用户去登录）
	// 这里为了简化流程，可以返回 success
	response.Created(ctx, map[string]interface{}{
		"message": "User registered successfully",
		"user_id": createdUser.ID,
	})
}

// Login 用户登录
func (h *Handler) Login(c context.Context, ctx *app.RequestContext) {
	var req LoginReq
	if err := ctx.BindAndValidate(&req); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}

	// 1. 查找用户
	user := h.Store.FindUserByEmail(req.Email)
	if user == nil {
		response.Unauthorized(ctx, "Invalid email or password")
		return
	}

	// 2. 校验密码
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		response.Unauthorized(ctx, "Invalid email or password")
		return
	}

	// 3. 生成 Tokens
	accessToken, err := h.generateAccessToken(user.ID)
	if err != nil {
		response.ServerError(ctx, err)
		return
	}

	// 简单的 Refresh Token 生成 (实际生产可以使用随机字符串或也是 JWT)
	refreshToken := h.Store.RandToken()

	// 4. 保存 Refresh Token 到数据库
	h.Store.SaveRefresh(&store.RefreshToken{
		Token:   refreshToken,
		UserID:  user.ID,
		Expire:  time.Now().Add(7 * 24 * time.Hour), // 7天过期
		Revoked: false,
	})

	// 5. 返回
	response.Success(ctx, AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	})
}

// Refresh 刷新 Token
func (h *Handler) Refresh(c context.Context, ctx *app.RequestContext) {
	var req RefreshReq
	if err := ctx.BindAndValidate(&req); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}

	// 1. 查库校验 Refresh Token
	rt := h.Store.GetRefresh(req.RefreshToken)
	if rt == nil || rt.Revoked {
		response.Unauthorized(ctx, "Invalid or revoked refresh token")
		return
	}

	if time.Now().After(rt.Expire) {
		response.Unauthorized(ctx, "Refresh token expired")
		return
	}

	// 2. 签发新的 Access Token
	newAccessToken, err := h.generateAccessToken(rt.UserID)
	if err != nil {
		response.ServerError(ctx, err)
		return
	}

	// 3. 返回新 Token (Refresh Token 可以保持不变，或者也轮转，这里简化为不变)
	response.Success(ctx, map[string]string{
		"access_token": newAccessToken,
	})
}

// Logout 登出
func (h *Handler) Logout(c context.Context, ctx *app.RequestContext) {
	var req RefreshReq
	// 尝试绑定，如果不传 refresh token，默认只在前端丢弃 access token 即可
	// 如果传了，后端将其注销
	if err := ctx.BindAndValidate(&req); err == nil && req.RefreshToken != "" {
		h.Store.RevokeRefresh(req.RefreshToken)
	}

	response.Success(ctx, map[string]string{"message": "Logged out"})
}

// Me 获取当前用户信息
// internal/handler/auth.go

func (h *Handler) Me(c context.Context, ctx *app.RequestContext) {
	userID, exists := middleware.GetUserID(ctx)
	if !exists {
		response.Unauthorized(ctx, "Unauthorized")
		return
	}

	user := h.Store.FindUserByID(userID)
	if user == nil {
		response.Error(ctx, http.StatusNotFound, 40400, "User not found")
		return
	}

	// 构造返回，使用 CreatedAt
	safeUser := map[string]interface{}{
		"id":         user.ID,
		"email":      user.Email,
		"name":       user.Name,
		"roles":      user.Roles,
		"created_at": user.CreatedAt, // 这里改成了 created_at
		// "updated_at": user.UpdatedAt, // 可选
	}

	response.Success(ctx, safeUser)
}

// ==========================================
// Helper Functions
// ==========================================

// generateAccessToken 生成 JWT
func (h *Handler) generateAccessToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(1 * time.Hour).Unix(), // 1小时过期
		"iss": "nexus-agent",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(h.JWTSecret)
}
