package utils

import (
	"os/exec" // For running 'go list'
	"strings"
	"fmt"
	"io/ioutil"
)

// --- NEW: Helper to get Go Module Path ---

func GetGoModulePath() (string, error) {
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