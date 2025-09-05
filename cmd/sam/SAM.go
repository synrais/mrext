package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wizzomafizzo/mrext/pkg/config"
	"github.com/wizzomafizzo/mrext/pkg/framebuffer"
	"github.com/wizzomafizzo/mrext/pkg/games"
	"github.com/wizzomafizzo/mrext/pkg/input/gamepad"
	"github.com/wizzomafizzo/mrext/pkg/input/keyboard"
	"github.com/wizzomafizzo/mrext/pkg/curses/listpicker"
	"github.com/wizzomafizzo/mrext/pkg/curses/onscreenkeyboard"
	"github.com/wizzomafizzo/mrext/pkg/mister"
	"github.com/wizzomafizzo/mrext/pkg/service"
	"github.com/wizzomafizzo/mrext/pkg/sqlindex"
)

var (
	log = service.NewLogger("sam")
)

func main() {
	delayFlag := flag.Int("delay", 30, "seconds between games")
	randomFlag := flag.Bool("random", false, "random order")
	cycleAllFlag := flag.Bool("cycle-all", false, "cycle through all systems")
	overlayFlag := flag.Bool("overlay", true, "show on-screen overlay")
	excludeFlag := flag.String("exclude", "", "comma-separated list of system IDs to skip")
	flag.Parse()

	cfg, err := config.LoadUserConfig("mrext", &config.UserConfig{})
	if err != nil {
		log.Error("Failed to load config: %s", err)
		os.Exit(1)
	}

	exclude := map[string]bool{}
	if *excludeFlag != "" {
		for _, id := range strings.Split(*excludeFlag, ",") {
			exclude[strings.TrimSpace(id)] = true
		}
	}

	svc := &service.Service{
		Name: "sam",
		Main: func() {
			runSAM(cfg, *delayFlag, *randomFlag, *cycleAllFlag, *overlayFlag, exclude)
		},
	}
	svc.Run()
}

func runSAM(cfg *config.UserConfig, delay int, random, cycleAll, showOverlay bool, exclude map[string]bool) {
	rand.Seed(time.Now().UnixNano())
	log.Info("Attract Mode starting: delay=%ds random=%v cycle_all=%v overlay=%v", delay, random, cycleAll, showOverlay)

	// === GAME SCAN ===
	systems := games.AllSystems()
	gameLists := make(map[string][]string)
	for _, sys := range systems {
		if exclude[sys.Id] {
			continue
		}
		folders := games.GetSystemPaths(cfg, []games.System{sys})
		var sysFiles []string
		for _, folder := range folders {
			files, _ := games.GetFiles(sys.Id, folder.Path)
			sysFiles = append(sysFiles, files...)
		}
		sysFiles = games.FilterUniqueFilenames(sysFiles)
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
	if showOverlay {
		fb.Open()
		defer fb.Close()
	}

	// === INDEXING ===
	flat := [][2]string{}
	for sys, files := range gameLists {
		for _, f := range files {
			flat = append(flat, [2]string{sys, f})
		}
	}
	sqlindex.Generate(flat, func(_ int) {})

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

			if showOverlay {
				fb.Fill(framebuffer.RGB{0, 0, 0})
				fb.DrawString(20, 20, overlayText)
			}

			log.Info("Launching %s <%s>", sys, game)
			if err := mister.LaunchGenericFile(cfg, game); err != nil {
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
						log.Info("Skip requested (START)")
						goto nextGame
					}
					if e.Button == "SELECT" && e.Pressed {
						log.Info("Exit requested (SELECT)")
						return
					}
					if e.Button == "DPAD_RIGHT" && e.Pressed {
						log.Info("Next game requested (RIGHT)")
						idx = (idx + 1) % len(files)
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
						log.Info("Exit requested (q)")
						return
					}
					if key == "n" {
						log.Info("Next game requested (n)")
						goto nextGame
					}
					if key == "p" {
						log.Info("Previous game requested (p)")
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
	query := onscreenkeyboard.Run("Search games:")
	if query == "" {
		return
	}
	results := sqlindex.SearchGames(query)
	if len(results) == 0 {
		log.Info("No results for %q", query)
		return
	}
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
