package config

import (
	"encoding/json"
	"io/ioutil"
	"strings"
)

type formatConfig struct {
	Name       string `json:"name"`
	YAMLFile   string `json:"yamlFile"`
	OutputDir  string `json:"outputDir"`
	TargetStub string `json:"targetStub"`
}

var formatDefaults = make(map[string]formatConfig)

func init() {
	data, err := ioutil.ReadFile("formats.json")
	if err != nil {
		return
	}
	
	var formats []formatConfig
	if err := json.Unmarshal(data, &formats); err == nil {
		for _, f := range formats {
			formatDefaults[strings.ToLower(f.Name)] = f
		}
	}
}

func GetDefaults(format string) (yamlFile, outputDir, targetStub string) {
	if f, ok := formatDefaults[strings.ToLower(format)]; ok {
		return f.YAMLFile, f.OutputDir, f.TargetStub
	}
	return format + ".yml", strings.ToLower(format), format + "_stubs.go"
}