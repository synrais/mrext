package main

import (
    "bufio"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strings"

    "github.com/rivo/tview"
    "github.com/gdamore/tcell/v2"
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

func main() {
    systems := loadSystems()

    // Force tcell to use stdin/stdout instead of /dev/tty
    screen, err := tcell.NewTerminfoScreen()
    if err != nil {
        fmt.Println("failed to create screen:", err)
        os.Exit(1)
    }
    if err = screen.Init(); err != nil {
        fmt.Println("screen init failed:", err)
        os.Exit(1)
    }
    defer screen.Fini()

    app := tview.NewApplication()
    app.SetScreen(screen)

    sysList := tview.NewList()
    gameList := tview.NewList()

    for sys := range systems {
        sysName := sys
        sysList.AddItem(sysName, "", 0, func() {
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

    gameList.SetDoneFunc(func() {
        app.SetRoot(sysList, true).SetFocus(sysList)
    })
    sysList.SetDoneFunc(func() {
        app.Stop()
    })

    if err := app.SetRoot(sysList, true).Run(); err != nil {
        fmt.Println("error:", err)
        os.Exit(1)
    }
}
