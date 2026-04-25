package tools

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/xdm67x/simple-agent/agent"
)

type AskUserTool struct{}

func (a *AskUserTool) Name() string {
	return "ask_user"
}

func (a *AskUserTool) Description() string {
	return "Ask the user a question and wait for their text answer. Use this only when you need clarification that cannot be found via bash or other tools. Do not use this to request file contents or directory listings — use bash for that."
}

func (a *AskUserTool) Parameters() agent.ToolFunctionParameters {
	return agent.ToolFunctionParameters{
		Type: "object",
		Properties: map[string]agent.ToolProperty{
			"question": {
				Type:        "string",
				Description: "The question to ask the user.",
			},
		},
		Required: []string{"question"},
	}
}

func (a *AskUserTool) Execute(args map[string]any) string {
	q, ok := args["question"].(string)
	if !ok || q == "" {
		return "Error: missing or invalid 'question' argument"
	}

	fmt.Printf("\nAgent asks: %s\nYour answer: ", q)
	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Sprintf("Error reading input: %v", err)
	}
	return strings.TrimSpace(answer)
}
