package run

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/wizzomafizzo/mrext/pkg/config"
	"github.com/wizzomafizzo/mrext/pkg/games"
	"github.com/wizzomafizzo/mrext/pkg/mister"
)

// Run executes a game directly.
// Usage: SAM run <gamefile|amigavision name>
func Run(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: SAM run <gamefile|amigavision name>")
		os.Exit(1)
	}

	target := args[0]

	// Case 1: AmigaVision name (no slash/backslash)
	if !strings.ContainsAny(target, "/\\") {
		amigaShared := findAmigaShared()
		if amigaShared == "" {
			fmt.Fprintln(os.Stderr, "games/Amiga/shared folder not found")
			os.Exit(1)
		}

		tmpShared := "/tmp/.SAM_tmp/Amiga_shared"
		os.RemoveAll(tmpShared)
		os.MkdirAll(tmpShared, 0755)

		exec.Command("cp", "-a", amigaShared+"/.", tmpShared).Run()

		bootFile := filepath.Join(tmpShared, "ags_boot")
		content := target + "\n\n"
		os.WriteFile(bootFile, []byte(content), 0644)

		unmount(amigaShared)

		if err := bindMount(tmpShared, amigaShared); err != nil {
			fmt.Fprintf(os.Stderr, "Bind mount failed: %v\n", err)
			os.Exit(1)
		}

		if err := mister.LaunchCore(&config.UserConfig{}, games.Systems["Amiga"]); err != nil {
			fmt.Fprintf(os.Stderr, "Launch failed: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Case 2: MGL file
	if strings.HasSuffix(strings.ToLower(target), ".mgl") {
		if err := mister.LaunchGenericFile(&config.UserConfig{}, target); err != nil {
			fmt.Fprintf(os.Stderr, "Launch failed: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Case 3: generic file path
	system, _ := games.BestSystemMatch(&config.UserConfig{}, target)
	if err := mister.LaunchGame(&config.UserConfig{}, system, target); err != nil {
		fmt.Fprintf(os.Stderr, "Launch failed: %v\n", err)
		os.Exit(1)
	}
}
