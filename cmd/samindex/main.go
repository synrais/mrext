package main

import (
    "flag"
    "fmt"
    "os"

    "github.com/wizzomafizzo/mrext/pkg/gameindex"
)

func main() {
    gamelistDir := flag.String("o", ".", "gamelist files directory")
    // ... parse other flags ...
    flag.Parse()

    if *runPath != "" {
        if err := gameindex.Run(*runPath); err != nil {
            fmt.Fprintln(os.Stderr, err)
            os.Exit(1)
        }
        os.Exit(0)
    }

    total := gameindex.CreateGamelists(*gamelistDir, systemPathsMap, *progress, *quiet, *noDupes)
    if total == 0 {
        os.Exit(8)
    }
}
