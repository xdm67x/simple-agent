package agent

import (
	"encoding/json"

	"github.com/sashabaranov/go-openai"
)

func toOpenAIMessage(m Message) openai.ChatCompletionMessage {
	msg := openai.ChatCompletionMessage{
		Role:       m.Role,
		Content:    m.Content,
		ToolCallID: m.ToolCallID,
	}
	for _, tc := range m.ToolCalls {
		argsJSON, _ := json.Marshal(tc.Function.Arguments)
		msg.ToolCalls = append(msg.ToolCalls, openai.ToolCall{
			ID:   tc.ID,
			Type: openai.ToolType(tc.Type),
			Function: openai.FunctionCall{
				Name:      tc.Function.Name,
				Arguments: string(argsJSON),
			},
		})
	}
	return msg
}

func fromOpenAIMessage(m openai.ChatCompletionMessage) Message {
	msg := Message{
		Role:    m.Role,
		Content: m.Content,
	}
	for _, tc := range m.ToolCalls {
		var args map[string]any
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
		msg.ToolCalls = append(msg.ToolCalls, ToolCall{
			ID:   tc.ID,
			Type: string(tc.Type),
			Function: ToolCallFunction{
				Name:      tc.Function.Name,
				Arguments: args,
			},
		})
	}
	return msg
}
