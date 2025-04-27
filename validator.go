// validator.go
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"FormatModules/application_structs"
	"FormatModules/generator" // Need access to helpers

	"github.com/knetic/govaluate"
	"gopkg.in/yaml.v2"
)

// ValidateAndReformYAML reads the original YAML, validates/reforms it,
// and saves the result to the target path in the output directory.
// It returns the path to the saved reformed YAML file.
func ValidateAndReformYAML(originalYAMLPath, outputDir string) (string, error) {
	// Calculate the path where the reformed YAML should reside inside the output dir
	reformedYamlPath := filepath.Join(outputDir, filepath.Base(originalYAMLPath))

	log.Printf("Validating/Reforming '%s' -> '%s'", originalYAMLPath, reformedYamlPath)

	// --- 1. Ensure output directory exists ---
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to ensure output directory '%s': %w", outputDir, err)
	}

	// --- 2. Read and Unmarshal the original YAML ---
	yamlData, err := ioutil.ReadFile(originalYAMLPath)
	if err != nil {
		return "", fmt.Errorf("failed to read original YAML file '%s': %w", originalYAMLPath, err)
	}
	var fileFormat application_structs.FileFormat
	err = yaml.Unmarshal(yamlData, &fileFormat)
	if err != nil {
		yamlErr, ok := err.(*yaml.TypeError)
		if ok {
			for _, msg := range yamlErr.Errors {
				log.Printf("YAML unmarshal error in %s: %s", originalYAMLPath, msg)
			}
		}
		return "", fmt.Errorf("error unmarshaling YAML from %s: %w", originalYAMLPath, err)
	}

	// --- 3. Validate and Reform YAML Data ---
	reformationsMade := 0
	validationErrors := 0

	for structName, structDef := range fileFormat.Structs {
		tempStructDef := structDef
		for i := range tempStructDef.Fields {
			field := &tempStructDef.Fields[i] // Use pointer to modify

			// Validate Type exists
			if strings.TrimSpace(field.Type) == "" {
				log.Printf("ERROR: Validation error in struct '%s': field '%s' is missing a 'type'.", structName, field.Name)
				validationErrors++
				continue // Skip further checks for this field
			}

			// Validate Length based on Type
			switch field.Type {
			case "string", "[]byte":
				if field.Length == "" {
					log.Printf("ERROR: Validation error in struct '%s': field '%s' of type '%s' requires a 'Length' specification.", structName, field.Name, field.Type)
					validationErrors++
					continue
				}
				if strings.TrimSpace(field.Length) == "..." {
					log.Printf("WARNING: Reforming struct '%s': field '%s' had invalid 'Length: ...'. Replacing with 'NEEDS_MANUAL_LENGTH'.", structName, field.Name)
					field.Length = "NEEDS_MANUAL_LENGTH" // Modify via pointer
					reformationsMade++
				}
				lengthInt, errConv := strconv.Atoi(field.Length)
				if errConv == nil { // Is an integer
					if lengthInt <= 0 {
						log.Printf("ERROR: Validation error in struct '%s': field '%s' has invalid non-positive integer 'Length: %s'", structName, field.Name, field.Length)
						validationErrors++
					}
				} else { // Not an integer, assume expression or placeholder
					if !generator.IsValidLengthExpression(field.Length) { // Checks for empty, "..."
						log.Printf("ERROR: Validation error in struct '%s': field '%s' has invalid 'Length: %s'. Must be a positive integer or a valid Go expression (cannot be empty or '...')", structName, field.Name, field.Length)
						validationErrors++
					} else if field.Length != "NEEDS_MANUAL_LENGTH" { // Don't try to validate our placeholder
						// Attempt basic static validation of the expression
						_, errExpr := govaluate.NewEvaluableExpressionWithFunctions(field.Length, generator.GetExpressionFunctions())
						if errExpr != nil {
							// Log as warning, complex expressions might fail static check but work at runtime
							log.Printf("Warning: Field '%s.%s' has expression length '%s' that may not be fully validatable statically: %v",
								structName, field.Name, field.Length, errExpr)
						}
					}
				}

			case "uint8", "uint16", "uint32", "uint64", "int8", "int16", "int32", "int64", "float32", "float64":
				if field.Length != "" {
					log.Printf("Warning: struct '%s': field '%s' of fixed-size type '%s' has an unnecessary 'Length: %s'. It will be ignored during generation.", structName, field.Name, field.Type, field.Length)
					// Optionally reform: field.Length = ""
				}
			default: // Custom struct types
				if field.Length != "" {
					log.Printf("Warning: struct '%s': field '%s' of custom type '%s' has a 'Length: %s' specification. Its usage depends on custom logic.", structName, field.Name, field.Type, field.Length)
				}
			}

			// Validate Condition
			if field.IsConditional() {
				trimmedCondition := strings.TrimSpace(field.Condition)
				if trimmedCondition == "" {
					log.Printf("ERROR: Validation error in struct '%s': conditional field '%s' has an empty 'Condition'", structName, field.Name)
					validationErrors++
				}
			}

			// Validate Tags (optional, basic check)
			if field.Tags != "" {
				// Could add more sophisticated tag validation if needed
				log.Printf("Info: Field '%s.%s' has tags: `%s`", structName, field.Name, field.Tags)
			}
		}
		fileFormat.Structs[structName] = tempStructDef // Update map with potentially modified struct
	}

	if validationErrors > 0 {
		return "", fmt.Errorf("found %d critical validation error(s) in %s. Please fix the original YAML", validationErrors, originalYAMLPath)
	}
	if reformationsMade > 0 {
		log.Printf("Made %d reformation(s) to the YAML data for %s.", reformationsMade, originalYAMLPath)
	} else {
		log.Printf("YAML validation complete for %s. No reformations needed.", originalYAMLPath)
	}

	// --- 4. Marshal and Save Reformed YAML ---
	reformedYamlData, err := yaml.Marshal(&fileFormat)
	if err != nil {
		return "", fmt.Errorf("failed to marshal reformed YAML data for %s: %w", originalYAMLPath, err)
	}
	err = ioutil.WriteFile(reformedYamlPath, reformedYamlData, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write reformed YAML file '%s': %w", reformedYamlPath, err)
	}
	log.Printf("Saved validated/reformed YAML to: %s", reformedYamlPath)

	return reformedYamlPath, nil // Return the path to the saved file
}
