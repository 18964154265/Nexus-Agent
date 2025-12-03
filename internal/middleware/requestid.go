package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/cloudwego/hertz/pkg/app"
)

func RequestID() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		rid := c.GetHeader("X-Request-ID")
		if len(rid) == 0 {
			b := make([]byte, 16)
			rand.Read(b)
			s := make([]byte, hex.EncodedLen(len(b)))
			hex.Encode(s, b)
			c.Response.Header.Set("X-Request-ID", string(s))
		}
		c.Next(ctx)
	}
}
