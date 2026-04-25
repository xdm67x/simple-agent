package agent

// Message represents a chat message in a provider-agnostic format.
type Message struct {
	Role       string
	Content    string
	ToolCalls  []ToolCall
	ToolCallID string
}

// ToolCall represents a tool call from the model.
type ToolCall struct {
	ID       string
	Type     string
	Function ToolCallFunction
}

// ToolCallFunction holds the function details of a tool call.
type ToolCallFunction struct {
	Name      string
	Arguments map[string]any
}

// APITool represents a tool definition passed to the model.
type APITool struct {
	Type     string
	Function ToolFunction
}

// ToolFunction defines a callable tool.
type ToolFunction struct {
	Name        string
	Description string
	Parameters  ToolFunctionParameters
}

// ToolFunctionParameters describes the JSON schema for tool parameters.
type ToolFunctionParameters struct {
	Type       string                   `json:"type"`
	Properties map[string]ToolProperty  `json:"properties"`
	Required   []string                 `json:"required"`
}

// ToolProperty describes a single parameter property.
type ToolProperty struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
}

// ChatRequest is a provider-agnostic chat completion request.
type ChatRequest struct {
	Model    string
	Messages []Message
	Tools    []APITool
	Stream   bool
}
