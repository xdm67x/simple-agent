package tools

import (
	"github.com/xdm67x/simple-agent/agent"
	"time"
)

type DateTool struct{}

func (t *DateTool) Name() string {
	return "get_current_date"
}

func (t *DateTool) Description() string {
	return "Returns the current date and time"
}

func (t *DateTool) Parameters() agent.ToolParams {
	return agent.ToolParams{
		Type:       "object",
		Properties: map[string]any{},
		Required:   []string{},
	}
}

func (t *DateTool) Execute(args map[string]any) string {
	return time.Now().String()
}
