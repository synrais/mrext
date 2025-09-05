package main

import (
    "bufio"
    "flag"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strings"

    "github.com/rivo/tview"
)

type Game struct {
    Title string
    Path  string
}

func loadSystems() map[string][]Game {
    systems := make(map[string][]Game)
    root := "/media/fat/Scripts/.MiSTer_SAM/SAM_Gamelists"

    files, _ := filepath.Glob(filepath.Join(root, "*.gamelist.txt"))
    for _, f := range files {
        sys := strings.TrimSuffix(filepath.Base(f), ".gamelist.txt")

        fh, _ := os.Open(f)
        scanner := bufio.NewScanner(fh)
        for scanner.Scan() {
            path := scanner.Text()
            title := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
            systems[sys] = append(systems[sys], Game{Title: title, Path: path})
        }
        fh.Close()
    }
    return systems
}

func runInteractive(systems map[string][]Game) {
    app := tview.NewApplication()
    sysList := tview.NewList()
    gameList := tview.NewList()

    // populate system list
    for sys := range systems {
        sysName := sys
        sysList.AddItem(sysName, "", 0, func() {
            // switch to game list
            gameList.Clear()
            for _, g := range systems[sysName] {
                game := g
                gameList.AddItem(game.Title, "", 0, func() {
                    exec.Command(
                        "/media/fat/Scripts/.MiSTer_SAM/samindex",
                        "-run", game.Path).Run()
                    app.Stop()
                })
            }
            app.SetRoot(gameList, true).SetFocus(gameList)
        })
    }

    // back key to return from games to system list
    gameList.SetDoneFunc(func() {
        app.SetRoot(sysList, true).SetFocus(sysList)
    })

    // also allow exiting from system list with Esc
    sysList.SetDoneFunc(func() {
        app.Stop()
    })

    // log to file
    logFile, _ := os.OpenFile("/tmp/mglmenu.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
    defer logFile.Close()
    fmt.Fprintln(logFile, "mglmenu started")

    if err := app.SetRoot(sysList, true).Run(); err != nil {
        fmt.Fprintln(logFile, "error:", err)
        panic(err)
    }
}

func runDebug(systems map[string][]Game) {
    for sys, games := range systems {
        fmt.Println("System:", sys)
        for _, g := range games {
            fmt.Println("   ", g.Title, "=>", g.Path)
        }
    }
}

func main() {
    debug := flag.Bool("debug", false, "run in debug (non-TUI) mode")
    flag.Parse()

    systems := loadSystems()

    if *debug {
        runDebug(systems)
    } else {
        runInteractive(systems)
    }
}
