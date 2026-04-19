package main

import (
	"fmt"
)

func main() {
	provider, err := NewOpenRouterProvider("")
	if err != nil {
		panic(err)
	}

	fmt.Printf("Available models: %v\n", provider.Models)

	fmt.Print("🤖> ")

	var input string
	if _, err := fmt.Scanln(&input); err != nil {
		panic(err)
	}

	fmt.Printf("User said: %s\n", input)
}
