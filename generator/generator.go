package generator

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"FormatModules/application_structs"

	"gopkg.in/yaml.v2"
)

// TemplateData holds all necessary info for template execution
type TemplateData struct {
	PackageName      string
	Imports         []string
	StructName       string
	Fields           []application_structs.Field
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

// ExpectedStruct holds both the field types and their order from the stub
type ExpectedStruct struct {
	FieldTypes map[string]string
	FieldOrder []string
}

// parseStubFileForExpectedStructs reads a Go stub file and extracts struct definitions.
func parseStubFileForExpectedStructs(stubFilePath string) (map[string]ExpectedStruct, error) {
	expected := make(map[string]ExpectedStruct)
	fset := token.NewFileSet() // Positions are relative to fset

	// Parse the Go source file
	node, err := parser.ParseFile(fset, stubFilePath, nil, parser.ParseComments)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("failed to parse stub file %s: %v", stubFilePath, err))
	}

	// Inspect the AST
	ast.Inspect(node, func(n ast.Node) bool {
		// Look for type declarations (like "type MyStruct struct { ... }")
		decl, ok := n.(*ast.GenDecl)
		if !ok || decl.Tok != token.TYPE {
			// Not a type declaration, continue traversal
			return true
		}

		// Iterate through the specs in the declaration (could be multiple types in one block)
		for _, spec := range decl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			structName := typeSpec.Name.Name

			// Check if it's a struct type
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue // Not a struct, skip
			}

			// Initialize struct for this struct if not already present
			if _, exists := expected[structName]; !exists {
				expected[structName] = ExpectedStruct{
					FieldTypes: make(map[string]string),
					FieldOrder: []string{},
				}
			}

			// Iterate through the fields of the struct
			if structType.Fields != nil {
				for _, field := range structType.Fields.List {
					// Get the field type as a string
					fieldType := ""
					// Handle different ways types can be represented in AST
					// This is a simplified version; more complex types (e.g., pointers, slices) might need more handling
					if ident, ok := field.Type.(*ast.Ident); ok {
						fieldType = ident.Name
					} else if selExpr, ok := field.Type.(*ast.SelectorExpr); ok {
						// Handles types like pkg.Type (e.g., time.Time) - unlikely in basic stubs but good practice
						if pkgIdent, ok := selExpr.X.(*ast.Ident); ok {
							fieldType = pkgIdent.Name + "." + selExpr.Sel.Name
						}
					} else if arrType, ok := field.Type.(*ast.ArrayType); ok {
						// Handle slice types like []byte
						if eltIdent, ok := arrType.Elt.(*ast.Ident); ok {
							fieldType = "[]" + eltIdent.Name
						}
					}
					// Add more type handlers if needed (e.g., *ast.StarExpr for pointers)

					// Get the field name(s) - usually one per line
					for _, name := range field.Names {
						fieldName := name.Name
						if fieldType != "" { // Only add if we could determine the type
							// Get the struct from map (or create new if not exists)
							expectedStruct := expected[structName]
							expectedStruct.FieldTypes[fieldName] = fieldType
							expectedStruct.FieldOrder = append(expectedStruct.FieldOrder, fieldName)
							expected[structName] = expectedStruct
						} else {
							log.Printf("Warning: Could not determine type for field '%s' in stub struct '%s'", fieldName, structName)
						}
					}
				}
			}
		}
		return true // Continue traversal
	})

	if len(expected) == 0 {
		log.Printf("Warning: No struct definitions found in stub file %s", stubFilePath)
	}

	return expected, nil
}

// GenerateCode takes the YAML description, generates Go code, and handles imports dynamically.
func GenerateCode(yamlFile, outputDir, packageName, targetStubName string) error {
	log.Printf("Starting code generation for file: %s, outputting to: %s (package %s)", yamlFile, outputDir, packageName)

	stubFilePath := filepath.Join(outputDir, targetStubName) // Define stub path early

	// --- START DYNAMIC EXPECTATION PARSING ---
	log.Printf("Parsing stub file %s for expected structure...", stubFilePath)
	expectedStructs, err := parseStubFileForExpectedStructs(stubFilePath)
	if err != nil {
		// If parsing fails (e.g., file not found), log a warning but maybe continue?
		// Or make it a fatal error depending on your workflow.
		// For now, let's make it non-fatal but log prominently.
		log.Printf("Warning: Failed to parse stub file '%s' to build expectations: %v. Proceeding without validation.", stubFilePath, err)
		// Optionally clear the map if parsing failed partially
		expectedStructs = make(map[string]ExpectedStruct) // Ensure it's empty if parsing failed
	} else if len(expectedStructs) > 0 {
		log.Println("Successfully parsed stub file for expectations.")
	} else {
		log.Println("No struct expectations loaded from stub file (file might be empty or contain no structs).")
	}
	// --- END DYNAMIC EXPECTATION PARSING ---

	// 1. Read the YAML file
	data, err := ioutil.ReadFile(yamlFile)
	if err != nil {
		return fmt.Errorf("error reading YAML file %s: %w", yamlFile, err)
	}
	log.Println("Successfully read YAML file.")

	// 2. Unmarshal the YAML data
	var fileFormat application_structs.FileFormat
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

	// Validate YAML against stub expectations if available
	if len(expectedStructs) > 0 {
		for structName, structDef := range fileFormat.Structs {
			if expected, exists := expectedStructs[structName]; exists {
				// Check for extra fields in YAML that aren't in stub
				for _, field := range structDef.Fields {
					if _, exists := expected.FieldTypes[field.Name]; !exists {
						log.Printf("Warning: Field '%s' in struct '%s' is not present in stub file", field.Name, structName)
					}
				}

				// Check field order matches stub
				if len(expected.FieldOrder) == len(structDef.Fields) {
					for i, field := range structDef.Fields {
						if i < len(expected.FieldOrder) && field.Name != expected.FieldOrder[i] {
							log.Printf("Warning: Field order mismatch in struct '%s' - expected '%s' at position %d but found '%s'",
								structName, expected.FieldOrder[i], i, field.Name)
						}
					}
				}
			}
		}
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to ensure output directory %s exists: %w", outputDir, err)
	}

	// 3. Parse the main template once, registering custom functions
	tmpl := template.New("struct").Funcs(template.FuncMap{
		"atoi": atoi,
		"isExpressionLength": func(f application_structs.Field) bool {
			return f.IsExpressionLength()
		},
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
					// Both fixed and expression lengths use err in Read
					fieldUsesErrRead = true
					// Check if length is valid int > 0 for ReadFull
					if _, errConv := strconv.Atoi(field.Length); errConv == nil && field.Length != "0" {
						fieldUsesErrRead = true // io.ReadFull uses err
					}
				}

				// Keep the existing logic for logging warnings/info based on Length for the Read method perspective
				if field.Length != "" {
					// Check if length is valid int
					if field.Length == "" {
						// Log warning about missing length
						log.Printf("Warning: []byte field '%s' in struct '%s' has no length specified. Read/Write logic might be incomplete.", field.Name, structName)
					} else if _, errConv := strconv.Atoi(field.Length); errConv != nil {
						// Log info about expression length
						log.Printf("Info: []byte field '%s' uses expression length '%s'. Generated code assumes dependencies are met.", field.Name, field.Length)
					}
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

	// --- START STUB REMOVAL STEP ---
	// Check if the stub file exists before trying to remove it
	if _, err := os.Stat(stubFilePath); err == nil {
		// Stub file exists, attempt to remove it
		log.Printf("Removing stub file: %s", stubFilePath)
		if err := os.Remove(stubFilePath); err != nil {
			// Log a warning if removal fails, but don't fail the whole process
			log.Printf("Warning: Failed to remove stub file %s: %v", stubFilePath, err)
		}
	} else if !os.IsNotExist(err) {
		// Log warning if there was an error checking for the stub file (other than not existing)
		log.Printf("Warning: Error checking for stub file %s: %v", stubFilePath, err)
	} else {
		// Stub file doesn't exist, which is fine after the first successful run
		// log.Printf("Stub file %s does not exist, skipping removal.", stubFilePath) // Optional log message
	}

	// 6. Log success and return nil only after the loop finishes
	log.Println("Code generation completed successfully.")
	return nil
	// --- END STUB REMOVAL STEP ---

	// 6. Log success and return nil only after the loop finishes
	log.Println("Code generation completed successfully.")
	return nil
}