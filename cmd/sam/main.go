package main

import (
	"fmt"
	"os"

	"github.com/wizzomafizzo/mrext/pkg/gameindex"
	"github.com/wizzomafizzo/mrext/pkg/run"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: SAM -gameindex [flags] | -run [flags]")
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "-gameindex":
		gameindex.Run(args)
	case "-run":
		run.Run(args)
	default:
		fmt.Printf("Unknown tool: %s\n", cmd)
		os.Exit(1)
	}
}
