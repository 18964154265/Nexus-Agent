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

			//在store.ChatMessage中存tool_calls的原始json
			//取出来时要能反序列化
		}
		if m.Role == openai.ChatMessageRoleTool {
			openaiMsg.ToolCallID = m.Content["tool_call_id"].(string)
		}
		result = append(result, openaiMsg)
	}
	return result
}

func DBToolsToOpenAI(dbTools []*store.MCPTool) []openai.Tool {
	result := make([]openai.Tool, 0, len(dbTools))
	for _, t := range dbTools {
		schemaBytes, _ := json.Marshal(t.InputSchema)
		tool := openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  json.RawMessage(schemaBytes),
			},
		}
		result = append(result, tool)
	}

	return result
}
