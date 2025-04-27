// main.go
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"os/exec" // For running 'go list'
	"gopkg.in/yaml.v2" // For parsing YAML in test generation
	"bytes"
	"go/format" // For formatting generated code

	// Keep other necessary imports
	"FormatModules/generator"
)

const (
	sourceDir      = "sources" // Directory containing source YAML files
	formatsDir     = "formats" // Parent directory for generated format code
	configFileName = "formats.json"
)

// --- Configuration Struct (remains the same) ---
type FormatConfig struct {
	Name        string `json:"name"`
	YAMLFile    string `json:"yamlFile"`    // Original YAML input path (relative to project root, e.g., sources/bmp.yml)
	OutputDir   string `json:"outputDir"`   // Directory for generated code (relative to project root, e.g., formats/bmp)
	PackageName string `json:"packageName"` // Go package name
}

// --- Main Function ---
func main() {
	// Define flags
	bootstrap := flag.Bool("bootstrap", false, "Enable bootstrapping mode to select, validate, and configure formats from the 'sources' directory")
	configPath := flag.String("config", "", "Path to the formats configuration JSON file (default: formats.json)")

	flag.Parse()

	// Determine config path
	actualConfigPath := *configPath
	if actualConfigPath == "" {
		actualConfigPath = configFileName // Default to root directory
	}
	log.Printf("Using configuration file: %s", actualConfigPath)

	// --- Bootstrap Logic ---
	if *bootstrap {
		log.Println("--- Running Bootstrap ---")
		err := runBootstrap(actualConfigPath)
		if err != nil {
			log.Fatalf("Bootstrapping failed: %v", err)
		}
		log.Println("--- Bootstrap Complete ---")
		// Optionally ask to continue with generation or exit
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Bootstrap finished. Continue with code generation for configured formats? (y/N): ")
		response, _ := reader.ReadString('\n')
		if !strings.EqualFold(strings.TrimSpace(response), "y") {
			log.Println("Exiting after bootstrap.")
			os.Exit(0)
		}
	}

	// --- Normal Generation Logic ---
	log.Println("--- Running Code Generation ---")
	runGeneration(actualConfigPath)
	log.Println("--- Code Generation Complete ---")
}

// --- Bootstrap Function ---
func runBootstrap(configPath string) error {
	log.Printf("Scanning '%s' directory for source YAML files...", sourceDir)
	files, err := ioutil.ReadDir(sourceDir)
	if err != nil {
		return fmt.Errorf("failed to read source directory '%s': %w", sourceDir, err)
	}

	var yamlFiles []string
	for _, file := range files {
		if !file.IsDir() && (strings.HasSuffix(file.Name(), ".yml") || strings.HasSuffix(file.Name(), ".yaml")) {
			yamlFiles = append(yamlFiles, filepath.Join(sourceDir, file.Name()))
		}
	}
	sort.Strings(yamlFiles) // Sort for consistent display

	if len(yamlFiles) == 0 {
		log.Printf("No YAML files found in '%s'. Nothing to bootstrap.", sourceDir)
		return nil
	}

	// --- Format Selection based on YAML files ---
	selectedYamlFiles, err := showSourceFileSelection(yamlFiles)
	if err != nil {
		return fmt.Errorf("format selection failed: %w", err)
	}

	if len(selectedYamlFiles) == 0 {
		log.Println("No formats selected for bootstrapping.")
		return nil
	}

	log.Printf("Processing %d selected source file(s) for bootstrap...", len(selectedYamlFiles))

	// --- Load existing config ---
	currentConfigs, err := loadConfig(configPath)
	if err != nil {
		return err // Error handled in loadConfig
	}
	configMap := make(map[string]int) // Map source YAML path to index in currentConfigs
	for i, cfg := range currentConfigs {
		configMap[cfg.YAMLFile] = i
	}

	// --- Process selected files ---
	configUpdated := false
	for _, yamlFile := range selectedYamlFiles {
		log.Printf("--- Bootstrapping from: %s ---", yamlFile)

		// Derive names and paths
		baseName := strings.TrimSuffix(filepath.Base(yamlFile), filepath.Ext(yamlFile))
		outputDir := filepath.Join(formatsDir, strings.ToLower(baseName)) // e.g., formats/bmp
		packageName := strings.ToLower(baseName)
		formatName := strings.Title(packageName)

		// Validate and Reform YAML using the function from validator.go
		_, err := ValidateAndReformYAML(yamlFile, outputDir)
		if err != nil {
			log.Printf("ERROR: Failed to validate/reform %s: %v. Skipping configuration update.", yamlFile, err)
			continue // Skip this file if validation fails
		}

		// Update or Add configuration
		if index, exists := configMap[yamlFile]; exists {
			log.Printf("Format for '%s' already exists in configuration. Updating paths.", yamlFile)
			currentConfigs[index].OutputDir = outputDir
			currentConfigs[index].PackageName = packageName
			// Name and YAMLFile remain the same
			configUpdated = true
		} else {
			log.Printf("Adding new format '%s' to configuration.", formatName)
			newConfig := FormatConfig{
				Name:        formatName,
				YAMLFile:    yamlFile, // Store relative path from project root
				OutputDir:   outputDir, // Store relative path from project root
				PackageName: packageName,
			}
			currentConfigs = append(currentConfigs, newConfig)
			configUpdated = true
		}
	}

	// --- Save updated config if changes were made ---
	if configUpdated {
		err = saveConfig(configPath, currentConfigs)
		if err != nil {
			return err
		}
		log.Println("Configuration file updated.")
	} else {
		log.Println("No configuration changes needed.")
	}

	return nil
}

// --- Generation Function ---
func runGeneration(configPath string) {
	// --- Load Config ---
	formatConfigs, err := loadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	if len(formatConfigs) == 0 {
		log.Println("No formats configured in", configPath, ". Nothing to generate.")
		log.Println("Hint: Run with -bootstrap to configure formats from the 'sources' directory.")
		return
	}

	// --- Format Selection based on Config ---
	selectedConfigs, err := showConfigSelection(formatConfigs)
	if err != nil {
		log.Fatalf("Format selection failed: %v", err)
	}
	if len(selectedConfigs) == 0 {
		log.Println("No formats selected for generation.")
		return
	}

	log.Printf("Processing %d selected format(s) for generation.", len(selectedConfigs))

	// --- Get Go Module Path (Needed for test import) ---
	goModulePath, err := getGoModulePath()
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
		reset(config.OutputDir) // Reset cleans only .go files in the target dir
		log.Println("Reset complete.")

		// --- Determine Reformed YAML Path ---
		reformedYamlPath := filepath.Join(config.OutputDir, filepath.Base(config.YAMLFile))
		if _, err := os.Stat(reformedYamlPath); os.IsNotExist(err) {
			log.Printf("Warning: Reformed YAML %s not found. Attempting validation/reformation...", reformedYamlPath)
			reformedYamlPath, err = ValidateAndReformYAML(config.YAMLFile, config.OutputDir)
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

		err = generator.GenerateCode(reformedYamlPath, config.OutputDir, config.PackageName, "")
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

// --- Helper Functions ---

// loadConfig reads and parses the formats.json file.
func loadConfig(configPath string) ([]FormatConfig, error) {
	var configs []FormatConfig
	configData, err := ioutil.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("Configuration file '%s' not found. Returning empty configuration.", configPath)
			return configs, nil // Return empty slice, not an error
		}
		return nil, fmt.Errorf("failed to read configuration file '%s': %w", configPath, err)
	}

	err = json.Unmarshal(configData, &configs)
	if err != nil {
		return nil, fmt.Errorf("failed to parse configuration file '%s': %w", configPath, err)
	}
	log.Printf("Loaded %d format configurations from %s.", len(configs), configPath)
	return configs, nil
}

// saveConfig saves the format configurations back to formats.json.
func saveConfig(configPath string, configs []FormatConfig) error {
	// Sort configs by name for consistency before saving
	sort.Slice(configs, func(i, j int) bool {
		return configs[i].Name < configs[j].Name
	})

	updatedConfigData, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated configuration: %w", err)
	}
	err = ioutil.WriteFile(configPath, updatedConfigData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write updated configuration file '%s': %w", configPath, err)
	}
	return nil
}

// showSourceFileSelection prompts the user to select from available source YAML files.
func showSourceFileSelection(yamlFiles []string) ([]string, error) {
	if len(yamlFiles) == 0 {
		return []string{}, nil
	}
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\nAvailable source YAML files:")
	for i, file := range yamlFiles {
		fmt.Printf("%d. %s\n", i+1, file)
	}

	fmt.Print("\nSelect file(s) to bootstrap (e.g., 1,3,4), or press Enter for all: ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return yamlFiles, nil // Return all if no selection
	}

	var selectedFiles []string
	parts := strings.Split(input, ",")
	for _, part := range parts {
		trimmedPart := strings.TrimSpace(part)
		if trimmedPart == "" {
			continue
		}
		idx, err := strconv.Atoi(trimmedPart)
		if err != nil || idx < 1 || idx > len(yamlFiles) {
			return nil, fmt.Errorf("invalid selection '%s': please enter numbers between 1 and %d, separated by commas", trimmedPart, len(yamlFiles))
		}
		selectedFiles = append(selectedFiles, yamlFiles[idx-1])
	}

	// Remove duplicates if any (though unlikely with numeric input)
	uniqueFiles := make([]string, 0, len(selectedFiles))
	seen := make(map[string]bool)
	for _, file := range selectedFiles {
		if !seen[file] {
			uniqueFiles = append(uniqueFiles, file)
			seen[file] = true
		}
	}
	return uniqueFiles, nil
}

// showConfigSelection prompts the user to select from configured formats.
func showConfigSelection(formatConfigs []FormatConfig) ([]FormatConfig, error) {
	if len(formatConfigs) == 0 {
		return []FormatConfig{}, nil
	}
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\nConfigured formats:")
	for i, config := range formatConfigs {
		fmt.Printf("%d. %s (%s -> %s)\n", i+1, config.Name, config.YAMLFile, config.OutputDir)
	}

	fmt.Print("\nSelect format(s) to generate (e.g., 1,3,4), or press Enter for all: ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return formatConfigs, nil // Return all if no selection
	}

	var selectedConfigs []FormatConfig
	parts := strings.Split(input, ",")
	for _, part := range parts {
		trimmedPart := strings.TrimSpace(part)
		if trimmedPart == "" {
			continue
		}
		idx, err := strconv.Atoi(trimmedPart)
		if err != nil || idx < 1 || idx > len(formatConfigs) {
			return nil, fmt.Errorf("invalid selection '%s': please enter numbers between 1 and %d, separated by commas", trimmedPart, len(formatConfigs))
		}
		selectedConfigs = append(selectedConfigs, formatConfigs[idx-1])
	}
	// Remove duplicates
	uniqueConfigs := make([]FormatConfig, 0, len(selectedConfigs))
	seen := make(map[string]bool)
	for _, cfg := range selectedConfigs {
		if !seen[cfg.Name] { // Use Name as unique identifier
			uniqueConfigs = append(uniqueConfigs, cfg)
			seen[cfg.Name] = true
		}
	}
	return uniqueConfigs, nil
}
func generateTestScript(reformedYamlPath, outputDir, packageName, goModulePath string) error {
	// 1. Parse the reformed YAML to find struct names
	yamlData, err := ioutil.ReadFile(reformedYamlPath)
	if err != nil {
		return fmt.Errorf("failed to read reformed YAML %s: %w", reformedYamlPath, err)
	}

	// Use a temporary struct to get just the struct keys
	var tempFormat struct {
		Structs map[string]interface{} `yaml:"structs"`
	}
	err = yaml.Unmarshal(yamlData, &tempFormat)
	if err != nil {
		return fmt.Errorf("failed to parse structs from reformed YAML %s: %w", reformedYamlPath, err)
	}

	if len(tempFormat.Structs) == 0 {
		return fmt.Errorf("no structs found in reformed YAML %s, cannot generate test", reformedYamlPath)
	}

	// Get struct names and sort them alphabetically for consistency
	structNames := make([]string, 0, len(tempFormat.Structs))
	for name := range tempFormat.Structs {
		structNames = append(structNames, name)
	}
	sort.Strings(structNames)
	firstStructName := structNames[0] // Use the first one alphabetically

	// 2. Prepare data for the test template
	testData := generator.TestTemplateData{
		PackageName:     packageName,
		FormatDir:       filepath.Base(filepath.Dir(outputDir)), // e.g., "formats"
		FirstStructName: firstStructName,
		GoModulePath:    goModulePath,
	}

	// 3. Parse the test template
	tmpl, err := template.New("test").Parse(generator.TestFileTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse test template: %w", err)
	}

	// 4. Execute the template
	var output bytes.Buffer
	err = tmpl.Execute(&output, testData)
	if err != nil {
		return fmt.Errorf("failed to execute test template for %s: %w", packageName, err)
	}

	// 5. Format the generated code
	formattedOutput, errFmt := format.Source(output.Bytes())
	if errFmt != nil {
		log.Printf("Warning: Failed to format generated test code for %s: %v. Writing unformatted code.", packageName, errFmt)
		formattedOutput = output.Bytes() // Fallback
	}

	// 6. Write the test file
	testFilePath := filepath.Join(outputDir, fmt.Sprintf("%s_test.go", packageName))
	err = ioutil.WriteFile(testFilePath, formattedOutput, 0644)
	if err != nil {
		return fmt.Errorf("failed to write test file %s: %w", testFilePath, err)
	}

	return nil
}

// --- NEW: Helper to get Go Module Path ---
func getGoModulePath() (string, error) {
	cmd := exec.Command("go", "list", "-m")
	output, err := cmd.Output()
	if err != nil {
		// Try checking go.mod directly as a fallback
		modData, readErr := ioutil.ReadFile("go.mod")
		if readErr == nil {
			lines := strings.Split(string(modData), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "module ") {
					return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
				}
			}
		}
		// If both fail, return the original error
		return "", fmt.Errorf("failed to run 'go list -m' and couldn't parse go.mod: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}
