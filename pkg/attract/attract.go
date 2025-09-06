// pkg/attract/attract.go
package attract

import (
	"bufio"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wizzomafizzo/mrext/pkg/run"
)

// readLines reads all non-empty lines from a file.
func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, scanner.Err()
}

// writeLines writes lines to a file (overwrites).
func writeLines(path string, lines []string) error {
	tmp, err := os.CreateTemp("", "list-*.txt")
	if err != nil {
		return err
	}
	defer tmp.Close()

	for _, l := range lines {
		_, _ = tmp.WriteString(l + "\n")
	}
	return os.Rename(tmp.Name(), path)
}

// appendLine appends a single line to a file.
func appendLine(path string, line string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(line + "\n")
	return err
}

// Run is the entry point for the attract tool.
func Run(args []string) {
	fs := flag.NewFlagSet("attract", flag.ExitOnError)
	delay := fs.Int("delay", 40, "seconds between loading each game")
	random := fs.Bool("random", false, "randomize the order of games")
	listDir := fs.String("lists", "/tmp/.SAM_List", "directory containing gamelists")
	historyFile := fs.String("history", "/tmp/.SAM_History.txt", "path to history file")
	_ = fs.Parse(args)

	// Collect all gamelist files from listDir
	files, err := filepath.Glob(filepath.Join(*listDir, "*_gamelist.txt"))
	if err != nil || len(files) == 0 {
		fmt.Println("No gamelists found in", *listDir)
		os.Exit(1)
	}

	// Seed random
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	fmt.Printf("Attract mode running with %ds delay. Ctrl-C to exit.\n", *delay)

	for {
		// Pick a random list file
		listFile := files[r.Intn(len(files))]

		// Load the list
		lines, err := readLines(listFile)
		if err != nil || len(lines) == 0 {
			// Skip empty lists
			continue
		}

		// Pick a game index
		index := 0
		if *random {
			index = r.Intn(len(lines))
		}

		gamePath := lines[index]

		// Display
		name := filepath.Base(gamePath)
		name = strings.TrimSuffix(name, filepath.Ext(name))
		fmt.Printf("%s - %s <%s>\n", time.Now().Format("15:04:05"), name, gamePath)

		// Hand off to run package
		run.Run([]string{gamePath})

		// Update list: remove played game
		lines = append(lines[:index], lines[index+1:]...)
		_ = writeLines(listFile, lines)

		// Append to history
		_ = appendLine(*historyFile, gamePath)

		// Wait
		time.Sleep(time.Duration(*delay) * time.Second)
	}
}
