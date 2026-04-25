package agent

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

type Registry map[string]Tool

type Agent struct {
	Model           string
	actualModel     string
	provider        Provider
	config          Config
	Registry        Registry
	Messages        []Message
	systemPrompt    string
	OnThinkingStart func()
	OnThinkingEnd   func()
	OnToolCall      func(name string, args map[string]any)
	OnToolResult    func(name string, result string)
	OnSafetyCheck   func(cmd string) bool
	OnAskUser       func(question string) string
}

func NewAgent(cfg Config, systemPrompt string) (*Agent, error) {
	provider, actualModel, err := NewProvider(cfg)
	if err != nil {
		return nil, err
	}

	messages := make([]Message, 0)
	if systemPrompt != "" {
		messages = append(messages, Message{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	return &Agent{
		Model:        cfg.Model,
		actualModel:  actualModel,
		Registry:     make(Registry),
		Messages:     messages,
		systemPrompt: systemPrompt,
		provider:     provider,
		config:       cfg,
	}, nil
}

func (a *Agent) RegisterTool(t Tool) {
	a.Registry[t.Name()] = t
}

func (a *Agent) ListModels() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return a.provider.ListModels(ctx)
}

func (a *Agent) SetModel(model string) error {
	a.config.Model = model
	provider, actualModel, err := NewProvider(a.config)
	if err != nil {
		return err
	}
	a.Model = model
	a.actualModel = actualModel
	a.provider = provider
	return nil
}

func (a *Agent) Clear() {
	messages := make([]Message, 0)
	if a.systemPrompt != "" {
		messages = append(messages, Message{
			Role:    "system",
			Content: a.systemPrompt,
		})
	}
	a.Messages = messages
}

func (a *Agent) Run(userInput string) (string, error) {
	a.Messages = append(a.Messages, Message{Role: "user", Content: userInput})

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
				a.Messages = append(a.Messages, Message{
					Role:       "tool",
					Content:    fmt.Sprintf("Error: Tool %s not found", tc.Function.Name),
					ToolCallID: tc.ID,
				})
				continue
			}

			args := tc.Function.Arguments

			if a.OnToolCall != nil {
				a.OnToolCall(tool.Name(), args)
			}

			// Bash safety gate
			if tool.Name() == "bash" {
				cmdStr, _ := args["command"].(string)
				if cmdStr != "" {
					safe, err := a.safetyCheck(cmdStr)
					if err != nil {
						a.Messages = append(a.Messages, Message{
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
							a.Messages = append(a.Messages, Message{
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

			a.Messages = append(a.Messages, Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
			})
		}
	}
}

func (a *Agent) callChat(tools []Tool) (Message, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	apiTools := make([]APITool, 0, len(tools))
	for _, t := range tools {
		apiTools = append(apiTools, APITool{
			Type: "function",
			Function: ToolFunction{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  t.Parameters(),
			},
		})
	}

	req := ChatRequest{
		Model:    a.actualModel,
		Messages: a.Messages,
		Tools:    apiTools,
		Stream:   false,
	}

	return a.provider.Chat(ctx, req)
}

func (a *Agent) safetyCheck(command string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := ChatRequest{
		Model: a.actualModel,
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are a security gatekeeper. A bash command will be provided. Determine if it is safe to execute automatically without user confirmation. A command is UNSAFE if it modifies, deletes, overwrites files, changes system state, installs software, or accesses sensitive data. Safe commands are read-only (e.g., ls, cat, grep, echo, pwd, df, ps). Answer only 'yes' if safe, or 'no' if unsafe. Do not provide any other output.",
			},
			{
				Role:    "user",
				Content: command,
			},
		},
		Stream: false,
	}

	resp, err := a.provider.Chat(ctx, req)
	if err != nil {
		return false, err
	}

	answer := strings.TrimSpace(strings.ToLower(resp.Content))
	if answer == "" {
		return false, fmt.Errorf("safety check returned empty response")
	}

	return answer == "yes", nil
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
