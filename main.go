// main.go
package main

//go:generate go run . // Simple generate command assuming all needed .go files are in the main package

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"FormatModules/application_structs"
	"FormatModules/generator"

	"gopkg.in/yaml.v2"
)

// --- Configuration Struct (Removed ResetStubSource) ---
type FormatConfig struct {
	Name        string `json:"name"`
	YAMLFile    string `json:"yamlFile"`
	OutputDir   string `json:"outputDir"`
	PackageName string `json:"packageName"`
	// ResetStubSource string `json:"resetStubSource"` // Removed
	TargetStub string `json:"targetStub"` // Path to the stub file (e.g., "jpg/jpg_stubs.go")
	TestFile   string `json:"testFile"`   // Path to the test file (e.g., "jpg/jpg_test.go")
}

// --- Templates (remain the same) ---
const resetStubTemplate = `// {{.FileName}} - Stub file
// IMPORTANT: This file contains stub definitions for compile-time checks.
//            It should NOT be removed by the generator.
package {{.PackageName}}

import "io"

{{range .Structs}}
// --- Stub for {{.Name}} ---
type {{.Name}} struct {
    {{range .Fields}}
    {{.Name}} {{.Type}} // {{.Description}}
    {{end}}
}

// Dummy Read method
func (s *{{.Name}}) Read(r io.Reader, ctx interface{}) error {
	return nil
}

// Dummy Write method
func (s *{{.Name}}) Write(w io.Writer) error {
	return nil
}
{{println}}
{{end}}
`

const testFileTemplate = `// {{.FileName}} - Test file for {{.PackageName}}
package {{.PackageName}}_test // <-- FIX: Use _test package convention

import (
	"testing"
	// "bytes"
	// "log"
	// "os"
	// "reflect"

	
)

// Test{{.FormatName}}GeneratedCode tests the generated code for the {{.FormatName}} format.
func Test{{.FormatName}}GeneratedCode(t *testing.T) {
	// TODO: Implement test logic for {{.FormatName}}
	t.Logf("Test for {{.FormatName}} not fully implemented yet.")
	t.Skip("Test logic for {{.FormatName}} needs implementation.")
}
`

// --- Template Data Structs (remain the same) ---
type TemplateData struct {
	FileName    string
	FormatName  string
	PackageName string
	Structs     []StructTemplateData
}
type StructTemplateData struct {
	Name   string
	Fields []application_structs.Field
}

// --- Main Function (remains the same) ---
func main() {
	// Define flags
	bootstrap := flag.Bool("bootstrap", false, "Enable bootstrapping mode to add a new format based on a YAML file")
	yamlFile := flag.String("yaml", "", "Input YAML definition file (used only with -bootstrap)")
	configPath := flag.String("config", "", "Path to the formats configuration JSON file (default: checks config/formats.json first, then formats.json)")

	flag.Parse()

	proceedWithGeneration := false

	// --- Bootstrap Logic ---
	if *bootstrap {
		if *yamlFile == "" {
			log.Fatal("Missing required flag for bootstrapping: -yaml <filename.yml>")
		}
		log.Println("--- Running Bootstrap ---")
		// First try config/formats.json, fall back to formats.json if not specified
		actualConfigPath := *configPath
		if actualConfigPath == "" {
			actualConfigPath = "config/formats.json"
			if _, err := os.Stat(actualConfigPath); os.IsNotExist(err) {
				actualConfigPath = "formats.json"
			}
		}
		err := runBootstrap(*yamlFile, actualConfigPath)
		if err != nil {
			log.Fatalf("Bootstrapping failed: %v", err)
		}
		log.Println("--- Bootstrap Complete ---")

		// Ask user if they want to continue
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Bootstrap finished. Continue with code generation for all formats? (y/N): ")
		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("Failed to read response: %v", err)
		}

		response = strings.ToLower(strings.TrimSpace(response))
		if response == "y" || response == "yes" {
			proceedWithGeneration = true
		} else {
			log.Println("Exiting after bootstrap.")
			os.Exit(0)
		}
	} else {
		proceedWithGeneration = true
	}

	// --- Normal Generation Logic ---
	if proceedWithGeneration {
		log.Println("--- Running Code Generation ---")
		// First try config/formats.json, fall back to formats.json if not specified
		actualConfigPath := *configPath
		if actualConfigPath == "" {
			actualConfigPath = "config/formats.json"
			if _, err := os.Stat(actualConfigPath); os.IsNotExist(err) {
				actualConfigPath = "formats.json"
			}
		}
		runGeneration(actualConfigPath)
		log.Println("--- Code Generation Complete ---")
	}
}

// --- Bootstrap Function (Modified) ---
func runBootstrap(yamlFile, configPath string) error {
	// Derive names
	yamlBaseName := strings.TrimSuffix(filepath.Base(yamlFile), filepath.Ext(yamlFile))
	if yamlBaseName == "" {
		return fmt.Errorf("could not derive base name from YAML file: %s", yamlFile)
	}

	// Try to find format in config
	var formatConfig *struct {
		Name        string `json:"name"`
		OutputDir   string `json:"outputDir"`
		PackageName string `json:"packageName"`
	}
	formatsData, _ := ioutil.ReadFile("config/formats.json")
	var formats []struct {
		Name        string `json:"name"`
		OutputDir   string `json:"outputDir"`
		PackageName string `json:"packageName"`
	}
	if err := json.Unmarshal(formatsData, &formats); err == nil {
		for _, f := range formats {
			if strings.EqualFold(f.Name, yamlBaseName) {
				formatConfig = &f
				break
			}
		}
	}

	// Set defaults from config or derive from filename
	formatName := strings.Title(yamlBaseName)
	outputDir := strings.ToLower(yamlBaseName)
	packageName := outputDir
	if formatConfig != nil {
		formatName = formatConfig.Name
		outputDir = formatConfig.OutputDir
		packageName = formatConfig.PackageName
	}

	// --- Define file paths ---
	// Stub file path *inside* the outputDir
	targetStubPath := fmt.Sprintf("%s_stubs.go", outputDir) // Just filename, will be joined with outputDir later
	// Test file path *inside* the outputDir
	testFilePath := filepath.Join(outputDir, fmt.Sprintf("%s_test.go", outputDir))

	log.Printf("Bootstrapping new format '%s' from '%s'", formatName, yamlFile)
	log.Printf(" -> Output Dir: %s", outputDir)
	log.Printf(" -> Package Name: %s", packageName)
	log.Printf(" -> Stub File (in output dir): %s", targetStubPath) // <-- Updated log
	log.Printf(" -> Test File (in output dir): %s", testFilePath)

	// Ensure output directory exists
	log.Printf("Ensuring output directory '%s' exists...", outputDir)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory '%s': %w", outputDir, err)
	}

	// Parse the input YAML
	log.Printf("Parsing YAML file: %s", yamlFile)
	yamlData, err := ioutil.ReadFile(yamlFile)
	if err != nil {
		return fmt.Errorf("failed to read YAML file '%s': %w", yamlFile, err)
	}
	var fileFormat application_structs.FileFormat
	err = yaml.Unmarshal(yamlData, &fileFormat)
	if err != nil {
		return fmt.Errorf("failed to parse YAML file '%s': %w", yamlFile, err)
	}

	// Prepare data for stub template
	stubTemplateData := TemplateData{
		FileName:    filepath.Base(targetStubPath), // Just filename for header
		FormatName:  formatName,
		PackageName: packageName,
		Structs:     make([]StructTemplateData, 0, len(fileFormat.Structs)),
	}
	for name, strct := range fileFormat.Structs {
		stubTemplateData.Structs = append(stubTemplateData.Structs, StructTemplateData{
			Name:   name,
			Fields: strct.Fields,
		})
	}

	// --- Generate the stub file directly (inside outputDir) ---
	log.Printf("Generating stub file: %s", targetStubPath) // Use target path
	stubFile, err := os.Create(targetStubPath)             // Use target path
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", targetStubPath, err)
	}
	tmplStub, err := template.New("resetStub").Parse(resetStubTemplate) // Use same template
	if err != nil {
		stubFile.Close()
		return fmt.Errorf("failed to parse stub template: %w", err)
	}
	err = tmplStub.Execute(stubFile, stubTemplateData)
	stubFile.Close()
	if err != nil {
		return fmt.Errorf("failed to execute stub template: %w", err)
	}

	// --- Generate the test file template (inside outputDir) ---
	log.Printf("Generating test template: %s", testFilePath)
	if _, err := os.Stat(testFilePath); err == nil {
		log.Printf("Warning: Test file '%s' already exists. Skipping generation.", testFilePath)
	} else if os.IsNotExist(err) {
		// ... (Test file generation logic remains the same) ...
		testOutFile, err := os.Create(testFilePath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", testFilePath, err)
		}
		tmplTest, err := template.New("testFile").Parse(testFileTemplate)
		if err != nil {
			testOutFile.Close()
			return fmt.Errorf("failed to parse test file template: %w", err)
		}
		testTemplateData := struct {
			FileName    string
			FormatName  string
			PackageName string
		}{
			FileName:    filepath.Base(testFilePath),
			FormatName:  formatName,
			PackageName: packageName,
		}
		err = tmplTest.Execute(testOutFile, testTemplateData)
		testOutFile.Close()
		if err != nil {
			return fmt.Errorf("failed to execute test file template: %w", err)
		}
	} else {
		log.Printf("Warning: Could not check status of test file '%s': %v. Skipping generation.", testFilePath, err)
	}

	// --- Update formats.json ---
	log.Printf("Updating configuration file: %s", configPath)
	var currentConfigs []FormatConfig
	configData, err := ioutil.ReadFile(configPath)
	// ... (JSON reading/parsing logic remains the same) ...
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("Configuration file '%s' not found, creating new one.", configPath)
			currentConfigs = []FormatConfig{}
		} else {
			return fmt.Errorf("failed to read configuration file '%s': %w", configPath, err)
		}
	} else {
		err = json.Unmarshal(configData, &currentConfigs)
		if err != nil {
			return fmt.Errorf("failed to parse configuration file '%s': %w", configPath, err)
		}
	}

	found := false
	for i := range currentConfigs {
		if currentConfigs[i].Name == formatName || currentConfigs[i].YAMLFile == yamlFile {
			log.Printf("Format '%s' (or YAML '%s') already exists in configuration. Updating paths.", formatName, yamlFile)
			currentConfigs[i].OutputDir = outputDir
			currentConfigs[i].PackageName = packageName
			// currentConfigs[i].ResetStubSource = "" // Remove this field
			currentConfigs[i].TargetStub = targetStubPath // Store full path
			currentConfigs[i].TestFile = testFilePath
			found = true
			break
		}
	}

	if !found {
		newConfig := FormatConfig{
			Name:        formatName,
			YAMLFile:    yamlFile,
			OutputDir:   outputDir,
			PackageName: packageName,
			// ResetStubSource: "", // Remove this field
			TargetStub: targetStubPath, // Store full path
			TestFile:   testFilePath,
		}
		currentConfigs = append(currentConfigs, newConfig)
		log.Printf("Added '%s' to configuration.", formatName)
	}

	// Marshal and write back
	updatedConfigData, err := json.MarshalIndent(currentConfigs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated configuration: %w", err)
	}
	err = ioutil.WriteFile(configPath, updatedConfigData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write updated configuration file '%s': %w", configPath, err)
	}

	log.Printf("Bootstrapping complete for %s. Please review generated files and complete test logic in %s.", formatName, testFilePath)
	return nil
}

// --- Format Selection Helper ---
func showFormatSelection(formatConfigs []FormatConfig) ([]FormatConfig, error) {
	reader := bufio.NewReader(os.Stdin)

	// Display available formats
	fmt.Println("Available formats:")
	for i, config := range formatConfigs {
		fmt.Printf("%d. %s\n", i+1, config.Name)
	}

	// Prompt for selection
	fmt.Print("\nSelect format to generate (number), or press Enter for all: ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %v", err)
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return formatConfigs, nil // Return all if no selection
	}

	// Parse selection
	selected, err := strconv.Atoi(input)
	if err != nil || selected < 1 || selected > len(formatConfigs) {
		return nil, fmt.Errorf("invalid selection: please enter a number between 1 and %d", len(formatConfigs))
	}

	return []FormatConfig{formatConfigs[selected-1]}, nil
}

// --- Normal Generation Function (Modified) ---
func runGeneration(configPath string) {
	log.Println("Reading formats configuration from", configPath)
	configData, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatalf("Failed to read %s: %v", configPath, err)
	}

	var formatConfigs []FormatConfig
	err = json.Unmarshal(configData, &formatConfigs)
	if err != nil {
		log.Fatalf("Failed to parse %s: %v", configPath, err)
	}

	// Get selected formats
	selectedConfigs, err := showFormatSelection(formatConfigs)
	if err != nil {
		log.Fatalf("Format selection failed: %v", err)
	}

	log.Printf("Processing %d selected format(s).", len(selectedConfigs))

	for _, config := range selectedConfigs {
		log.Printf("Processing format: %s", config.Name)

		fmt.Printf("Running generator reset for %s...\n", config.Name)
		// Pass the paths of the files to *keep* to the reset function
		reset(config.OutputDir, config.TargetStub, config.TestFile) // Pass stub and test file paths
		fmt.Println("Reset complete.")

		fmt.Println("Starting code generation...")

		if _, err := os.Stat(config.YAMLFile); os.IsNotExist(err) {
			log.Fatalf("%s: YAML file %s does not exist: %v", config.Name, config.YAMLFile, err)
		}
		if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
			log.Fatalf("%s: Failed to create output directory %s: %v", config.Name, config.OutputDir, err)
		}

		// Pass just the filename to GenerateCode and let it construct the full path
		err := generator.GenerateCode(config.YAMLFile, config.OutputDir, config.PackageName, filepath.Base(config.TargetStub))
		if err != nil {
			log.Fatalf("%s: Error generating code: %v", config.Name, err)
		}
		log.Printf("%s: Code generation completed successfully.", config.Name)
		fmt.Println("---")
	}

	log.Println("Selected format(s) processed.")
}
