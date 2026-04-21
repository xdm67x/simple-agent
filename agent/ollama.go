package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type ToolCall struct {
	ID       string `json:"id"`
	Function string `json:"function"`
	Args     string `json:"args"`
}

type Message struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string    `json:"tool_call_id,omitempty"`
}

type ToolDef struct {
	Type     string `json:"type"`
	Function struct {
		Name        string     `json:"name"`
		Description string     `json:"description"`
		Parameters  ToolParams `json:"parameters"`
	} `json:"function"`
}

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Tools    []ToolDef `json:"tools,omitempty"`
}

type ChatResponse struct {
	Message Message `json:"message"`
}

func CallLLM(messages []Message, tools []Tool) (Message, error) {
	url := os.Getenv("OLLAMA_CLOUD_URL")
	if url == "" {
		url = "https://api.ollama.cloud/chat"
	}

	toolDefs := make([]ToolDef, 0, len(tools))
	for _, t := range tools {
		td := ToolDef{Type: "function"}
		td.Function.Name = t.Name()
		td.Function.Description = t.Description()
		td.Function.Parameters = t.Parameters()
		toolDefs = append(toolDefs, td)
	}

	reqBody := ChatRequest{
		Model:    "llama3",
		Messages: messages,
		Tools:    toolDefs,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return Message{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return Message{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Message{}, fmt.Errorf("api returned non-OK status: %d", resp.StatusCode)
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return Message{}, fmt.Errorf("failed to decode response: %w", err)
	}

	return chatResp.Message, nil
}
