package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/xdm67x/simple-agent/agent"
	"github.com/xdm67x/simple-agent/tools"
)

func main() {
	model := os.Getenv("OLLAMA_MODEL")
	if model == "" {
		model = "gemma4:31b-cloud"
	}

	a, err := agent.NewAgent(model)
	if err != nil {
		fmt.Printf("Failed to create agent: %v\n", err)
		os.Exit(1)
	}

	a.RegisterTool(&tools.BashTool{})
	a.RegisterTool(&tools.AskUserTool{})

	fmt.Println("Agent started. Type your messages below.")
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			break
		}

		input := scanner.Text()
		if input == "exit" || input == "quit" {
			break
		}

		resp, err := a.Run(input)
		if err != nil {
			fmt.Printf("Agent error: %v\n", err)
			continue
		}

		fmt.Printf("Agent: %s\n", resp)
	}
}
