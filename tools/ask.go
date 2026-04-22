package tools

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/ollama/ollama/api"
)

type AskUserTool struct{}

func (a *AskUserTool) Name() string {
	return "ask_user"
}

func (a *AskUserTool) Description() string {
	return "Ask the user a question and wait for their text answer."
}

func (a *AskUserTool) Parameters() api.ToolFunctionParameters {
	props := api.NewToolPropertiesMap()
	props.Set("question", api.ToolProperty{
		Type:        api.PropertyType{"string"},
		Description: "The question to ask the user.",
	})
	return api.ToolFunctionParameters{
		Type:       "object",
		Properties: props,
		Required:   []string{"question"},
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
