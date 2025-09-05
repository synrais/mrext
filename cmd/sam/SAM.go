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
	"github.com/wizzomafizzo/mrext/pkg/input"
	"github.com/wizzomafizzo/mrext/pkg/curses"
	"github.com/wizzomafizzo/mrext/pkg/mister"
	"github.com/wizzomafizzo/mrext/pkg/service"
	"github.com/wizzomafizzo/mrext/pkg/sqlindex"
)

var (
	log *service.Logger
)

func main() {
	delayFlag := flag.Int("delay", 30, "seconds between games")
	randomFlag := flag.Bool("random", false, "random order")
	cycleAllFlag := flag.Bool("cycle-all", false, "cycle through all systems")
	flag.Parse()

	log = service.NewLogger("sam")

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

func runSAM(cfg *config.UserConfig, delay int, random, cycleAll bool) {
	rand.Seed(time.Now().UnixNano())
	log.Info("Attract Mode starting: delay=%ds random=%v cycle_all=%v", delay, random, cycleAll)

	// === GAME SCAN ===
	systems := games.AllSystems()
	exclude := make(map[string]bool)

	if cfg.Exclude != nil {
		for _, id := range cfg.Exclude {
			exclude[strings.TrimSpace(id)] = true
		}
	}

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
	if cfg.ShowOverlay {
		fb.Open()
		defer fb.Close()
	}

	// === INDEXING ===
	var flat [][2]string
	for sys, files := range gameLists {
		for _, f := range files {
			flat = append(flat, [2]string{sys, f})
		}
	}
	sqlindex.Generate(flat, nil)

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

			if cfg.ShowOverlay {
				fb.Fill(framebuffer.Color{0, 0, 0})
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

				// poll gamepads
				events, _ := input.ReadAll()
				for _, e := range events {
					if e.Pressed {
						switch e.Button {
						case "START":
							log.Info("Skip requested (START)")
							goto nextGame
						case "SELECT":
							log.Info("Exit requested (SELECT)")
							return
						case "DPAD_RIGHT":
							log.Info("Next game requested (RIGHT)")
							idx++
							if idx >= len(files) {
								idx = 0
							}
							goto nextGame
						case "DPAD_LEFT":
							log.Info("Previous game requested (LEFT)")
							idx--
							if idx < 0 {
								idx = len(files) - 1
							}
							goto nextGame
						case "Y":
							runSearchUI(cfg)
						}
					}
				}

				// poll keyboard
				if key := input.ReadKey(); key != "" {
					switch key {
					case "q":
						log.Info("Exit requested (q)")
						return
					case "n":
						log.Info("Next game requested (n)")
						goto nextGame
					case "p":
						log.Info("Previous game requested (p)")
						idx--
						if idx < 0 {
							idx = len(files) - 1
						}
						goto nextGame
					case "/":
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
	query := curses.OnscreenKeyboard("Search games:")
	if query == "" {
		return
	}
	results, _ := sqlindex.SearchGames(query)
	if len(results) == 0 {
		log.Info("No results for %q", query)
		return
	}
	var labels []string
	for _, r := range results {
		labels = append(labels, fmt.Sprintf("[%s] %s", r.System, r.Name))
	}
	choice := curses.ListPicker("Search Results", labels)
	if choice < 0 || choice >= len(results) {
		return
	}
	selected := results[choice]
	log.Info("Launching from search: %s <%s>", selected.System, selected.Path)
	_ = mister.LaunchGenericFile(cfg, selected.Path)
}
