package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
)

func main() {
	// ==========================================
	// 1. 在这里填入你的配置 (硬编码)
	// ==========================================

	// ⚠️ 务必确认前后没有空格！
	// 硅基流动的 Key 通常以 sk- 开头
	myKey := "sk-iuiztlqmrnkftmdcgpkddselvqktgzattffbulagtmhezipw"

	// 硅基流动官方地址 (必须带 /v1)
	myURL := "https://api.siliconflow.com/v1"

	// 模型名称 (请在硅基流动后台确认此模型你有权限)
	myModel := "Qwen/QwQ-32B"

	// ==========================================
	// 2. 安全检查 (帮你把脉)
	// ==========================================

	// 打印 Key 的长度和前后字符，检查是否有隐形字符
	fmt.Printf("--- 配置检查 ---\n")
	fmt.Printf("BaseURL:  [%s]\n", myURL)
	fmt.Printf("Key长度:   %d\n", len(myKey))
	if len(myKey) > 5 {
		fmt.Printf("Key预览:   [%s...%s]\n", myKey[:4], myKey[len(myKey)-4:])
	}

	// 强制清洗
	cleanKey := strings.TrimSpace(myKey)
	if cleanKey != myKey {
		fmt.Printf("⚠️ 警告: 检测到 Key 前后有空格或换行符！已自动清洗。\n")
	}

	// ==========================================
	// 3. 发起请求
	// ==========================================

	config := openai.DefaultConfig(cleanKey)
	config.BaseURL = myURL

	client := openai.NewClientWithConfig(config)

	fmt.Printf("\n--- 正在请求 LLM (%s) ---\n", myModel)

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: myModel,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: "Hello, 1+1=?",
				},
			},
		},
	)

	if err != nil {
		fmt.Printf("\n❌ 请求失败!\n")
		fmt.Printf("错误详情: %v\n", err)
		return
	}

	fmt.Printf("\n✅ 请求成功!\n")
	fmt.Printf("回复: %s\n", resp.Choices[0].Message.Content)
}
