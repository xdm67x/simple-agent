package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/ollama/ollama/api"
)

type BashTool struct{}

func (b *BashTool) Name() string {
	return "bash"
}

func (b *BashTool) Description() string {
	return "Execute a bash shell command and return its combined stdout and stderr."
}

func (b *BashTool) Parameters() api.ToolFunctionParameters {
	props := api.NewToolPropertiesMap()
	props.Set("command", api.ToolProperty{
		Type:        api.PropertyType{"string"},
		Description: "The bash command to execute.",
	})
	return api.ToolFunctionParameters{
		Type:       "object",
		Properties: props,
		Required:   []string{"command"},
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
