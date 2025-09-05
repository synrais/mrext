package main

import (
    "bufio"
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

func main() {
    app := tview.NewApplication()
    systems := loadSystems()

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

    if err := app.SetRoot(sysList, true).Run(); err != nil {
        panic(err)
    }
}
