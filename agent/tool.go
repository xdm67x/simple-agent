package agent

type Tool interface {
	Name() string
	Description() string
	Parameters() ToolFunctionParameters
	Execute(args map[string]any) string
}
