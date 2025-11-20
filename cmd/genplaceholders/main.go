package main

import (
	"fmt"
	"os"

	"chosenoffset.com/outpost9/placeholders"
)

func main() {
	fmt.Println("Outpost-9 Placeholder Graphics Generator")
	fmt.Println("=========================================")
	fmt.Println()

	if err := placeholders.GenerateAndSave(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("Done! Placeholder graphics are ready to use.")
	fmt.Println("Run the game to see your placeholders in action!")
}
