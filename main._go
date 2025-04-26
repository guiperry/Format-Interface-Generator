// project/main.go (or gen.go)
package main

//go:generate go run main.go // Tells 'go generate' to run this command

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"FormatModules/generator" // Assuming correct path
)

func main() {
	fmt.Println("Starting code generation...")

	yamlFile := "full_bmp.yml"
	outputDir := "fullbmp"
	packageName := filepath.Base(outputDir)

	// Ensure YAML exists
	if _, err := os.Stat(yamlFile); os.IsNotExist(err) {
		log.Fatalf("%s does not exist: %v", yamlFile, err)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory %s: %v", outputDir, err)
	}


	// 1. Generate code
	err := generator.GenerateCode(yamlFile, outputDir, packageName)
	if err != nil {
		log.Fatalf("Error generating code: %v", err)
		// No return needed after log.Fatalf
	}
	

	// 2. Generate test bmp file (Optional - keep if needed for other purposes)
	// err = generateTestBMP()
	// if err != nil {
	// 	log.Printf("Warning: Error generating separate test BMP file: %v", err)
	// } else {
	// 	log.Println("Successfully generated separate test BMP file (test.bmp).")
	// }

}

// Include generateTestBMP() if you still need it, otherwise remove it.
// func generateTestBMP() error { ... }
