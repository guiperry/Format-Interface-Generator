// generator/generator.go
package generator

import (
	"bytes"
	"fmt"
	// "go/ast" // No longer needed for stub parsing
	"go/format"
	// "go/parser" // No longer needed for stub parsing
	// "go/token" // No longer needed for stub parsing
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"FIG/app_structs"

	 
	"gopkg.in/yaml.v2"
)

// TemplateData holds all necessary info for template execution (no changes needed)
type TemplateData struct {
	PackageName      string
	Imports          []string
	StructName       string
	Fields           []app_structs.Field
	FieldMap         map[string]string
	VersionFieldPath string // Path to the version field
	// Flags to control variable declarations in the template
	NeedsErrVarRead  bool // True if any read operation generates code that uses 'err'
	NeedsErrVarWrite bool // True if any write operation generates code that uses 'err'
	NeedsBVar        bool // True if any string read operation generates code that uses 'b'
	NeedsTmpUint8    bool 
}

// atoi helper function (keep as is)
func atoi(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

// ExpectedStruct removed
// parseStubFileForExpectedStructs removed



// GenerateCode takes the YAML description, generates Go code, and handles imports dynamically.
// Assumes YAML is pre-validated. targetStubName is ignored.
func GenerateCode(yamlFile, outputDir, packageName, targetStubName string) error {
	// ... (Read YAML, Unmarshal YAML, Ensure output dir exists) ...
	log.Printf("Starting code generation for validated YAML: %s, outputting to: %s (package %s)", yamlFile, outputDir, packageName)

	// 1. Read the YAML file (this is the *reformed* YAML)
	data, err := ioutil.ReadFile(yamlFile)
	if err != nil {
		return fmt.Errorf("error reading YAML file %s: %w", yamlFile, err)
	}
	log.Println("Successfully read YAML file.")

	// 2. Unmarshal the YAML data
	var fileFormat app_structs.FileFormat
	err = yaml.Unmarshal(data, &fileFormat)
	if err != nil {
		yamlErr, ok := err.(*yaml.TypeError)
		if ok {
			for _, msg := range yamlErr.Errors {
				log.Printf("YAML unmarshal error: %s", msg)
			}
		}
		return fmt.Errorf("error unmarshaling YAML from %s: %w", yamlFile, err)
	}
	log.Println("Successfully unmarshaled YAML data.")

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to ensure output directory %s exists: %w", outputDir, err)
	}


	// 3. Parse the main template once...
	// ... (template parsing logic remains the same) ...
	tmpl := template.New("struct").Funcs(template.FuncMap{
		"atoi": atoi,
		"isExpressionLength": func(f app_structs.Field) bool {
			// Check if it's not an integer and not empty/placeholder
			_, err := strconv.Atoi(f.Length)
			return err != nil && f.Length != "" && f.Length != "NEEDS_MANUAL_LENGTH"
		},
		"isConditional": func(f app_structs.Field) bool {
			return f.IsConditional()
		},
		"generateConditionCheck": func(condition string) string {
			// Assume condition is reasonable (validated in bootstrap)
			return condition
		},
		// Add a helper to check for the manual length placeholder
		"needsManualLength": func(f app_structs.Field) bool {
			return f.Length == "NEEDS_MANUAL_LENGTH"
		},
	})
	tmpl, err = tmpl.Parse(StructTemplate) // Assumes StructTemplate is defined elsewhere
	if err != nil {
		return fmt.Errorf("error parsing base template: %w", err)
	}
	log.Println("Successfully parsed base template.")


	// 4. For each struct defined in the YAML, execute the template
	for structName, structDef := range fileFormat.Structs {
		log.Printf("Generating code for struct: %s", structName)

		// 4A. Determine required imports, build field map, AND check variable needs
		requiredImports := map[string]bool{"io": true}
		fieldMap := make(map[string]string)
		needsBinary := false
		needsFmt := false
		needsErrVarRead := false
		needsErrVarWrite := false
		needsBVar := false
		needsTmpUint8 := false // <-- Initialize flag
		needsGovaluate := false
		needsGeneratorHelpers := false

		// Iterate through fields to determine needs accurately
		for _, field := range structDef.Fields {
			fieldMap[field.Name] = field.Type
			fieldUsesErrRead := false
			fieldUsesErrWrite := false

			_, errConv := strconv.Atoi(field.Length)
			if errConv != nil && field.Length != "" && field.Length != "NEEDS_MANUAL_LENGTH" {
				needsGovaluate = true
				needsGeneratorHelpers = true
			}

			switch field.Type {
			case "uint8": // <-- Check for uint8
				needsTmpUint8 = true // <-- Set the flag if found
				needsBinary = true
				needsFmt = true
				fieldUsesErrRead = true
				fieldUsesErrWrite = true
			case "uint16", "uint32", "uint64", "int8", "int16", "int32", "int64", "float32", "float64":
				needsBinary = true
				needsFmt = true
				fieldUsesErrRead = true
				fieldUsesErrWrite = true
			case "string":
				needsFmt = true
				needsBVar = true
				if field.Length != "" && field.Length != "NEEDS_MANUAL_LENGTH" {
					fieldUsesErrRead = true
				}
				fieldUsesErrWrite = true

			case "[]byte":
				needsFmt = true
				if field.Length != "" && field.Length != "NEEDS_MANUAL_LENGTH" {
					fieldUsesErrRead = true
				}
				fieldUsesErrWrite = true

			default:
				needsFmt = true
				log.Printf("Info: Field '%s.%s' has custom type '%s'. Manual Read/Write implementation might be needed.", structName, field.Name, field.Type)
			}

			if fieldUsesErrRead {
				needsErrVarRead = true
			}
			if fieldUsesErrWrite {
				needsErrVarWrite = true
			}
		}

		// Determine imports based on flags
		// ... (import logic remains the same) ...
		if needsBinary {
			requiredImports["encoding/binary"] = true
		}
		if needsFmt || len(structDef.Fields) > 0 { // Include fmt if fields exist or errors are possible
			requiredImports["fmt"] = true
		}
		if needsGovaluate {
			requiredImports["github.com/knetic/govaluate"] = true
		}
		if needsGeneratorHelpers {
			// Assuming GetExpressionFunctions is in the 'generator' package
			// Adjust if moved to a different shared package
			requiredImports["FIG/utils"] = true
		}


		// Convert map keys to sorted slice for consistent import order
		importsList := make([]string, 0, len(requiredImports))
		for imp := range requiredImports {
			importsList = append(importsList, imp)
		}
		sort.Strings(importsList)

		// 4B. Prepare data for the template
		templateData := TemplateData{
			PackageName:      packageName,
			Imports:          importsList,
			StructName:       structName,
			Fields:           structDef.Fields,
			FieldMap:         fieldMap,
			VersionFieldPath: fileFormat.VersionFieldPath,
			NeedsErrVarRead:  needsErrVarRead,
			NeedsErrVarWrite: needsErrVarWrite,
			NeedsBVar:        needsBVar,
			NeedsTmpUint8:    needsTmpUint8, // <-- Pass the flag
		}

		// 4C. Execute the template
		// ... (template execution, formatting, writing remain the same) ...
		var output bytes.Buffer
		err = tmpl.Execute(&output, templateData)
		if err != nil {
			return fmt.Errorf("error executing template for struct %s: %w", structName, err)
		}
		log.Printf("Successfully executed template for %s.", structName)

		// 5. Write the generated code to a file
		goFileName := strings.Title(structName) + ".go"
		outputPath := filepath.Join(outputDir, goFileName)

		formattedOutput, errFmt := format.Source(output.Bytes())
		if errFmt != nil {
			log.Printf("Warning: Failed to format generated code for %s: %v. Writing unformatted code.", structName, errFmt)
			formattedOutput = output.Bytes() // Fallback
		}

		err = ioutil.WriteFile(outputPath, formattedOutput, 0644)
		if err != nil {
			return fmt.Errorf("error writing generated code to file %s: %w", outputPath, err)
		}
		log.Printf("Generated %s", outputPath)

	} // End loop through structs

	log.Println("Code generation completed successfully.")
	return nil
}
