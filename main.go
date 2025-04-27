// main.go
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	
	"FIG/generator"

)

const (
	sourceDir      = "sources" // Directory containing source YAML files
	formatsDir     = "formats" // Parent directory for generated format code
	configFileName = "config/formats.json"
)


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
		err := RunBootstrap(actualConfigPath)
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
	generator.RunGeneration(actualConfigPath)
	log.Println("--- Code Generation Complete ---")
}

