package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wizzomafizzo/mrext/pkg/arcadedb"
	"github.com/wizzomafizzo/mrext/pkg/config"
	"github.com/wizzomafizzo/mrext/pkg/dupes"
	"github.com/wizzomafizzo/mrext/pkg/framebuffer"
	"github.com/wizzomafizzo/mrext/pkg/games"
	"github.com/wizzomafizzo/mrext/pkg/gamepad"
	"github.com/wizzomafizzo/mrext/pkg/keyboard"
	"github.com/wizzomafizzo/mrext/pkg/listpicker"
	"github.com/wizzomafizzo/mrext/pkg/logging"
	"github.com/wizzomafizzo/mrext/pkg/mister"
	"github.com/wizzomafizzo/mrext/pkg/onscreenkeyboard"
	"github.com/wizzomafizzo/mrext/pkg/service"
	"github.com/wizzomafizzo/mrext/pkg/sqlindex"
)

var (
	log *logging.Logger
)

func main() {
	delayFlag := flag.Int("delay", 0, "seconds between games")
	randomFlag := flag.Bool("random", false, "random order")
	cycleAllFlag := flag.Bool("cycle-all", false, "cycle through all systems")
	flag.Parse()

	log = logging.NewLogger("sam")

	cfg, err := config.LoadUserConfig("mrext", &config.UserConfig{})
	if err != nil {
		log.Error("Failed to load config: %s", err)
		os.Exit(1)
	}

	svc := service.New("sam", func() {
		runSAM(cfg, *delayFlag, *randomFlag, *cycleAllFlag)
	})
	svc.Run()
}

func runSAM(cfg *config.UserConfig, delayOverride int, randomOverride, cycleOverride bool) {
	rand.Seed(time.Now().UnixNano())

	attract := cfg.Attract
	delay := attract.Delay
	if delayOverride > 0 {
		delay = delayOverride
	}
	random := attract.Random || randomOverride
	cycleAll := attract.CycleAll || cycleOverride

	log.Info("Attract Mode starting: delay=%ds random=%v cycle_all=%v", delay, random, cycleAll)

	// Load ArcadeDB metadata if enabled
	var arcadeMeta map[string]arcadedb.ArcadeDbEntry
	if attract.ArcadeUseMetadata {
		entries, _ := arcadedb.ReadArcadeDb()
		arcadeMeta = make(map[string]arcadedb.ArcadeDbEntry)
		for _, e := range entries {
			arcadeMeta[e.Name] = e
		}
	}

	// === GAME SCAN ===
	systems := games.GetAllSystems()
	exclude := map[string]bool{}
	for _, id := range attract.Exclude {
		exclude[strings.TrimSpace(id)] = true
	}

	gameLists := make(map[string][]string)
	for _, sys := range systems {
		if exclude[sys.Id] {
			continue
		}
		folders := games.GetSystemPaths(&cfg.Systems, []games.System{sys})
		var sysFiles []string
		for _, folder := range folders {
			files, _ := games.GetFiles(sys.Id, folder.Path)
			sysFiles = append(sysFiles, files...)
		}
		sysFiles = dupes.FilterUniqueFilenames(sysFiles)
		if len(sysFiles) > 0 {
			gameLists[sys.Id] = sysFiles
			log.Info("System %s: %d games", sys.Id, len(sysFiles))
		}
	}
	if len(gameLists) == 0 {
		log.Warn("No games found")
		return
	}

	// === OVERLAY ===
	var fb framebuffer.Framebuffer
	if attract.ShowOverlay {
		fb.Open()
		defer fb.Close()
	}

	// === INDEXING (for search) ===
	sqlindex.Generate(gameLists) // build or refresh index

	// === MAIN LOOP ===
	for sys, files := range gameLists {
		if len(files) == 0 {
			continue
		}

		idx := 0
		for {
			if random {
				idx = rand.Intn(len(files))
			}
			if idx >= len(files) {
				break
			}

			game := files[idx]
			name := strings.TrimSuffix(filepath.Base(game), filepath.Ext(game))
			overlayText := fmt.Sprintf("Now Playing: %s [%s]", name, sys)

			if attract.ArcadeUseMetadata && sys.Id == "arcade" {
				if meta, ok := arcadeMeta[name]; ok {
					overlayText = fmt.Sprintf("%s\n%s (%d) - %s",
						meta.Name, meta.Manufacturer, meta.Year, meta.Category)
				}
			}

			if attract.ShowOverlay {
				fb.Fill(framebuffer.Color{0, 0, 0})
				fb.DrawString(20, 20, overlayText)
			}

			log.Info("Launching %s <%s>", sys, game)
			err := mister.LaunchGenericFile(cfg, game)
			if err != nil {
				log.Error("Launch failed: %s", err)
			}

			// countdown with input check
			start := time.Now()
			for {
				if time.Since(start) > time.Duration(delay)*time.Second {
					break
				}

				// poll gamepad
				events, _ := gamepad.ReadAll()
				for _, e := range events {
					if e.Button == "START" && e.Pressed {
						log.Info("Skip requested by gamepad (START)")
						goto nextGame
					}
					if e.Button == "SELECT" && e.Pressed {
						log.Info("Exit requested by gamepad (SELECT)")
						return
					}
					if e.Button == "DPAD_RIGHT" && e.Pressed {
						log.Info("Next game requested (RIGHT)")
						idx++
						if idx >= len(files) {
							idx = 0
						}
						goto nextGame
					}
					if e.Button == "DPAD_LEFT" && e.Pressed {
						log.Info("Previous game requested (LEFT)")
						idx--
						if idx < 0 {
							idx = len(files) - 1
						}
						goto nextGame
					}
					if e.Button == "Y" && e.Pressed {
						runSearchUI(cfg)
					}
				}

				// poll keyboard
				if key := keyboard.ReadKey(); key != "" {
					if key == "q" {
						log.Info("Exit requested by keyboard (q)")
						return
					}
					if key == "n" {
						log.Info("Next game requested by keyboard (n)")
						goto nextGame
					}
					if key == "p" {
						log.Info("Previous game requested by keyboard (p)")
						idx--
						if idx < 0 {
							idx = len(files) - 1
						}
						goto nextGame
					}
					if key == "/" {
						runSearchUI(cfg)
					}
				}

				time.Sleep(250 * time.Millisecond)
			}

		nextGame:
			if !random {
				idx++
			}
			if !cycleAll {
				break
			}
		}
	}
}

func runSearchUI(cfg *config.UserConfig) {
	// On-screen keyboard
	query := onscreenkeyboard.Run("Search games:")
	if query == "" {
		return
	}

	// Run search in SQL index
	results := sqlindex.SearchGames(query)
	if len(results) == 0 {
		log.Info("No results for %q", query)
		return
	}

	// Build labels for picker
	var labels []string
	for _, r := range results {
		labels = append(labels, fmt.Sprintf("[%s] %s", r.System, r.Name))
	}

	choice := listpicker.Run("Search Results", labels)
	if choice < 0 || choice >= len(results) {
		return
	}

	selected := results[choice]
	log.Info("Launching from search: %s <%s>", selected.System, selected.Path)
	_ = mister.LaunchGenericFile(cfg, selected.Path)
}
