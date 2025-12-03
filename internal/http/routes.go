package http

import (
    "example.com/agent-server/internal/middleware"
    "github.com/cloudwego/hertz/pkg/app/server"
)

func RegisterRoutes(h *server.Hertz, secret string) {
    h.POST("/api/auth/register", Register)
    h.POST("/api/auth/login", Login(secret))
    h.POST("/api/auth/refresh", Refresh(secret))

    g := h.Group("/api")
    g.Use(middleware.Auth(secret))
    g.POST("/auth/logout", Logout)
    g.GET("/auth/me", Me)

    g.GET("/agents", ListAgents)
    g.POST("/agents", CreateAgent)
    g.GET("/agents/:id", GetAgent)
    g.PUT("/agents/:id", UpdateAgent)
    g.DELETE("/agents/:id", DeleteAgent)

    g.POST("/tasks", CreateTask)
    g.GET("/tasks", ListTasks)
    g.GET("/tasks/:id", GetTask)
}

