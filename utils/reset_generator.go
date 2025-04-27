// reset_generator.go
package utils

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// reset cleans the target directory of generated .go files.
func Reset(targetDir string) { // Simplified parameters
	log.Printf("Starting generator reset for directory: %s", targetDir)

	// --- Stub renaming logic removed ---

	// 1. Ensure target directory exists
	log.Printf("Ensuring directory '%s' exists...", targetDir)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		log.Fatalf("Failed to create directory '%s': %v", targetDir, err)
	}

	// 2. Clean generated .go files from the target directory
	log.Printf("Cleaning generated Go files in directory '%s'...", targetDir)
	dirEntries, err := ioutil.ReadDir(targetDir)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("Directory '%s' does not exist yet, nothing to clean.", targetDir)
			log.Println("Reset step complete (directory created).")
			return
		}
		log.Fatalf("Failed to read directory '%s': %v", targetDir, err)
	}

	filesRemoved := 0
	for _, entry := range dirEntries {
		entryName := entry.Name()
		// Only remove .go files (don't touch the reformed .yml file)
		if !entry.IsDir() && strings.HasSuffix(entryName, ".go") {
			filePath := filepath.Join(targetDir, entryName)
			log.Printf("  Removing generated file: %s", filePath)
			if err := os.Remove(filePath); err != nil {
				log.Printf("  Warning: Failed to remove file '%s': %v", filePath, err)
			} else {
				filesRemoved++
			}
		}
	}
	log.Printf("Removed %d generated Go file(s) from '%s'.", filesRemoved, targetDir)

	log.Println("Reset step complete.")
}
