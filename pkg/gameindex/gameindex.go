package gameindex

import (
	"fmt"
	"github.com/wizzomafizzo/mrext/pkg/config"
	"github.com/wizzomafizzo/mrext/pkg/games"
	"github.com/wizzomafizzo/mrext/pkg/utils"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ---- ID helpers ----
var idMap = map[string]string{
	"Gameboy":        "gb",
	"GameboyColor":   "gbc",
	"GameGear":       "gg",
	"Nintendo64":     "n64",
	"MasterSystem":   "sms",
	"Sega32X":        "s32x",
	"SuperGameboy":   "sgb",
	"TurboGrafx16":   "tgfx16",
	"TurboGrafx16CD": "tgfx16cd",
}

func SamId(id string) string {
	if id, ok := idMap[id]; ok {
		return id
	}
	return id
}

func ReverseId(id string) string {
	for k, v := range idMap {
		if strings.EqualFold(v, id) {
			return k
		}
	}
	return id
}

func gamelistFilename(systemId string) string {
	if id, ok := idMap[systemId]; ok {
		return strings.ToLower(id) + "_gamelist.txt"
	}
	return strings.ToLower(systemId) + "_gamelist.txt"
}

// ---- Gamelist writing ----
func WriteGamelist(gamelistDir string, systemId string, files []string) error {
	gamelistPath := filepath.Join(gamelistDir, gamelistFilename(systemId))
	tmpPath, err := os.CreateTemp("", "gamelist-*.txt")
	if err != nil {
		return err
	}
	for _, file := range files {
		_, _ = tmpPath.WriteString(file + "\n")
	}
	_ = tmpPath.Sync()
	_ = tmpPath.Close()
	return utils.MoveFile(tmpPath.Name(), gamelistPath)
}

func FilterUniqueWithMGL(files []string) []string {
	chosen := make(map[string]string)
	for _, f := range files {
		base := strings.TrimSuffix(strings.ToLower(filepath.Base(f)), filepath.Ext(f))
		ext := strings.ToLower(filepath.Ext(f))
		if prev, ok := chosen[base]; ok {
			if strings.HasSuffix(prev, ".mgl") {
				continue
			}
			if ext == ".mgl" {
				chosen[base] = f
			}
		} else {
			chosen[base] = f
		}
	}
	result := []string{}
	for _, v := range chosen {
		result = append(result, v)
	}
	return result
}

// ---- AmigaVision helpers ----
func parseLines(data string) []string {
	var out []string
	lines := strings.Split(strings.ReplaceAll(data, "\r\n", "\n"), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func writeCustomList(dir, filename string, entries []string) {
	path := filepath.Join(dir, filename)
	tmp, _ := os.CreateTemp("", "amiga-*.txt")
	for _, e := range entries {
		_, _ = tmp.WriteString(e + "\n")
	}
	tmp.Close()
	_ = utils.MoveFile(tmp.Name(), path)
}

func WriteAmigaVisionLists(gamelistDir string, paths []string) {
	var gamesList, demosList []string

	for _, path := range paths {
		filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			switch strings.ToLower(d.Name()) {
			case "games.txt":
				data, _ := os.ReadFile(p)
				gamesList = append(gamesList, parseLines(string(data))...)
			case "demos.txt":
				data, _ := os.ReadFile(p)
				demosList = append(demosList, parseLines(string(data))...)
			}
			return nil
		})
	}

	if len(gamesList) > 0 {
		writeCustomList(gamelistDir, "amigavisiongames_gamelist.txt", gamesList)
	}
	if len(demosList) > 0 {
		writeCustomList(gamelistDir, "amigavisiondemos_gamelist.txt", demosList)
	}
}

// ---- Main workflow ----
func CreateGamelists(gamelistDir string, systemPaths map[string][]string, progress bool, quiet bool, filter bool) int {
	start := time.Now()
	if !quiet && !progress {
		fmt.Println("Finding system folders...")
	}

	totalPaths := 0
	for _, v := range systemPaths {
		totalPaths += len(v)
	}
	totalSteps := totalPaths
	currentStep := 0
	totalGames := 0

	for systemId, paths := range systemPaths {
		var systemFiles []string
		for _, path := range paths {
			if !quiet {
				if progress {
					fmt.Println("XXX")
					fmt.Println(int(float64(currentStep) / float64(totalSteps) * 100))
					fmt.Printf("Scanning %s (%s)\n", systemId, path)
					fmt.Println("XXX")
				} else {
					fmt.Printf("Scanning %s: %s\n", systemId, path)
				}
			}

			files, err := games.GetFiles(systemId, path)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				continue
			}
			systemFiles = append(systemFiles, files...)
			currentStep++
		}

		if filter {
			systemFiles = FilterUniqueWithMGL(systemFiles)
		}

		if len(systemFiles) > 0 {
			totalGames += len(systemFiles)
			_ = WriteGamelist(gamelistDir, systemId, systemFiles)
		}

		if strings.EqualFold(systemId, "Amiga") {
			WriteAmigaVisionLists(gamelistDir, paths)
		}
	}

	if !quiet {
		taken := int(time.Since(start).Seconds())
		if progress {
			fmt.Println("XXX")
			fmt.Println("100")
			fmt.Printf("Indexing complete (%d games in %ds)\n", totalGames, taken)
			fmt.Println("XXX")
		} else {
			fmt.Printf("Indexing complete (%d games in %ds)\n", totalGames, taken)
		}
	}

	return totalGames
}

// ---- Amiga shared discovery ----
func FindAmigaShared() string {
	amigaPaths := games.GetSystemPaths(&config.UserConfig{}, []games.System{games.Systems["Amiga"]})
	for _, p := range amigaPaths {
		candidate := filepath.Join(p.Path, "shared")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	for i := 0; i < 4; i++ {
		usbCandidate := fmt.Sprintf("/media/usb%d/games/Amiga/shared", i)
		if _, err := os.Stat(usbCandidate); err == nil {
			return usbCandidate
		}
	}
	if _, err := os.Stat("/media/fat/games/Amiga/shared"); err == nil {
		return "/media/fat/games/Amiga/shared"
	}
	return ""
}
