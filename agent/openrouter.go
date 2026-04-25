package agent

import (
	"context"
	"fmt"
	"os"

	"github.com/sashabaranov/go-openai"
)

const openRouterBaseURL = "https://openrouter.ai/api/v1"

type openRouterProvider struct {
	client *openai.Client
	model  string
}

func newOpenRouterProvider(model string, cfg ProviderConfig) (*openRouterProvider, error) {
	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENROUTER_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("openrouter api_key not configured and OPENROUTER_API_KEY environment variable not set")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = openRouterBaseURL
	}

	config := openai.DefaultConfig(apiKey)
	config.BaseURL = baseURL

	return &openRouterProvider{
		client: openai.NewClientWithConfig(config),
		model:  model,
	}, nil
}

func (p *openRouterProvider) Chat(ctx context.Context, req ChatRequest) (Message, error) {
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
		return Message{}, fmt.Errorf("no response from OpenRouter")
	}

	return fromOpenAIMessage(resp.Choices[0].Message), nil
}

func (p *openRouterProvider) ListModels(ctx context.Context) ([]string, error) {
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
