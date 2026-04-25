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
	Model           string
	Registry        Registry
	Messages        []api.Message
	systemPrompt    string
	client          *api.Client
	OnThinkingStart func()
	OnThinkingEnd   func()
	OnToolCall      func(name string, args map[string]any)
	OnToolResult    func(name string, result string)
	OnSafetyCheck   func(cmd string) bool
	OnAskUser       func(question string) string
}

func NewAgent(model string, systemPrompt string) (*Agent, error) {
	client, err := newOllamaClient()
	if err != nil {
		return nil, err
	}

	messages := make([]api.Message, 0)
	if systemPrompt != "" {
		messages = append(messages, api.Message{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	return &Agent{
		Model:        model,
		Registry:     make(Registry),
		Messages:     messages,
		systemPrompt: systemPrompt,
		client:       client,
	}, nil
}

func (a *Agent) RegisterTool(t Tool) {
	a.Registry[t.Name()] = t
}

func (a *Agent) ListModels() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := a.client.List(ctx)
	if err != nil {
		return nil, err
	}

	models := make([]string, 0, len(resp.Models))
	for _, m := range resp.Models {
		models = append(models, m.Name)
	}
	return models, nil
}

func (a *Agent) SetModel(model string) {
	a.Model = model
}

func (a *Agent) Clear() {
	messages := make([]api.Message, 0)
	if a.systemPrompt != "" {
		messages = append(messages, api.Message{
			Role:    "system",
			Content: a.systemPrompt,
		})
	}
	a.Messages = messages
}

func (a *Agent) Run(userInput string) (string, error) {
	a.Messages = append(a.Messages, api.Message{Role: "user", Content: userInput})

	tools := make([]Tool, 0, len(a.Registry))
	for _, t := range a.Registry {
		tools = append(tools, t)
	}

	for {
		if a.OnThinkingStart != nil {
			a.OnThinkingStart()
		}
		resp, err := a.callChat(tools)
		if a.OnThinkingEnd != nil {
			a.OnThinkingEnd()
		}
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

			if a.OnToolCall != nil {
				a.OnToolCall(tool.Name(), args)
			}

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
					var approved bool
					if a.OnSafetyCheck != nil {
						approved = a.OnSafetyCheck(cmdStr)
					} else {
						fmt.Printf("\nThe agent wants to run: %s\nExecute? (y/n): ", cmdStr)
						reader := bufio.NewReader(os.Stdin)
						answer, _ := reader.ReadString('\n')
						answer = strings.TrimSpace(strings.ToLower(answer))
						approved = answer == "y" || answer == "yes"
					}
					if !approved {
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

			var result string
			if tool.Name() == "ask_user" && a.OnAskUser != nil {
				q, _ := args["question"].(string)
				result = a.OnAskUser(q)
			} else {
				result = tool.Execute(args)
			}
			result = compressOutput(result)

			if a.OnToolResult != nil {
				a.OnToolResult(tool.Name(), result)
			}

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
