package agent

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/ollama/ollama/api"
)

func newOllamaClient() (*api.Client, error) {
	host := os.Getenv("OLLAMA_HOST")
	if host == "" {
		host = "https://ollama.com"
	}

	baseURL, err := url.Parse(host)
	if err != nil {
		return nil, fmt.Errorf("invalid OLLAMA_HOST: %w", err)
	}

	httpClient := &http.Client{Timeout: 120 * time.Second}

	apiKey := os.Getenv("OLLAMA_API_KEY")
	if apiKey != "" {
		httpClient.Transport = &authTransport{
			base:   http.DefaultTransport,
			token:  apiKey,
		}
	}

	return api.NewClient(baseURL, httpClient), nil
}

type authTransport struct {
	base  http.RoundTripper
	token string
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(req)
}

// safetyCheck asks the LLM whether a bash command is safe to execute without user confirmation.
// It returns true if the LLM replies "yes", false otherwise.
func safetyCheck(client *api.Client, model, command string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := &api.ChatRequest{
		Model: model,
		Messages: []api.Message{
			{
				Role:    "system",
				Content: "You are a security gatekeeper. A bash command will be provided. Determine if it is safe to execute automatically without user confirmation. A command is UNSAFE if it modifies, deletes, overwrites files, changes system state, installs software, or accesses sensitive data. Safe commands are read-only (e.g., ls, cat, grep, echo, pwd, df, ps). Answer only 'yes' if safe, or 'no' if unsafe. Do not provide any other output.",
			},
			{
				Role:    "user",
				Content: command,
			},
		},
		Stream: boolPtr(false),
	}

	var answer string
	var gotAnswer bool
	respFunc := func(resp api.ChatResponse) error {
		if resp.Message.Content != "" {
			answer = strings.TrimSpace(strings.ToLower(resp.Message.Content))
			gotAnswer = true
		}
		return nil
	}

	if err := client.Chat(ctx, req, respFunc); err != nil {
		return false, err
	}
	if !gotAnswer {
		return false, fmt.Errorf("safety check returned empty response")
	}

	return answer == "yes", nil
}

func boolPtr(b bool) *bool {
	return &b
}
