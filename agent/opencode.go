package agent

import (
	"context"
	"fmt"
	"os"

	"github.com/sashabaranov/go-openai"
)

type openCodeProvider struct {
	client *openai.Client
	model  string
}

func newOpenCodeProvider(model string, cfg ProviderConfig) (*openCodeProvider, error) {
	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENCODE_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("opencode api_key not configured and OPENCODE_API_KEY environment variable not set")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = os.Getenv("OPENCODE_BASE_URL")
	}
	if baseURL == "" {
		baseURL = "https://api.opencode.ai/v1"
	}

	config := openai.DefaultConfig(apiKey)
	config.BaseURL = baseURL

	return &openCodeProvider{
		client: openai.NewClientWithConfig(config),
		model:  model,
	}, nil
}

func (p *openCodeProvider) Chat(ctx context.Context, req ChatRequest) (Message, error) {
	messages := make([]openai.ChatCompletionMessage, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = toOpenAIMessage(m)
	}

	tools := make([]openai.Tool, len(req.Tools))
	for i, t := range req.Tools {
		tools[i] = openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  t.Function.Parameters,
			},
		}
	}

	resp, err := p.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:    p.model,
		Messages: messages,
		Tools:    tools,
	})
	if err != nil {
		return Message{}, fmt.Errorf("chat completion failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return Message{}, fmt.Errorf("no response from OpenCode")
	}

	return fromOpenAIMessage(resp.Choices[0].Message), nil
}

func (p *openCodeProvider) ListModels(ctx context.Context) ([]string, error) {
	list, err := p.client.ListModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("list models failed: %w", err)
	}

	models := make([]string, 0, len(list.Models))
	for _, m := range list.Models {
		models = append(models, m.ID)
	}
	return models, nil
}
