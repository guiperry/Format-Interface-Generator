// validator.go
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"FormatModules/application_structs"
	"FormatModules/generator"

	"github.com/knetic/govaluate"
	"github.com/mitchellh/mapstructure" // Import mapstructure
	"gopkg.in/yaml.v2"
)

// lowercaseFieldKeysRecursive (Keep the existing function for now,
// although we suspect it might have a subtle bug, mapstructure might tolerate it better
// or we can refine it later if needed)
func lowercaseFieldKeysRecursive(data interface{}) interface{} {
	// ... (existing implementation) ...
	if data == nil {
		return nil
	}

	originalValue := reflect.ValueOf(data)
	kind := originalValue.Kind()

	// Define the keys that correspond to the application_structs.Field tags
	knownFieldKeys := map[string]bool{
		"name":        true,
		"type":        true,
		"description": true,
		"length":      true,
		"condition":   true,
		"tags":        true,
	}

	switch kind {
	case reflect.Map:
		// Create a new map to store results
		newMap := make(map[string]interface{})
		// Iterate over the map keys
		iter := originalValue.MapRange()
		for iter.Next() {
			key := iter.Key()
			val := iter.Value()

			// Recursively process the value *before* deciding on the key
			processedValue := lowercaseFieldKeysRecursive(val.Interface())

			// Handle the key
			if key.Kind() == reflect.String {
				keyStr := key.String()
				lowerKeyStr := strings.ToLower(keyStr)

				// Check if the lowercased key is one of the known Field keys
				if _, isFieldKey := knownFieldKeys[lowerKeyStr]; isFieldKey {
					// Use the lowercased key for known field attributes
					newMap[lowerKeyStr] = processedValue
				} else {
					// Keep the original key casing for other keys (struct names, "structs", "fields", etc.)
					newMap[keyStr] = processedValue
				}
			} else {
				// Keep non-string keys as they are (though unlikely in YAML structure)
				// Use fmt.Sprintf for safety, although non-string keys are unexpected here
				newMap[fmt.Sprintf("%v", key.Interface())] = processedValue
			}
		}
		return newMap

	case reflect.Slice, reflect.Array:
		// Create a new slice of the same size
		newSlice := make([]interface{}, originalValue.Len())
		for i := 0; i < originalValue.Len(); i++ {
			// Recursively process each element
			newSlice[i] = lowercaseFieldKeysRecursive(originalValue.Index(i).Interface())
		}
		return newSlice

	default:
		// Return primitive types or other kinds as is
		return data
	}
}


// ValidateAndReformYAML reads the original YAML, handles key case-insensitivity,
// validates/reforms values, and saves the result to the target path.
// It returns the path to the saved reformed YAML file.
func ValidateAndReformYAML(originalYAMLPath, outputDir string) (string, error) {
	// Calculate the path where the reformed YAML should reside inside the output dir
	reformedYamlPath := filepath.Join(outputDir, filepath.Base(originalYAMLPath))

	log.Printf("Validating/Reforming '%s' -> '%s'", originalYAMLPath, reformedYamlPath)

	// --- 1. Ensure output directory exists ---
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to ensure output directory '%s': %w", outputDir, err)
	}

	// --- 2. Read original YAML bytes ---
	yamlBytes, err := ioutil.ReadFile(originalYAMLPath)
	if err != nil {
		return "", fmt.Errorf("failed to read original YAML file '%s': %w", originalYAMLPath, err)
	}

	// --- 3. Unmarshal into generic map ---
	var genericData map[string]interface{}
	err = yaml.Unmarshal(yamlBytes, &genericData)
	if err != nil {
		return "", fmt.Errorf("error parsing initial YAML structure from %s: %w", originalYAMLPath, err)
	}

	// --- 4. Recursively lowercase specific field keys ---
	log.Printf("Normalizing specific YAML field keys to lowercase for %s...", originalYAMLPath)
	lowerCasedData := lowercaseFieldKeysRecursive(genericData)

	// --- 5. Decode the modified map directly into the struct using mapstructure ---
	log.Printf("Decoding normalized map into struct for %s...", originalYAMLPath)
	var fileFormat application_structs.FileFormat
	// Configure mapstructure to use the 'yaml' tag
	config := &mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   &fileFormat,
		TagName:  "yaml", // Tell mapstructure to use the 'yaml' tags
		WeaklyTypedInput: true,
	}
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return "", fmt.Errorf("failed to create mapstructure decoder for %s: %w", originalYAMLPath, err)
	}
	// Decode the lowerCasedData map (which should have correct keys now)
	if err := decoder.Decode(lowerCasedData); err != nil {
		log.Printf("Mapstructure decoding error details: %v", err)
		return "", fmt.Errorf("error decoding normalized map to struct for %s: %w", originalYAMLPath, err)
	}
	log.Printf("Successfully decoded normalized map for %s.", originalYAMLPath)


	// --- Steps 5 & 6 (Marshal/Unmarshal normalized bytes) are REMOVED ---
	// We now directly decode the map in the new Step 5 above.

	// --- 7. Validate and Reform Values (Existing Logic) ---
	// This logic now operates on the fileFormat struct populated by mapstructure
	reformationsMade := 0
	validationErrors := 0
	// ... (Keep the entire validation loop exactly as it was) ...
	for structName, structDef := range fileFormat.Structs {
		tempStructDef := structDef
		for i := range tempStructDef.Fields {
			field := &tempStructDef.Fields[i] // Use pointer to modify

			// Validate Type exists (should be populated now)
			if strings.TrimSpace(field.Type) == "" {
				log.Printf("ERROR: Validation error in struct '%s': field '%s' is missing a 'type'.", structName, field.Name)
				validationErrors++
				continue // Skip further checks for this field
			}

			// Validate Length based on Type (existing logic)
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
						_, errExpr := govaluate.NewEvaluableExpressionWithFunctions(field.Length, generator.GetExpressionFunctions())
						if errExpr != nil {
							log.Printf("Warning: Field '%s.%s' has expression length '%s' that may not be fully validatable statically: %v",
								structName, field.Name, field.Length, errExpr)
						}
					}
				}

			case "uint8", "uint16", "uint32", "uint64", "int8", "int16", "int32", "int64", "float32", "float64":
				if field.Length != "" {
					log.Printf("Warning: struct '%s': field '%s' of fixed-size type '%s' has an unnecessary 'Length: %s'. It will be ignored during generation.", structName, field.Name, field.Type, field.Length)
				}
			default: // Custom struct types
				if field.Length != "" {
					log.Printf("Warning: struct '%s': field '%s' of custom type '%s' has a 'Length: %s' specification. Its usage depends on custom logic.", structName, field.Name, field.Type, field.Length)
				}
			}

			// Validate Condition (existing logic)
			if field.IsConditional() {
				trimmedCondition := strings.TrimSpace(field.Condition)
				if trimmedCondition == "" {
					log.Printf("ERROR: Validation error in struct '%s': conditional field '%s' has an empty 'Condition'", structName, field.Name)
					validationErrors++
				}
			}

			// Validate Tags (existing logic)
			if field.Tags != "" {
				log.Printf("Info: Field '%s.%s' has tags: `%s`", structName, field.Name, field.Tags)
			}
		}
		fileFormat.Structs[structName] = tempStructDef // Update map with potentially modified struct
	}


	if validationErrors > 0 {
		return "", fmt.Errorf("found %d critical validation error(s) in %s (after key normalization). Please fix the original YAML", validationErrors, originalYAMLPath)
	}
	if reformationsMade > 0 {
		log.Printf("Made %d value reformation(s) to the YAML data for %s.", reformationsMade, originalYAMLPath)
	} else {
		log.Printf("YAML value validation complete for %s. No value reformations needed.", originalYAMLPath)
	}

	// --- 8. Marshal the final validated/reformed struct back to YAML ---
	finalYamlData, err := yaml.Marshal(&fileFormat) // Marshal the validated struct
	if err != nil {
		return "", fmt.Errorf("failed to marshal final reformed YAML data for %s: %w", originalYAMLPath, err)
	}

	// --- 9. Save the final YAML ---
	err = ioutil.WriteFile(reformedYamlPath, finalYamlData, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write final reformed YAML file '%s': %w", reformedYamlPath, err)
	}
	log.Printf("Saved validated/reformed YAML to: %s", reformedYamlPath)

	return reformedYamlPath, nil
}
