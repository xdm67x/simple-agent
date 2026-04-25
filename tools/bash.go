package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/xdm67x/simple-agent/agent"
)

type BashTool struct{}

func (b *BashTool) Name() string {
	return "bash"
}

func (b *BashTool) Description() string {
	return "Execute a bash shell command and return its combined stdout and stderr. Use this to explore the project structure (ls, find), read files (cat, head, tail), search content (grep), run tests, or check git status. Always use this tool first when the user asks about files, code, or project context."
}

func (b *BashTool) Parameters() agent.ToolFunctionParameters {
	return agent.ToolFunctionParameters{
		Type: "object",
		Properties: map[string]agent.ToolProperty{
			"command": {
				Type:        "string",
				Description: "The bash command to execute.",
			},
		},
		Required: []string{"command"},
	}
}

func (b *BashTool) Execute(args map[string]any) string {
	cmdStr, ok := args["command"].(string)
	if !ok || cmdStr == "" {
		return "Error: missing or invalid 'command' argument"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", cmdStr)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Sprintf("Error: %v\n%s", err, out.String())
	}
	return out.String()
}
