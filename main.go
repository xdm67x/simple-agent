package main

import (
	"bufio"
	"fmt"
	"os"
	"github.com/xdm67x/simple-agent/agent"
	"github.com/xdm67x/simple-agent/tools"
)

func main() {
	a := agent.NewAgent("llama3")

	a.RegisterTool(&tools.HelloTool{})
	a.RegisterTool(&tools.DateTool{})

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
