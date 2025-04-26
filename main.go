// project/main.go (or gen.go)
package main

//go:generate go run main.go // Tells 'go generate' to run this command

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"FormatModules/generator" // Assuming correct path
)

func main() {
	// Define flags
	yamlFile := flag.String("yaml", "full_bmp.yml", "Input YAML definition file")
	outputDir := flag.String("output", "fullbmp", "Output directory for generated code")
	resetStubSource := flag.String("reset-stub", "fullbmp_stubs_source.go", "Source stub file for reset")
	targetStubName := flag.String("target-stub", "fullbmp_stubs.go", "Name of the stub file in the output directory")

	flag.Parse() // Parse command-line arguments

	// Derive package name from output directory
	pkgName := filepath.Base(*outputDir)

	// Call reset() first to clean up and prepare stubs
	fmt.Println("Running generator reset...")
	reset(*outputDir, *resetStubSource, *targetStubName) // Pass flag values to reset
	fmt.Println("Reset complete.")

	fmt.Println("Starting code generation...")

	// Ensure YAML exists
	if _, err := os.Stat(*yamlFile); os.IsNotExist(err) {
		log.Fatalf("%s does not exist: %v", *yamlFile, err)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory %s: %v", *outputDir, err)
	}


	// 1. Generate code
	err := generator.GenerateCode(*yamlFile, *outputDir, pkgName, *targetStubName)
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
