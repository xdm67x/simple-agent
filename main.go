package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/xdm67x/simple-agent/agent"
	"github.com/xdm67x/simple-agent/tools"
)

func runSpinner(stop, done chan struct{}) {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	i := 0
	clear := strings.Repeat(" ", 60)
	for {
		select {
		case <-stop:
			fmt.Printf("\r%s\r", clear)
			close(done)
			return
		case <-time.After(100 * time.Millisecond):
			fmt.Printf("\rAgent: %s Réfléchit...", frames[i])
			i = (i + 1) % len(frames)
		}
	}
}

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

	var stopSpinner, spinnerDone chan struct{}

	a.OnThinkingStart = func() {
		stopSpinner = make(chan struct{})
		spinnerDone = make(chan struct{})
		go runSpinner(stopSpinner, spinnerDone)
	}

	a.OnThinkingEnd = func() {
		if stopSpinner != nil {
			close(stopSpinner)
			<-spinnerDone
			stopSpinner = nil
			spinnerDone = nil
		}
	}

	a.OnToolCall = func(name string, args map[string]any) {
		argsJSON, _ := json.Marshal(args)
		fmt.Printf("\nAgent: 🔧 Utilise l'outil %s(%s)\n", name, string(argsJSON))
	}

	a.OnToolResult = func(name string, result string) {
		summary := result
		if len(summary) > 80 {
			summary = summary[:77] + "..."
		}
		fmt.Printf("Agent: ✅ Outil %s terminé — %s\n", name, summary)
	}

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
