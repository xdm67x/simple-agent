package agent

type ToolParams struct {
    Type       string `json:"type"`
    Properties map[string]any `json:"properties"`
    Required   []string `json:"required"`
}

type Tool interface {
    Name() string
    Description() string
    Parameters() ToolParams
    Execute(args map[string]any) string
}
