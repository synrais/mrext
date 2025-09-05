package run

import (
	"flag"
	"fmt"
	"github.com/wizzomafizzo/mrext/pkg/config"
	"github.com/wizzomafizzo/mrext/pkg/games"
	"github.com/wizzomafizzo/mrext/pkg/mister"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// 🔑 Entry point for the run tool when called from SAM
func Run(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	runPath := fs.String("path", "", "launch a single game by path or AmigaVision name")
	_ = fs.Parse(args)

	if *runPath == "" {
		fmt.Fprintln(os.Stderr, "No game path provided")
		os.Exit(1)
	}

	// Case 1: AmigaVision name (anything without slash/backslash)
	if !strings.ContainsAny(*runPath, "/\\") {
		amigaShared := findAmigaShared()
		if amigaShared == "" {
			fmt.Fprintln(os.Stderr, "games/Amiga/shared folder not found")
			os.Exit(1)
		}

		// Create tmp copy of shared
		tmpShared := "/tmp/.SAM_tmp/Amiga_shared"
		os.RemoveAll(tmpShared)
		os.MkdirAll(tmpShared, 0755)

		// Copy real shared into tmp
		exec.Command("cp", "-a", amigaShared+"/.", tmpShared).Run()

		// Write ags_boot
		bootFile := filepath.Join(tmpShared, "ags_boot")
		content := *runPath + "\n\n"
		os.WriteFile(bootFile, []byte(content), 0644)

		// Always unmount first
		unmount(amigaShared)

		// Bind tmp shared over real shared
		if err := bindMount(tmpShared, amigaShared); err != nil {
			fmt.Fprintf(os.Stderr, "Bind mount failed: %v\n", err)
			os.Exit(1)
		}

		// Launch minimig
		err := mister.LaunchCore(&config.UserConfig{}, games.Systems["Amiga"])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Launch failed: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Case 2: MGL file
	if strings.HasSuffix(strings.ToLower(*runPath), ".mgl") {
		if err := mister.LaunchGenericFile(&config.UserConfig{}, *runPath); err != nil {
			fmt.Fprintf(os.Stderr, "Launch failed: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Case 3: generic file path
	system, _ := games.BestSystemMatch(&config.UserConfig{}, *runPath)
	if err := mister.LaunchGame(&config.UserConfig{}, system, *runPath); err != nil {
		fmt.Fprintf(os.Stderr, "Launch failed: %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
