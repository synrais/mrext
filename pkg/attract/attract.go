package attract

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/wizzomafizzo/mrext/pkg/config"
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

// parsePlayTime handles "40" or "40-130"
func parsePlayTime(value string, r *rand.Rand) time.Duration {
	if strings.Contains(value, "-") {
		parts := strings.SplitN(value, "-", 2)
		min, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
		max, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
		if max > min {
			return time.Duration(r.Intn(max-min+1)+min) * time.Second
		}
		return time.Duration(min) * time.Second
	}
	secs, _ := strconv.Atoi(value)
	return time.Duration(secs) * time.Second
}

// matchesPattern checks if string matches a wildcard (*foo*, bar*, *baz)
func matchesPattern(s, pattern string) bool {
	p := strings.ToLower(pattern)
	s = strings.ToLower(s)

	if strings.HasPrefix(p, "*") && strings.HasSuffix(p, "*") {
		return strings.Contains(s, strings.Trim(p, "*"))
	}
	if strings.HasPrefix(p, "*") {
		return strings.HasSuffix(s, strings.TrimPrefix(p, "*"))
	}
	if strings.HasSuffix(p, "*") {
		return strings.HasPrefix(s, strings.TrimSuffix(p, "*"))
	}
	return s == p
}

// disabled checks if a game should be blocked by rules
func disabled(system string, gamePath string, cfg *config.UserConfig) bool {
	rules, ok := cfg.Disable[system]
	if !ok {
		return false
	}

	base := strings.ToLower(filepath.Base(gamePath))
	ext := strings.ToLower(filepath.Ext(gamePath))
	dir := strings.ToLower(filepath.Base(filepath.Dir(gamePath)))

	for _, f := range rules.Folders {
		if matchesPattern(dir, f) {
			return true
		}
	}
	for _, f := range rules.Files {
		if matchesPattern(base, f) {
			return true
		}
	}
	for _, e := range rules.Extensions {
		if strings.EqualFold(ext, e) {
			return true
		}
	}
	return false
}

// Run is the entry point for the attract tool.
func Run(_ []string) {
	cfg, _ := config.LoadUserConfig("SAM", &config.UserConfig{})
	attractCfg := cfg.Attract

	listDir := "/tmp/.SAM_List"
	historyFile := "/tmp/.SAM_History.txt"

	allFiles, err := filepath.Glob(filepath.Join(listDir, "*_gamelist.txt"))
	if err != nil || len(allFiles) == 0 {
		fmt.Println("No gamelists found in", listDir)
		os.Exit(1)
	}

	// Filter gamelists by Systems if set in INI
	files := allFiles
	if len(attractCfg.Systems) > 0 {
		allowed := map[string]bool{}
		for _, sys := range attractCfg.Systems {
			allowed[strings.ToLower(strings.TrimSpace(sys))] = true
		}

		var filtered []string
		for _, f := range allFiles {
			base := strings.TrimSuffix(filepath.Base(f), "_gamelist.txt")
			if allowed[strings.ToLower(base)] {
				filtered = append(filtered, f)
			}
		}
		if len(filtered) == 0 {
			fmt.Println("No gamelists match the configured Systems in INI")
			os.Exit(1)
		}
		files = filtered
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	fmt.Printf("Attract mode running. Ctrl-C to exit.\n")

	for {
		listFile := files[r.Intn(len(files))]

		lines, err := readLines(listFile)
		if err != nil || len(lines) == 0 {
			continue
		}

		index := 0
		if attractCfg.Random {
			index = r.Intn(len(lines))
		}

		gamePath := lines[index]
		systemID := strings.TrimSuffix(filepath.Base(listFile), "_gamelist.txt")

		if disabled(systemID, gamePath, cfg) {
			lines = append(lines[:index], lines[index+1:]...)
			_ = writeLines(listFile, lines)
			continue
		}

		// ✅ Log system ID in uppercase
		name := filepath.Base(gamePath)
		name = strings.TrimSuffix(name, filepath.Ext(name))
		fmt.Printf("[%s] %s <%s>\n", strings.ToUpper(systemID), name, gamePath)

		run.Run([]string{gamePath})

		lines = append(lines[:index], lines[index+1:]...)
		_ = writeLines(listFile, lines)
		_ = appendLine(historyFile, fmt.Sprintf("[%s] %s", strings.ToUpper(systemID), gamePath))

		wait := parsePlayTime(attractCfg.PlayTime, r)
		time.Sleep(wait)
	}
}
