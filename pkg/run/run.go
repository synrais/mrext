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

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func bindMount(src, dst string) error {
	os.MkdirAll(dst, 0755)
	cmd := exec.Command("mount", "-o", "bind", src, dst)
	return cmd.Run()
}

func unmount(path string) {
	exec.Command("umount", path).Run()
}

func findAmigaShared() string {
	amigaPaths := games.GetSystemPaths(&config.UserConfig{}, []games.System{games.Systems["Amiga"]})
	for _, p := range amigaPaths {
		candidate := filepath.Join(p.Path, "shared")
		if pathExists(candidate) {
			return candidate
		}
	}
	// fallback: try usb0-3
	for i := 0; i < 4; i++ {
		usbCandidate := fmt.Sprintf("/media/usb%d/games/Amiga/shared", i)
		if pathExists(usbCandidate) {
			return usbCandidate
		}
	}
	// fallback: fat
	if pathExists("/media/fat/games/Amiga/shared") {
		return "/media/fat/games/Amiga/shared"
	}
	return ""
}

// Run launches a game or AmigaVision target.
func Run(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("Usage: SAM -run <path-or-name>")
	}
	runPath := args[0]

	// Case 1: AmigaVision name (anything without slash/backslash)
	if !strings.ContainsAny(runPath, "/\\") {
		amigaShared := findAmigaShared()
		if amigaShared == "" {
			return fmt.Errorf("games/Amiga/shared folder not found")
		}

		tmpShared := "/tmp/.SAM_tmp/Amiga_shared"
		os.RemoveAll(tmpShared)
		os.MkdirAll(tmpShared, 0755)

		exec.Command("cp", "-a", amigaShared+"/.", tmpShared).Run()

		bootFile := filepath.Join(tmpShared, "ags_boot")
		content := runPath + "\n\n"
		os.WriteFile(bootFile, []byte(content), 0644)

		unmount(amigaShared)

		if err := bindMount(tmpShared, amigaShared); err != nil {
			return fmt.Errorf("Bind mount failed: %v", err)
		}

		return mister.LaunchCore(&config.UserConfig{}, games.Systems["Amiga"])
	}

	// Case 2: MGL file
	if strings.HasSuffix(strings.ToLower(runPath), ".mgl") {
		return mister.LaunchGenericFile(&config.UserConfig{}, runPath)
	}

	// Case 3: generic file path
	system, _ := games.BestSystemMatch(&config.UserConfig{}, runPath)
	return mister.LaunchGame(&config.UserConfig{}, system, runPath)
}
