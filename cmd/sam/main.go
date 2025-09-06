package main

import (
	"fmt"
	"os"

	"github.com/wizzomafizzo/mrext/pkg/list"
	"github.com/wizzomafizzo/mrext/pkg/run"
	"github.com/wizzomafizzo/mrext/pkg/attract"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: SAM -list [flags] | -run [flags] | -attract [flags]")
		os.Exit(1)
	}

	cmd := os.Args[1]
	cfg, err := config.LoadUserConfig("SAM", &config.UserConfig{})
	if err != nil {
    	fmt.Println("Config load error:", err)
	} else {
    	fmt.Println("Loaded config from:", cfg.IniPath)
    	fmt.Printf("INI Debug -> Attract Systems: %v | PlayTime: %s | Random: %v\n",
        	cfg.Attract.Systems, cfg.Attract.PlayTime, cfg.Attract.Random)
	}

	args := os.Args[2:]

	switch cmd {
	case "-list":
		list.Run(args)
	case "-run":
    	if err := run.Run(args); err != nil {
        	fmt.Fprintln(os.Stderr, "Run failed:", err)
        	os.Exit(1)
    	}
	case "-attract":
		attract.Run(args)
	default:
		fmt.Printf("Unknown tool: %s\n", cmd)
		os.Exit(1)
	}
}
