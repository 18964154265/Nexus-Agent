package main

import (
	"log"
	"os"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/joho/godotenv"

	"example.com/agent-server/internal/bootstrap"
	"example.com/agent-server/internal/handler"
	myhttp "example.com/agent-server/internal/http"
	"example.com/agent-server/internal/middleware"
	"example.com/agent-server/internal/service"
	"example.com/agent-server/internal/service/llm"
	"example.com/agent-server/internal/store"
)

func main() {
	// 1. 【核心修改】使用构造函数初始化 DB，确保所有 map 都已 make
	if err := godotenv.Load(); err != nil {
		log.Println("NO env file")
	}
	db := store.NewMemoryStore()

	// 2. 初始化预设的 Agent 团队 (Manager, Coder 等)
	bootstrap.SeedDevOpsTeam(db)
	tempStr := os.Getenv("LLM_TEMPERATURE")
	temp := 0.1
	if t, err := strconv.ParseFloat(tempStr, 32); err == nil {
		temp = t
	}
	llmConfig := llm.LLMConfig{
		ApiKey:      "sk-iuiztlqmrnkftmdcgpkddselvqktgzattffbulagtmhezipw",
		BaseURL:     os.Getenv("LLM_BASE_URL"),
		ModelName:   os.Getenv("LLM_MODEL_NAME"),
		Temperature: float32(temp),
	}
	log.Printf("Init LLM Client: model name:=%s", llmConfig.ModelName)
	llmClient := llm.NewClient(llmConfig)
	bootstrap.SeedMCPServers(db)

	// 3. 初始化 Handler (注入 db)
	// 注意：jwt-secret 应该从环境变量读取，这里为了演示写死
	svc := service.NewService(db, llmClient)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8888"
	}

	h := handler.New(db, "your-secure-jwt-secret-key", svc)

	// 4. 初始化 Hertz Server，绑定到 127.0.0.1:8888
	srv := server.New(server.WithHostPorts("127.0.0.1:" + port))

	//5.注册中间件
	srv.Use(middleware.Recovery())
	srv.Use(middleware.RequestID())
	srv.Use(middleware.AccessLog())
	srv.Use(middleware.Cors())

	// 5. 注册路由
	myhttp.RegisterRoutes(srv, h, "your-secure-jwt-secret-key")

	// 6. 启动服务
	srv.Spin()
}
