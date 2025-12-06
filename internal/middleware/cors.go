package middleware

import (
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/hertz-contrib/cors"
)

func Cors() app.HandlerFunc {
	return cors.New(cors.Config{
		// 允许的域名
		// 开发环境可以用 "*"，生产环境建议指定具体域名，如 "https://nexus-agent.com"
		AllowOrigins: []string{"*"},

		// 允许的请求方法
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},

		// 允许的前端 Header
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
			"X-Request-ID",
		},

		// 暴露给前端的 Header (比如前端需要读取 X-Request-ID)
		ExposeHeaders: []string{
			"Content-Length",
			"X-Request-ID",
		},

		// 是否允许携带 Cookie
		AllowCredentials: true,

		// 预检请求缓存时间 (12小时)
		MaxAge: 12 * time.Hour,
	})
}
