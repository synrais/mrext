package list

import (
	"flag"
	"fmt"
	"github.com/wizzomafizzo/mrext/pkg/config"
	"github.com/wizzomafizzo/mrext/pkg/games"
	"github.com/wizzomafizzo/mrext/pkg/utils"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SAM uses slightly different system IDs.
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

func samId(id string) string {
	if id, ok := idMap[id]; ok {
		return id
	}
	return id
}

func reverseId(id string) string {
	for k, v := range idMap {
		if strings.EqualFold(v, id) {
			return k
		}
	}
	return id
}

func gamelistFilename(systemId string) string {
	var prefix string
	if id, ok := idMap[systemId]; ok {
		prefix = id
	} else {
		prefix = systemId
	}
	return strings.ToLower(prefix) + "_gamelist.txt"
}

func writeGamelist(gamelistDir string, systemId string, files []string) string {
	gamelistPath := filepath.Join(gamelistDir, gamelistFilename(systemId))
	tmpPath, err := os.CreateTemp("", "gamelist-*.txt")
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		_, _ = tmpPath.WriteString(file + "\n")
	}
	_ = tmpPath.Sync()
	_ = tmpPath.Close()

	err = utils.MoveFile(tmpPath.Name(), gamelistPath)
	if err != nil {
		panic(err)
	}
	return gamelistPath
}

func filterUniqueWithMGL(files []string) []string {
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

func writeAmigaVisionLists(gamelistDir string, paths []string) {
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

func Run(args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	gamelistDir := fs.String("o", ".", "gamelist files directory")
	filter := fs.String("s", "all", "list of systems to index (comma separated)")
	progress := fs.Bool("p", false, "print output for dialog gauge")
	quiet := fs.Bool("q", false, "suppress all status output")
	detect := fs.Bool("d", false, "list active system folders")
	noDupes := fs.Bool("nodupes", false, "filter out duplicate games")
	overwrite := fs.Bool("overwrite", false, "overwrite existing gamelists (default false)")
	_ = fs.Parse(args)

	var systems []games.System
	if *filter == "all" {
		systems = games.AllSystems()
	} else {
		for _, filterId := range strings.Split(*filter, ",") {
			systemId := reverseId(filterId)

			if system, ok := games.Systems[systemId]; ok {
				systems = append(systems, system)
				continue
			}

			system, err := games.LookupSystem(systemId)
			if err != nil {
				continue
			}

			systems = append(systems, *system)
		}
	}

	if *detect {
		results := games.GetActiveSystemPaths(&config.UserConfig{}, systems)
		for _, r := range results {
			fmt.Printf("%s:%s\n", strings.ToLower(samId(r.System.Id)), r.Path)
		}
		os.Exit(0)
	}

	start := time.Now()
	totalGames := 0
	var generatedLists []string

	for _, system := range systems {
		glPath := filepath.Join(*gamelistDir, gamelistFilename(system.Id))

		// ✅ Skip scanning if gamelist exists and overwrite=false
		if _, err := os.Stat(glPath); err == nil && !*overwrite {
			if !*quiet {
				fmt.Printf("Skipping %s (list already exists: %s)\n", system.Id, glPath)
			}
			generatedLists = append(generatedLists, glPath)
			continue
		}

		// Scan and build gamelist
		systemPaths := games.GetSystemPaths(&config.UserConfig{}, []games.System{system})
		var systemFiles []string
		for _, p := range systemPaths {
			files, err := games.GetFiles(system.Id, p.Path)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				continue
			}
			systemFiles = append(systemFiles, files...)
		}

		if *noDupes {
			systemFiles = filterUniqueWithMGL(systemFiles)
		}

		if len(systemFiles) > 0 {
			totalGames += len(systemFiles)
			path := writeGamelist(*gamelistDir, system.Id, systemFiles)
			generatedLists = append(generatedLists, path)
		}

		if strings.EqualFold(system.Id, "Amiga") {
			writeAmigaVisionLists(*gamelistDir, systemPaths[0:1]) // reuse Amiga paths
		}
	}

	// ✅ Copy all lists into /tmp/.SAM_List (clean first)
	tmpDir := "/tmp/.SAM_List"
	_ = os.RemoveAll(tmpDir)
	_ = os.MkdirAll(tmpDir, 0755)
	for _, listFile := range generatedLists {
		dest := filepath.Join(tmpDir, filepath.Base(listFile))
		_ = utils.CopyFile(listFile, dest)
	}

	if !*quiet {
		taken := int(time.Since(start).Seconds())
		fmt.Printf("Indexing complete (%d games in %ds)\n", totalGames, taken)
	}

	if totalGames == 0 {
		os.Exit(8)
	}
}
