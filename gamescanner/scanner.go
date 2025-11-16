package gamescanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GameEntry represents a discoverable game in the data directory
type GameEntry struct {
	Name          string   // Display name (directory name)
	Dir           string   // Directory path relative to data/
	RoomLibraries []string // List of room library files (for procedural generation)
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

		// Scan for room library files in this directory
		gamePath := filepath.Join(dataPath, dirName)
		roomLibraries, err := scanRoomLibraries(gamePath)
		if err != nil {
			// Skip directories that can't be read
			continue
		}

		// Only include directories with at least one room library
		if len(roomLibraries) > 0 {
			games = append(games, GameEntry{
				Name:          dirName,
				Dir:           dirName,
				RoomLibraries: roomLibraries,
			})
		}
	}

	return games, nil
}

// scanRoomLibraries finds all room library files in a game directory
func scanRoomLibraries(gamePath string) ([]string, error) {
	entries, err := os.ReadDir(gamePath)
	if err != nil {
		return nil, err
	}

	var roomLibraries []string
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

			// Look for room library files (files with "room" in the name)
			if strings.Contains(strings.ToLower(name), "room") {
				roomLibraries = append(roomLibraries, name)
			}
		}
	}

	return roomLibraries, nil
}
