package agent

import (
	"encoding/json"
	"fmt"
)

type Registry map[string]Tool

type Agent struct {
	Model    string
	Registry Registry
	Messages []Message
}

func NewAgent(model string) *Agent {
	return &Agent{
		Model:    model,
		Registry: make(Registry),
		Messages: make([]Message, 0),
	}
}

func (a *Agent) RegisterTool(t Tool) {
	a.Registry[t.Name()] = t
}

func (a *Agent) Run(userInput string) (string, error) {
	a.Messages = append(a.Messages, Message{Role: "user", Content: userInput})

	tools := make([]Tool, 0, len(a.Registry))
	for _, t := range a.Registry {
		tools = append(tools, t)
	}

	for {
		resp, err := CallLLM(a.Messages, tools)
		if err != nil {
			return "", err
		}

		a.Messages = append(a.Messages, resp)

		if len(resp.ToolCalls) == 0 {
			return resp.Content, nil
		}

		for _, tc := range resp.ToolCalls {
			tool, ok := a.Registry[tc.Function]
			if !ok {
				a.Messages = append(a.Messages, Message{
					Role:       "tool",
					Content:    fmt.Sprintf("Error: Tool %s not found", tc.Function),
					ToolCallID: tc.ID,
				})
				continue
			}

			var args map[string]any
			if err := json.Unmarshal([]byte(tc.Args), &args); err != nil {
				a.Messages = append(a.Messages, Message{
					Role:       "tool",
					Content:    fmt.Sprintf("Error parsing arguments for tool %s: %v", tc.Function, err),
					ToolCallID: tc.ID,
				})
				continue
			}

			result := tool.Execute(args)
			a.Messages = append(a.Messages, Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
			})
		}
	}
}
