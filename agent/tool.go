package agent

import "github.com/ollama/ollama/api"

type Tool interface {
	Name() string
	Description() string
	Parameters() api.ToolFunctionParameters
	Execute(args map[string]any) string
}
