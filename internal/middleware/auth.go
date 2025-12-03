package middleware

import (
    "context"
    "strings"

    "example.com/agent-server/internal/auth"
    "github.com/cloudwego/hertz/pkg/app"
)

func Auth(secret string) app.HandlerFunc {
    return func(ctx context.Context, c *app.RequestContext) {
        h := string(c.GetHeader("Authorization"))
        if h == "" {
            c.JSON(401, map[string]interface{}{"code": 1002, "message": "unauthorized"})
            return
        }
        parts := strings.SplitN(h, " ", 2)
        if len(parts) != 2 || parts[0] != "Bearer" {
            c.JSON(401, map[string]interface{}{"code": 1002, "message": "unauthorized"})
            return
        }
        cl, err := auth.Parse(secret, parts[1])
        if err != nil {
            c.JSON(401, map[string]interface{}{"code": 1002, "message": "unauthorized"})
            return
        }
        c.Set("user_id", cl.Sub)
        c.Set("user_email", cl.Email)
        c.Set("user_name", cl.Name)
        c.Set("user_roles", cl.Roles)
        c.Next(ctx)
    }
}

