package middleware

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

func CORS() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		c.Response.Header.Set("Access-Control-Allow-Origin", "*")
		c.Response.Header.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Requested-With, X-Request-ID")
		c.Response.Header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Response.Header.Set("Access-Control-Allow-Credentials", "true")
		if string(c.Method()) == "OPTIONS" {
			c.SetStatusCode(204)
			return
		}
		c.Next(ctx)
	}
}
