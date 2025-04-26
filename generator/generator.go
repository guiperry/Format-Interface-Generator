package generator

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"FormatModules/structs"

	"gopkg.in/yaml.v2"
)

// TemplateData holds all necessary info for template execution
type TemplateData struct {
	PackageName      string
	Imports         []string
	StructName       string
	Fields           []structs.Field
	FieldMap         map[string]string
	// Flags to control variable declarations in the template
	NeedsErrVarRead  bool // True if any read operation generates code that uses 'err'
	NeedsErrVarWrite bool // True if any write operation generates code that uses 'err'
	NeedsBVar        bool // True if any string read operation generates code that uses 'b'
}

// atoi helper function (keep as is)
func atoi(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

// GenerateCode takes the YAML description, generates Go code, and handles imports dynamically.
func GenerateCode(yamlFile, outputDir, packageName string) error {
	log.Printf("Starting code generation for file: %s, outputting to: %s (package %s)", yamlFile, outputDir, packageName)
	// 1. Read the YAML file
	data, err := ioutil.ReadFile(yamlFile)
	if err != nil {
		return fmt.Errorf("error reading YAML file %s: %w", yamlFile, err)
	}
	log.Println("Successfully read YAML file.")

	// 2. Unmarshal the YAML data
	var fileFormat structs.FileFormat
	err = yaml.Unmarshal(data, &fileFormat)
	if err != nil {
		// Provide more context on YAML parsing errors
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

	// 3. Parse the main template once, registering custom functions
	tmpl := template.New("struct").Funcs(template.FuncMap{
		"atoi": atoi,
	})
	tmpl, err = tmpl.Parse(StructTemplate)
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
		needsErrVarRead := false  // Flag specific to Read function
		needsErrVarWrite := false // Flag specific to Write function
		needsBVar := false

		// Iterate through fields to determine needs accurately
		for _, field := range structDef.Fields {
			fieldMap[field.Name] = field.Type
			fieldUsesErrRead := false  // Does this field's Read logic use err?
			fieldUsesErrWrite := false // Does this field's Write logic use err?

			switch field.Type {
			case "uint8", "uint16", "uint32", "uint64", "int8", "int16", "int32", "int64", "float32", "float64":
				needsBinary = true
				needsFmt = true
				fieldUsesErrRead = true  // binary.Read uses err
				fieldUsesErrWrite = true // binary.Write uses err
			case "string":
				needsFmt = true
				needsBVar = true
				if field.Length != "" {
					// Check if length is valid int
					if _, errConv := strconv.Atoi(field.Length); errConv == nil && field.Length != "0" {
						// Fixed length string read uses err
						fieldUsesErrRead = true
					}
					// If length is expression or 0, the template returns early - no err use here
				}
				// If no length, template returns early - no err use here
			case "[]byte":
				needsFmt = true
				fieldUsesErrWrite = true // Write always uses err for []byte
				if field.Length != "" {
					// Check if length is valid int > 0 for ReadFull
					if _, errConv := strconv.Atoi(field.Length); errConv == nil && field.Length != "0" {
						fieldUsesErrRead = true // io.ReadFull uses err
					}
				}

				// Keep the existing logic for logging warnings/info based on Length for the Read method perspective
				if field.Length != "" {
					// Check if length is valid int
					if _, errConv := strconv.Atoi(field.Length); errConv != nil && field.Length != "0" {
						// Log info about expression length
						log.Printf("Info: []byte field '%s' uses expression length '%s'. Generated code assumes dependencies are met.", field.Name, field.Length)
					}
					// No need to check if length is 0 here for fieldUsesErr anymore
				} else {
					// Log warning about missing length for Read
					log.Printf("Warning: []byte field '%s' in struct '%s' has no length specified. Read/Write logic might be incomplete.", field.Name, structName)
				}
			default:
				// Unsupported type path returns early - no err use here
				needsFmt = true
			}

			// Aggregate flags: if *any* field uses err in Read, the function needs the var
			if fieldUsesErrRead {
				needsErrVarRead = true
			}
			// Aggregate flags: if *any* field uses err in Write, the function needs the var
			if fieldUsesErrWrite {
				needsErrVarWrite = true
			}
		}

		// Determine imports based on flags
		if needsBinary {
			requiredImports["encoding/binary"] = true
		}
		// Always include fmt if there are fields, for potential error messages
		if needsFmt || len(structDef.Fields) > 0 {
			requiredImports["fmt"] = true
		}

		// Convert map keys to sorted slice for consistent import order
		importsList := make([]string, 0, len(requiredImports))
		for imp := range requiredImports {
			importsList = append(importsList, imp)
		}
		sort.Strings(importsList)

		// 4B. Prepare data for the template, including the new flags
		templateData := TemplateData{
			PackageName: packageName,
			Imports:     importsList,
			StructName:  structName,
			Fields:      structDef.Fields,
			FieldMap:    fieldMap,
			NeedsErrVarRead:  needsErrVarRead,  // Pass specific flags
			NeedsErrVarWrite: needsErrVarWrite,
			NeedsBVar:   needsBVar,
		}

		// 4C. Execute the template
		var output bytes.Buffer
		err = tmpl.Execute(&output, templateData)
		if err != nil {
			// Provide template execution context in error
			return fmt.Errorf("error executing template for struct %s: %w", structName, err)
		}
		log.Printf("Successfully executed template for %s.", structName)

		// 5. Write the generated code to a file
		// Use Capitalized struct name for the Go file name convention
		goFileName := strings.Title(structName) + ".go"
		outputPath := filepath.Join(outputDir, goFileName)

		// Use ioutil.WriteFile (or os.WriteFile in Go 1.16+)
		err = ioutil.WriteFile(outputPath, output.Bytes(), 0644)
		if err != nil {
			return fmt.Errorf("error writing generated code to file %s: %w", outputPath, err)
		}
		log.Printf("Generated %s", outputPath)
		
	}
	// 6. Log success and return nil only after the loop finishes
	log.Println("Code generation completed successfully.")
	return nil

}