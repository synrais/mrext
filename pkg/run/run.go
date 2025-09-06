package run

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wizzomafizzo/mrext/pkg/config"
	"github.com/wizzomafizzo/mrext/pkg/games"
	"github.com/wizzomafizzo/mrext/pkg/mister"
	"github.com/wizzomafizzo/mrext/pkg/utils"
)

// Run launches a game or AmigaVision target.
// It no longer calls os.Exit – instead it returns errors for the caller to handle.
func Run(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("Usage: SAM -run <path-or-name>")
	}

	runPath := args[0]

	// Case 1: AmigaVision (name without slashes)
	if !strings.ContainsAny(runPath, "/\\") {
		amigaShared := findAmigaShared()
		if amigaShared == "" {
			return fmt.Errorf("games/Amiga/shared folder not found")
		}

		tmpShared := filepath.Join(os.TempDir(), "amiga-shared")
		_ = os.RemoveAll(tmpShared)
		_ = os.MkdirAll(tmpShared, 0755)

		// Copy demo/game into tmp shared
		paths := []string{
			filepath.Join(amigaShared, "Games.txt"),
			filepath.Join(amigaShared, "Demos.txt"),
		}
		found := false
		for _, p := range paths {
			lines, _ := utils.ReadLines(p)
			for _, l := range lines {
				if strings.EqualFold(l, runPath) {
					found = true
					_ = os.WriteFile(filepath.Join(tmpShared, "Games.txt"), []byte(runPath+"\n"), 0644)
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return fmt.Errorf("AmigaVision game %q not found", runPath)
		}

		// Bind mount to Amiga shared folder
		if err := bindMount(tmpShared, amigaShared); err != nil {
			return fmt.Errorf("Bind mount failed: %v", err)
		}

		// Launch Amiga core
		return mister.LaunchCore(&config.UserConfig{}, games.Systems["Amiga"])
	}

	// Case 2: MGL file
	if strings.HasSuffix(strings.ToLower(runPath), ".mgl") {
		return mister.LaunchGenericFile(&config.UserConfig{}, runPath)
	}

	// Case 3: Generic file path (ROM, etc.)
	system, _ := games.BestSystemMatch(&config.UserConfig{}, runPath)
	return mister.LaunchGame(&config.UserConfig{}, system, runPath)
}

// findAmigaShared locates the Amiga shared folder on SD or USB.
func findAmigaShared() string {
	paths := []string{
		filepath.Join(config.SdFolder, "games/Amiga/shared"),
		filepath.Join(config.UsbFolder, "games/Amiga/shared"),
	}
	for _, p := range paths {
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
	}
	return ""
}

// bindMount mounts src to dst (used for AmigaVision).
func bindMount(src, dst string) error {
	// BusyBox mount --bind
	if err := utils.Exec("mount", "--bind", src, dst); err != nil {
		return fmt.Errorf("mount bind failed: %v", err)
	}
	return nil
}
