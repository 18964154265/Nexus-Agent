package http

import (
	"context"
	"time"

	"example.com/agent-server/internal/auth"
	"example.com/agent-server/internal/store"
	"example.com/agent-server/pkg/types"
	"github.com/cloudwego/hertz/pkg/app"
	"golang.org/x/crypto/bcrypt"
)

type registerReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

func Register(ctx context.Context, c *app.RequestContext) {
	var req registerReq
	if err := c.Bind(&req); err != nil || req.Email == "" || req.Password == "" {
		types.Write(c, 1001, "invalid_params", nil)
		return
	}
	hashed, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	u := &store.User{Email: req.Email, Name: req.Name, Password: string(hashed), Roles: []string{"user"}}
	created, ok := store.Store().CreateUser(u)
	if !ok {
		types.Write(c, 1001, "email_exists", nil)
		return
	}
	types.Write(c, 0, "OK", map[string]interface{}{"user_id": created.ID})
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func Login(secret string) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		var req loginReq
		if err := c.Bind(&req); err != nil || req.Email == "" || req.Password == "" {
			types.Write(c, 1001, "invalid_params", nil)
			return
		}
		u := store.Store().FindUserByEmail(req.Email)
		if u == nil || bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.Password)) != nil {
			types.Write(c, 1002, "unauthorized", nil)
			return
		}
		cl := auth.Claims{Sub: u.ID, Email: u.Email, Name: u.Name, Roles: u.Roles}
		access, err := auth.Sign(secret, cl, time.Hour)
		if err != nil {
			types.Write(c, 5000, "server_error", nil)
			return
		}
		rt := &store.RefreshToken{Token: store.Store().RandToken(), UserID: u.ID, Expire: time.Now().Add(7 * 24 * time.Hour)}
		store.Store().SaveRefresh(rt)
		types.Write(c, 0, "OK", map[string]interface{}{"access_token": access, "expires_in": int64(time.Hour.Seconds()), "refresh_token": rt.Token})
	}
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

func Refresh(secret string) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		var req refreshReq
		if err := c.Bind(&req); err != nil || req.RefreshToken == "" {
			types.Write(c, 1001, "invalid_params", nil)
			return
		}
		r := store.Store().GetRefresh(req.RefreshToken)
		if r == nil || r.Revoked || time.Now().After(r.Expire) {
			types.Write(c, 1002, "unauthorized", nil)
			return
		}
		u := store.Store().FindUserByID(r.UserID)
		if u == nil {
			types.Write(c, 1002, "unauthorized", nil)
			return
		}
		cl := auth.Claims{Sub: u.ID, Email: u.Email, Name: u.Name, Roles: u.Roles}
		access, err := auth.Sign(secret, cl, time.Hour)
		if err != nil {
			types.Write(c, 5000, "server_error", nil)
			return
		}
		types.Write(c, 0, "OK", map[string]interface{}{"access_token": access, "expires_in": int64(time.Hour.Seconds())})
	}
}

func Logout(ctx context.Context, c *app.RequestContext) {
	var m map[string]string
	_ = c.Bind(&m)
	r := m["refresh_token"]
	if r != "" {
		store.Store().RevokeRefresh(r)
	}
	types.Write(c, 0, "OK", map[string]interface{}{})
}

func Me(ctx context.Context, c *app.RequestContext) {
	id, _ := c.Get("user_id")
	email, _ := c.Get("user_email")
	name, _ := c.Get("user_name")
	roles, _ := c.Get("user_roles")
	types.Write(c, 0, "OK", map[string]interface{}{"id": id, "email": email, "name": name, "roles": roles})
}
