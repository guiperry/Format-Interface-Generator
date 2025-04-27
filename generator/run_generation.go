package generator

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"FIG/config"
	"FIG/utils"
	"FIG/dialogue"
)
// --- Generation Function ---
func RunGeneration(configPath string) {
	// --- Load Config ---
	formatConfigs, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	if len(formatConfigs) == 0 {
		log.Println("No formats configured in", configPath, ". Nothing to generate.")
		log.Println("Hint: Run with -bootstrap to configure formats from the 'sources' directory.")
		return
	}

	// --- Format Selection based on Config ---
	selectedConfigs, err := dialogue.ShowConfigSelection(formatConfigs)
	if err != nil {
		log.Fatalf("Format selection failed: %v", err)
	}
	if len(selectedConfigs) == 0 {
		log.Println("No formats selected for generation.")
		return
	}

	log.Printf("Processing %d selected format(s) for generation.", len(selectedConfigs))

	// --- Get Go Module Path (Needed for test import) ---
	goModulePath, err := utils.GetGoModulePath()
	if err != nil {
		log.Printf("Warning: Could not determine Go module path: %v. Test imports might be incorrect.", err)
		// You could potentially ask the user for it here if needed
	}
	// --- End Get Go Module Path ---

	reader := bufio.NewReader(os.Stdin) // Reader for user input

	for _, config := range selectedConfigs {
		log.Printf("--- Processing format: %s ---", config.Name)

		// --- Reset Generated Go Files ---
		log.Printf("Running generator reset for %s...", config.OutputDir)
		utils.Reset(config.OutputDir) // Reset cleans only .go files in the target dir
		log.Println("Reset complete.")

		// --- Determine Reformed YAML Path ---
		reformedYamlPath := filepath.Join(config.OutputDir, filepath.Base(config.YAMLFile))
		if _, err := os.Stat(reformedYamlPath); os.IsNotExist(err) {
			log.Printf("Warning: Reformed YAML %s not found. Attempting validation/reformation...", reformedYamlPath)
			reformedYamlPath, err = utils.ValidateAndReformYAML(config.YAMLFile, config.OutputDir)
			if err != nil {
				log.Printf("ERROR: On-the-fly validation/reformation failed for %s: %v. Skipping generation.", config.Name, err)
				continue
			}
		} else {
			log.Printf("Using reformed YAML: %s", reformedYamlPath)
		}

		// --- Generate Code ---
		log.Println("Starting code generation...")
		if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
			log.Printf("ERROR: Failed to ensure output directory %s exists: %v. Skipping generation.", config.OutputDir, err)
			continue
		}

		err = GenerateCode(reformedYamlPath, config.OutputDir, config.PackageName, "")
		if err != nil {
			log.Printf("ERROR: %s: Error generating code: %v", config.Name, err)
			continue // Skip test generation if code generation failed
		}
		log.Printf("%s: Code generation completed successfully.", config.Name)

		// --- Ask to Generate Test Script ---
		fmt.Printf("Generate basic test script for %s? (y/N): ", config.Name)
		response, _ := reader.ReadString('\n')
		if strings.EqualFold(strings.TrimSpace(response), "y") {
			log.Printf("Generating test script for %s...", config.Name)
			err := generateTestScript(reformedYamlPath, config.OutputDir, config.PackageName, goModulePath)
			if err != nil {
				log.Printf("ERROR: Failed to generate test script for %s: %v", config.Name, err)
			} else {
				log.Printf("Successfully generated test script: %s", filepath.Join(config.OutputDir, config.PackageName+"_test.go"))
			}
		}
		// --- End Test Script Generation ---

		fmt.Println("---") // Separator between formats
	}

	log.Println("Selected format(s) processed.")
}
