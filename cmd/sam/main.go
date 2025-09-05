package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/wizzomafizzo/mrext/pkg/gameindex"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: sam <tool> [flags]")
		fmt.Println("Available tools: gameindex")
		os.Exit(1)
	}

	tool := strings.ToLower(os.Args[1])
	args := os.Args[2:]

	switch tool {
	case "gameindex":
		gameindex.RunCLI(args) // delegate args to gameindex
	default:
		fmt.Printf("Unknown tool: %s\n", tool)
		os.Exit(1)
	}
}
