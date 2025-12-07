package runner

import (
	"encoding/json"

	"example.com/agent-server/internal/store"
	"github.com/sashabaranov/go-openai"
)

func DBMessageToOpenAI(msgs []*store.ChatMessage) []openai.ChatCompletionMessage {
	result := make([]openai.ChatCompletionMessage, 0, len(msgs))

	for _, m := range msgs {
		openaiMsg := openai.ChatCompletionMessage{
			Role: m.Role,
		}
		if text, ok := m.Content["text"].(string); ok {
			openaiMsg.Content = text
		} else {
			b, _ := json.Marshal(m.Content)
			openaiMsg.Content = string(b)
		}
		if m.Role == openai.ChatMessageRoleAssistant {
			continue
			//在store.ChatMessage中存toool_calls的原始json
			//取出来时要能反序列化
		}
		if m.Role == openai.ChatMessageRoleTool {
			openaiMsg.ToolCallID = m.Content["tool_call_id"].(string)
		}
		result = append(result, openaiMsg)
	}
	return result
}
