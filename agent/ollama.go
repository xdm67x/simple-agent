package agent

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/ollama/ollama/api"
)

type ollamaProvider struct {
	client *api.Client
	model  string
}

func newOllamaProvider(model string, cfg ProviderConfig) (*ollamaProvider, error) {
	host := cfg.Host
	if host == "" {
		host = os.Getenv("OLLAMA_HOST")
	}
	if host == "" {
		host = "https://ollama.com"
	}

	baseURL, err := url.Parse(host)
	if err != nil {
		return nil, fmt.Errorf("invalid OLLAMA_HOST: %w", err)
	}

	httpClient := &http.Client{Timeout: 120 * time.Second}

	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OLLAMA_API_KEY")
	}
	if apiKey != "" {
		httpClient.Transport = &authTransport{
			base:  http.DefaultTransport,
			token: apiKey,
		}
	}

	return &ollamaProvider{
		client: api.NewClient(baseURL, httpClient),
		model:  model,
	}, nil
}

func (p *ollamaProvider) Chat(ctx context.Context, req ChatRequest) (Message, error) {
	messages := make([]api.Message, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = toOllamaMessage(m)
	}

	apiTools := make([]api.Tool, len(req.Tools))
	for i, t := range req.Tools {
		apiTools[i] = api.Tool{
			Type: t.Type,
			Function: api.ToolFunction{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  toOllamaParams(t.Function.Parameters),
			},
		}
	}

	stream := false
	chatReq := &api.ChatRequest{
		Model:    p.model,
		Messages: messages,
		Tools:    apiTools,
		Stream:   &stream,
	}

	var final api.Message
	respFunc := func(resp api.ChatResponse) error {
		final = resp.Message
		return nil
	}

	if err := p.client.Chat(ctx, chatReq, respFunc); err != nil {
		return Message{}, fmt.Errorf("chat failed: %w", err)
	}

	return fromOllamaMessage(final), nil
}

func (p *ollamaProvider) ListModels(ctx context.Context) ([]string, error) {
	resp, err := p.client.List(ctx)
	if err != nil {
		return nil, err
	}

	models := make([]string, 0, len(resp.Models))
	for _, m := range resp.Models {
		models = append(models, m.Name)
	}
	return models, nil
}

func toOllamaMessage(m Message) api.Message {
	msg := api.Message{
		Role:       m.Role,
		Content:    m.Content,
		ToolCallID: m.ToolCallID,
	}
	for _, tc := range m.ToolCalls {
		args := api.NewToolCallFunctionArguments()
		for k, v := range tc.Function.Arguments {
			args.Set(k, v)
		}
		msg.ToolCalls = append(msg.ToolCalls, api.ToolCall{
			ID: tc.ID,
			Function: api.ToolCallFunction{
				Name:      tc.Function.Name,
				Arguments: args,
			},
		})
	}
	return msg
}

func fromOllamaMessage(m api.Message) Message {
	msg := Message{
		Role:       m.Role,
		Content:    m.Content,
		ToolCallID: m.ToolCallID,
	}
	for _, tc := range m.ToolCalls {
		msg.ToolCalls = append(msg.ToolCalls, ToolCall{
			ID: tc.ID,
			Function: ToolCallFunction{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments.ToMap(),
			},
		})
	}
	return msg
}

func toOllamaParams(p ToolFunctionParameters) api.ToolFunctionParameters {
	props := api.NewToolPropertiesMap()
	for name, prop := range p.Properties {
		enumVals := make([]any, len(prop.Enum))
		for i, v := range prop.Enum {
			enumVals[i] = v
		}
		props.Set(name, api.ToolProperty{
			Type:        api.PropertyType{prop.Type},
			Description: prop.Description,
			Enum:        enumVals,
		})
	}
	return api.ToolFunctionParameters{
		Type:       p.Type,
		Properties: props,
		Required:   p.Required,
	}
}

type authTransport struct {
	base  http.RoundTripper
	token string
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(req)
}
