package tools

import (
	"fmt"
	"github.com/xdm67x/simple-agent/agent"
)

type HelloTool struct{}

func (t *HelloTool) Name() string {
	return "hello"
}

func (t *HelloTool) Description() string {
	return "Says hello to the user"
}

func (t *HelloTool) Parameters() agent.ToolParams {
	return agent.ToolParams{
		Type: "object",
		Properties: map[string]any{
			"name": map[string]any{
				"type": "string",
				"description": "The name of the person to greet",
			},
		},
		Required: []string{"name"},
	}
}

func (t *HelloTool) Execute(args map[string]any) string {
	name, ok := args["name"].(string)
	if !ok {
		return "Error: name parameter is required and must be a string"
	}
	return fmt.Sprintf("Hello, %s!", name)
}
