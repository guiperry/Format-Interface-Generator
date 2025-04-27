package main

import (
	"log"
	"path/filepath"
	"sort"
	"strings"
	"fmt"
	"io/ioutil"
	
	"FIG/config"
	"FIG/utils"
	"FIG/dialogue"
)

// --- Bootstrap Function ---
func RunBootstrap(configPath string) error {
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
	selectedYamlFiles, err := dialogue.ShowSourceFileSelection(yamlFiles)
	if err != nil {
		return fmt.Errorf("format selection failed: %w", err)
	}

	if len(selectedYamlFiles) == 0 {
		log.Println("No formats selected for bootstrapping.")
		return nil
	}

	log.Printf("Processing %d selected source file(s) for bootstrap...", len(selectedYamlFiles))

	// --- Load existing config ---
	currentConfigs, err := config.LoadConfig(configPath)
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
		_, err := utils.ValidateAndReformYAML(yamlFile, outputDir)
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
			newConfig := config.FormatConfig{
				Name:        formatName,
				YAMLFile:    yamlFile,  // Store relative path from project root
				OutputDir:   outputDir, // Store relative path from project root
				PackageName: packageName,
			}
			currentConfigs = append(currentConfigs, newConfig)
			configUpdated = true
		}
	}

	// --- Save updated config if changes were made ---
	if configUpdated {
		err = config.SaveConfig(configPath, currentConfigs)
		if err != nil {
			return err
		}
		log.Println("Configuration file updated.")
	} else {
		log.Println("No configuration changes needed.")
	}

	return nil
}


// --- Helper Functions ---

