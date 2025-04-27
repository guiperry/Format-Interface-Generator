package dialogue

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"FIG/config"
	
)
// showSourceFileSelection prompts the user to select from available source YAML files.
func ShowSourceFileSelection(yamlFiles []string) ([]string, error) {
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
func ShowConfigSelection(formatConfigs []config.FormatConfig) ([]config.FormatConfig, error) {
	if len(formatConfigs) == 0 {
		return []config.FormatConfig{}, nil
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

	var selectedConfigs []config.FormatConfig
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
	uniqueConfigs := make([]config.FormatConfig, 0, len(selectedConfigs))
	seen := make(map[string]bool)
	for _, cfg := range selectedConfigs {
		if !seen[cfg.Name] { // Use Name as unique identifier
			uniqueConfigs = append(uniqueConfigs, cfg)
			seen[cfg.Name] = true
		}
	}
	return uniqueConfigs, nil
}



