package main

import (
	"flag"
	"fmt"
	"os"

	"chosenoffset.com/outpost9/internal/placeholders"
)

func main() {
	// Parse command line flags
	gameDir := flag.String("game", "data/Example", "Game directory to generate assets for")
	flag.Parse()

	fmt.Println("Outpost-9 Placeholder Graphics Generator")
	fmt.Println("=========================================")
	fmt.Printf("Game directory: %s\n", *gameDir)
	fmt.Println()

	if err := placeholders.GenerateAndSave(*gameDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("Done! Placeholder graphics are ready to use.")
	fmt.Println("Run the game to see your placeholders in action!")
}
