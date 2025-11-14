package gamescanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GameEntry represents a discoverable game in the data directory
type GameEntry struct {
	Name   string   // Display name (directory name)
	Dir    string   // Directory path relative to data/
	Levels []string // List of available level files
}

// ScanDataDirectory scans the data directory for available games
// Returns a list of GameEntry objects, one for each valid game directory
func ScanDataDirectory(dataPath string) ([]GameEntry, error) {
	entries, err := os.ReadDir(dataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read data directory: %w", err)
	}

	var games []GameEntry

	for _, entry := range entries {
		// Skip non-directories
		if !entry.IsDir() {
			continue
		}

		// Skip special directories
		dirName := entry.Name()
		if dirName == "atlases" || strings.HasPrefix(dirName, ".") {
			continue
		}

		// Scan for level files in this directory
		gamePath := filepath.Join(dataPath, dirName)
		levels, err := scanLevels(gamePath)
		if err != nil {
			// Skip directories that can't be read
			continue
		}

		// Only include directories with at least one level file
		if len(levels) > 0 {
			games = append(games, GameEntry{
				Name:   dirName,
				Dir:    dirName,
				Levels: levels,
			})
		}
	}

	return games, nil
}

// scanLevels finds all .json level files in a game directory
func scanLevels(gamePath string) ([]string, error) {
	entries, err := os.ReadDir(gamePath)
	if err != nil {
		return nil, err
	}

	var levels []string
	for _, entry := range entries {
		// Skip directories
		if entry.IsDir() {
			continue
		}

		// Check if this is a JSON file
		name := entry.Name()
		if strings.HasSuffix(strings.ToLower(name), ".json") {
			// Skip atlas files
			if name == "atlas.json" {
				continue
			}
			levels = append(levels, name)
		}
	}

	return levels, nil
}
