package agent

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ollama/ollama/api"
)

type Registry map[string]Tool

type Agent struct {
	Model    string
	Registry Registry
	Messages []api.Message
	client   *api.Client
}

func NewAgent(model string) (*Agent, error) {
	client, err := newOllamaClient()
	if err != nil {
		return nil, err
	}
	return &Agent{
		Model:    model,
		Registry: make(Registry),
		Messages: make([]api.Message, 0),
		client:   client,
	}, nil
}

func (a *Agent) RegisterTool(t Tool) {
	a.Registry[t.Name()] = t
}

func (a *Agent) Run(userInput string) (string, error) {
	a.Messages = append(a.Messages, api.Message{Role: "user", Content: userInput})

	tools := make([]Tool, 0, len(a.Registry))
	for _, t := range a.Registry {
		tools = append(tools, t)
	}

	for {
		resp, err := a.callChat(tools)
		if err != nil {
			return "", err
		}

		a.Messages = append(a.Messages, resp)

		if len(resp.ToolCalls) == 0 {
			return resp.Content, nil
		}

		for _, tc := range resp.ToolCalls {
			tool, ok := a.Registry[tc.Function.Name]
			if !ok {
				a.Messages = append(a.Messages, api.Message{
					Role:       "tool",
					Content:    fmt.Sprintf("Error: Tool %s not found", tc.Function.Name),
					ToolCallID: tc.ID,
				})
				continue
			}

			args := tc.Function.Arguments.ToMap()

			// Bash safety gate
			if tool.Name() == "bash" {
				cmdStr, _ := args["command"].(string)
				if cmdStr != "" {
					safe, err := safetyCheck(a.client, a.Model, cmdStr)
					if err != nil {
						a.Messages = append(a.Messages, api.Message{
							Role:       "tool",
							Content:    fmt.Sprintf("Error during safety check: %v", err),
							ToolCallID: tc.ID,
						})
						continue
					}
					if !safe {
						fmt.Printf("\nThe agent wants to run: %s\nExecute? (y/n): ", cmdStr)
						reader := bufio.NewReader(os.Stdin)
						answer, _ := reader.ReadString('\n')
						answer = strings.TrimSpace(strings.ToLower(answer))
						if answer != "y" && answer != "yes" {
							a.Messages = append(a.Messages, api.Message{
								Role:       "tool",
								Content:    "User declined execution",
								ToolCallID: tc.ID,
							})
							continue
						}
					}
				}
			}

			result := tool.Execute(args)
			result = compressOutput(result)

			a.Messages = append(a.Messages, api.Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
			})
		}
	}
}

func (a *Agent) callChat(tools []Tool) (api.Message, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	apiTools := make([]api.Tool, 0, len(tools))
	for _, t := range tools {
		apiTools = append(apiTools, api.Tool{
			Type: "function",
			Function: api.ToolFunction{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  t.Parameters(),
			},
		})
	}

	stream := false
	req := &api.ChatRequest{
		Model:    a.Model,
		Messages: a.Messages,
		Tools:    apiTools,
		Stream:   &stream,
	}

	var final api.Message
	respFunc := func(resp api.ChatResponse) error {
		final = resp.Message
		return nil
	}

	if err := a.client.Chat(ctx, req, respFunc); err != nil {
		return api.Message{}, fmt.Errorf("chat failed: %w", err)
	}

	return final, nil
}

func compressOutput(s string) string {
	const maxLen = 3000
	if len(s) <= maxLen {
		return s
	}
	head := s[:1500]
	tail := s[len(s)-1500:]
	truncated := len(s) - 3000
	return fmt.Sprintf("%s\n\n[...truncated %d chars...]\n\n%s", head, truncated, tail)
}
