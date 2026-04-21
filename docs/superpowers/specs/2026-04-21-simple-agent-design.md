# Design: Simple Agentic Coding Tool (Go)

## Overview
A minimal agentic loop implemented in Go that interacts with Ollama Cloud API to perform tasks using a set of defined tools (skills).

## Architecture

### 1. Tool System
The agent uses an interface-based approach to define and execute skills.

```go
type Tool interface {
    Name() string
    Description() string
    Parameters() ToolParams // Defines JSON schema for LLM
    Execute(args map[string]any) string
}
```

### 2. Agentic Loop
The core logic resides in a conversation loop:
1. **Send**: The current conversation history and tool definitions are sent to Ollama.
2. **Receive**: The LLM returns either a final response or a request to call one or more tools.
3. **Execute**: If tools are requested, the agent looks up the `Tool` interface in a registry, executes it with provided arguments, and captures the output.
4. **Feed Back**: The tool output is added to the conversation history as a `tool` role message.
5. **Repeat**: Step 1 is repeated until the LLM provides a final answer.

### 3. Ollama Integration
- **API**: uses `net/http` to interact with Ollama Cloud.
- **Format**: Sends requests in the Chat API format, providing the `tools` field containing the JSON schema of all registered tools.

## Data Flow
`User Input` $\to$ `Agent Loop` $\to$ `Ollama API` $\to$ `Tool Execution` $\to$ `Result to Context` $\to$ `Final Answer`.

## Success Criteria
- Successfully connects to Ollama Cloud.
- Correctly parses and executes tool calls.
- Maintains conversation state across the loop.
- Provides a final text response to the user.
