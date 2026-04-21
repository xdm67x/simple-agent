# Simple Agentic Coding Tool Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a simple agentic loop in Go that uses tool calling via Ollama Cloud API to execute hard-coded skills.

**Architecture:** Interface-based tool registry with a loop that processes LLM tool calls and feeds results back into the conversation context.

**Tech Stack:** Go, `net/http`, Ollama Cloud API.

---

### Task 1: Project Setup and Tool Interface

**Files:**
- Create: `agent/tool.go`

- [ ] **Step 1: Define the Tool interface and Parameters struct**

```go
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
```

- [ ] **Step 2: Commit**

```bash
git add agent/tool.go
git commit -m "feat: define tool interface and parameters"
```

### Task 2: Ollama Cloud Client

**Files:**
- Create: `agent/ollama.go`

- [ ] **Step 1: Implement ChatRequest and ChatResponse structs**

```go
package agent

type Message struct {
    Role    string `json:"role"`
    Content string `json:"content"`
    ToolCall []ToolCall `json:"tool_calls,omitempty"`
}

type ToolCall struct {
    ID       string         `json:"id"`
    Function string         `json:"function"`
    Args     map[string]any `json:"arguments"`
}

type ChatRequest struct {
    Model    string    `json:"model"`
    Messages []Message `json:"messages"`
    Tools    []ToolDef `json:"tools,omitempty"`
}

type ToolDef struct {
    Type struct {
        Type string `json:"type"`
    } `json:"type"`
    Function struct {
        Name        string     `json:"name"`
        Description string     `json:"description"`
        Parameters  ToolParams `json:"parameters"`
    } `json:"function"`
}
```

- [ ] **Step 2: Implement the `CallLLM` function using `net/http`**

Implement a function that takes messages and tools, sends them to the Ollama API, and returns the response.

- [ ] **Step 3: Commit**

```bash
git add agent/ollama.go
git commit -m "feat: implement ollama cloud client"
```

### Task 3: Agent Loop and Registry

**Files:**
- Create: `agent/agent.go`

- [ ] **Step 1: Implement ToolRegistry**

A map `map[string]Tool` to store and retrieve tools by name.

- [ ] **Step 2: Implement the `Run` loop**

The loop:
1. Call Ollama.
2. Check for `tool_calls`.
3. If present:
    - Lookup tool in registry.
    - Execute tool.
    - Add result to messages as role `tool`.
    - Repeat.
4. Otherwise: return final content.

- [ ] **Step 3: Commit**

```bash
git add agent/agent.go
git commit -m "feat: implement agentic loop and tool registry"
```

### Task 4: Implement Example Tools

**Files:**
- Create: `tools/hello.go`
- Create: `tools/date.go`

- [ ] **Step 1: Implement Hello Tool** (prints a greeting)
- [ ] **Step 2: Implement Date Tool** (returns current date/time)
- [ ] **Step 3: Commit**

```bash
git add tools/hello.go tools/date.go
git commit -m "feat: add example hello and date tools"
```

### Task 5: Main Entry Point

**Files:**
- Create: `main.go`

- [ ] **Step 1: Instantiate Agent, Register Tools, and Start Loop**

- [ ] **Step 2: Run and verify tool calling flow.**
- [ ] **Step 3: Commit**

```bash
git add main.go
git commit -m "feat: complete agent implementation"
```
