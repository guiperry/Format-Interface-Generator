package config

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"fmt"
)

type FormatConfig struct {
	Name       string `json:"name"`
	YAMLFile   string `json:"yamlFile"`
	OutputDir  string `json:"outputDir"`
	PackageName string `json:"packageName"` // Go package name
}

// loadConfig reads and parses the formats.json file.
func LoadConfig(configPath string) ([]FormatConfig, error) {
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
func SaveConfig(configPath string, configs []FormatConfig) error {
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
