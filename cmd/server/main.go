package main

import (
	"os"

	"example.com/agent-server/internal/http"
	"example.com/agent-server/internal/middleware"

	hertzServer "github.com/cloudwego/hertz/pkg/app/server"
)

func main() {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-secret"
	}

	h := hertzServer.Default(hertzServer.WithHostPorts(addr))
	h.Use(middleware.RequestID())
	h.Use(middleware.CORS())

	http.RegisterRoutes(h, secret)

	h.Spin()
}
