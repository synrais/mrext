package games

import (
	"os"
	"strings"
	"path/filepath"
)

type PathsCustomizer func(filePath string) ([]string, error)

var pathsCustomizers = map[string]PathsCustomizer{}

func RunPathsCustomizer(systemId, filePath string) ([]string, error, bool) {
	fn, ok := pathsCustomizers[systemId]
	if !ok {
		return nil, nil, false
	}
	results, err := fn(filePath)
	return results, err, true
}

func RegisterPathsCustomizer(systemId string, fn PathsCustomizer) {
	pathsCustomizers[systemId] = fn
}

func init() {
	RegisterPathsCustomizer("AmigaVision", customizeAmigaVision)
}

// customizeAmigaVision expands demos.txt / games.txt into pseudo-paths.
func customizeAmigaVision(txtPath string) ([]string, error) {
	data, err := os.ReadFile(txtPath)
	if err != nil {
		return nil, err
	}

	var results []string
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Pseudo-path: txt file plus entry
		results = append(results, txtPath+"#"+line)
	}
	return results, nil
}
