package main

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

func reset(targetDir, sourceStubFile, destStubFile string) {
	log.Printf("Starting generator reset for directory: %s", targetDir)

	// 1. Ensure target directory exists
	log.Printf("Ensuring directory '%s' exists...", targetDir)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		log.Fatalf("Failed to create directory '%s': %v", targetDir, err)
	}

	// 2. Clean the target directory
	log.Printf("Cleaning directory '%s'...", targetDir)
	dirEntries, err := ioutil.ReadDir(targetDir)
	if err != nil {
		log.Fatalf("Failed to read directory '%s': %v", targetDir, err)
	}

	for _, entry := range dirEntries {
		// We only want to remove files, not subdirectories (though unlikely here)
		if !entry.IsDir() {
			filePath := filepath.Join(targetDir, entry.Name())
			log.Printf("  Removing file: %s", filePath)
			if err := os.Remove(filePath); err != nil {
				log.Printf("  Warning: Failed to remove file '%s': %v", filePath, err)
				// Decide if you want to stop on error or just warn
				// log.Fatalf("Stopping due to error removing file: %v", err)
			}
		}
	}
	log.Printf("Directory '%s' cleaned.", targetDir)

	// 3. Copy the reset stub file content
	destPath := filepath.Join(targetDir, destStubFile)
	log.Printf("Copying '%s' to '%s'...", sourceStubFile, destPath)

	// Check if source exists first
	if _, err := os.Stat(sourceStubFile); os.IsNotExist(err) {
		log.Fatalf("Source stub file '%s' does not exist.", sourceStubFile)
	}

	// Read the source content
	content, err := ioutil.ReadFile(sourceStubFile)
	if err != nil {
		log.Fatalf("Failed to read source stub file '%s': %v", sourceStubFile, err)
	}

	// Write the content to the destination
	// Use 0644 permissions for standard Go source files
	err = ioutil.WriteFile(destPath, content, 0644)
	if err != nil {
		log.Fatalf("Failed to write destination stub file '%s': %v", destPath, err)
	}

	log.Printf("Successfully reset stub file to '%s'.", destPath)
	log.Println("Generator reset complete.")
}
