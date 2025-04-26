// reset_generator.go (or wherever reset is defined)
package main

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings" // Import strings
)

// reset cleans the target directory, preserving specified stub and test files.
func reset(targetDir, stubFilePath, testFilePath string) { // Changed parameters
	log.Printf("Starting generator reset for directory: %s", targetDir)

	// First, rename any _stubs._go files back to _stubs.go
	if strings.HasSuffix(stubFilePath, "_stubs.go") {
		renamedPath := strings.TrimSuffix(stubFilePath, ".go") + "._go"
		if _, err := os.Stat(renamedPath); err == nil {
			if err := os.Rename(renamedPath, stubFilePath); err != nil {
				log.Printf("Warning: Failed to rename stub file %s back to %s: %v", renamedPath, stubFilePath, err)
			} else {
				log.Printf("Renamed stub file back to original name: %s", stubFilePath)
			}
		}
	}

	log.Printf(" -> Preserving stub file: %s", stubFilePath)
	log.Printf(" -> Preserving test file: %s", testFilePath)

	// 1. Ensure target directory exists
	log.Printf("Ensuring directory '%s' exists...", targetDir)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		log.Fatalf("Failed to create directory '%s': %v", targetDir, err)
	}

	// 2. Clean the target directory, preserving specific files
	log.Printf("Cleaning generated files in directory '%s'...", targetDir)
	dirEntries, err := ioutil.ReadDir(targetDir)
	if err != nil {
		// If dir doesn't exist yet (first run), that's okay
		if os.IsNotExist(err) {
			log.Printf("Directory '%s' does not exist yet, nothing to clean.", targetDir)
			log.Println("Reset step complete (directory created).")
			return // Nothing else to do
		}
		log.Fatalf("Failed to read directory '%s': %v", targetDir, err)
	}

	// Get just the base names of the files to keep
	stubFileName := filepath.Base(stubFilePath)
	testFileName := filepath.Base(testFilePath)

	filesRemoved := 0
	for _, entry := range dirEntries {
		entryName := entry.Name()
		// Only consider .go files that are not the stub or test file
		if !entry.IsDir() && strings.HasSuffix(entryName, ".go") && entryName != stubFileName && entryName != testFileName {
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

	// 3. No copying needed anymore

	log.Println("Reset step complete.")
}
