package agent

import (
	"encoding/json"
	"fmt"
)

type Registry map[string]Tool

type Agent struct {
	Model    string
	Registry Registry
}

func NewAgent(model string) *Agent {
	return &Agent{
		Model:    model,
		Registry: make(Registry),
	}
}

func (a *Agent) RegisterTool(t Tool) {
	a.Registry[t.Name()] = t
}

func (a *Agent) Run(userInput string) (string, error) {
	messages := []Message{
		{Role: "user", Content: userInput},
	}

	tools := make([]Tool, 0, len(a.Registry))
	for _, t := range a.Registry {
		tools = append(tools, t)
	}

	for {
		resp, err := CallLLM(messages, tools)
		if err != nil {
			return "", err
		}

		messages = append(messages, resp)

		if len(resp.ToolCalls) == 0 {
			return resp.Content, nil
		}

		for _, tc := range resp.ToolCalls {
			tool, ok := a.Registry[tc.Function]
			if !ok {
				messages = append(messages, Message{
					Role:    "tool",
					Content: fmt.Sprintf("Error: Tool %s not found", tc.Function),
				})
				continue
			}

			var args map[string]any
			if err := json.Unmarshal([]byte(tc.Args), &args); err != nil {
				messages = append(messages, Message{
					Role:    "tool",
					Content: fmt.Sprintf("Error parsing arguments for tool %s: %v", tc.Function, err),
				})
				continue
			}

			result := tool.Execute(args)
			messages = append(messages, Message{
				Role:    "tool",
				Content: result,
			})
		}
	}
}
