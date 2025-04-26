package generator

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
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

	"github.com/knetic/govaluate"
	"gopkg.in/yaml.v2"
)

// TemplateData holds all necessary info for template execution
type TemplateData struct {
	PackageName      string
	Imports          []string
	StructName       string
	Fields           []application_structs.Field
	FieldMap         map[string]string
	VersionFieldPath string // Path to the version field
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
		return nil, fmt.Errorf("failed to parse stub file %s: %w", stubFilePath, err)
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

// isValidLengthExpression performs basic checks on a potential length expression.
// Returns true if it passes basic sanity checks, false otherwise.
// This is NOT a full Go expression parser.
func isValidLengthExpression(expr string) bool {
	trimmed := strings.TrimSpace(expr)
	if trimmed == "" || trimmed == "..." {
		return false // Empty or placeholder is invalid here
	}
	// Add more basic checks if needed, e.g., for invalid characters
	// For now, we assume if it's not empty or "..." and not an integer, it's intended as an expression.
	return true
}

// GenerateCode takes the YAML description, generates Go code, and handles imports dynamically.
func GenerateCode(yamlFile, outputDir, packageName, targetStubName string) error {
	log.Printf("Starting code generation for file: %s, outputting to: %s (package %s)", yamlFile, outputDir, packageName)

	stubFilePath := filepath.Join(outputDir, targetStubName) // Define stub path early

	// Rename stub file to avoid redeclaration errors during generation
	if strings.HasSuffix(stubFilePath, "_stubs.go") {
		newPath := strings.TrimSuffix(stubFilePath, ".go") + "._go"
		if err := os.Rename(stubFilePath, newPath); err != nil {
			log.Printf("Warning: Failed to rename stub file %s to %s: %v", stubFilePath, newPath, err)
		} else {
			stubFilePath = newPath
			defer func() {
				// Try to rename back after generation completes
				originalPath := strings.TrimSuffix(stubFilePath, "._go") + ".go"
				if err := os.Rename(stubFilePath, originalPath); err != nil {
					log.Printf("Warning: Failed to rename stub file back to original name %s: %v", originalPath, err)
				}
			}()
		}
	}

	// --- START DYNAMIC EXPECTATION PARSING ---
	log.Printf("Parsing stub file %s for expected structure...", stubFilePath)
	expectedStructs, err := parseStubFileForExpectedStructs(stubFilePath)
	if err != nil {
		log.Printf("Warning: Failed to parse stub file '%s' to build expectations: %v. Proceeding without validation.", stubFilePath, err)
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

	// --- START YAML VALIDATION ---
	log.Println("Validating YAML structure definitions...")
	for structName, structDef := range fileFormat.Structs {
		for _, field := range structDef.Fields {
			// Validate Length field based on Type
			switch field.Type {
			case "string", "[]byte":
				if field.Length == "" {
					// Strings and byte slices generally need a length for automatic reading/writing
					// Allow conditional fields to potentially skip length, but log a warning?
					// For now, enforce length for simplicity unless it's handled manually later.
					return fmt.Errorf("validation error in struct '%s': field '%s' of type '%s' requires a 'Length' specification (either a positive integer or a valid Go expression)", structName, field.Name, field.Type)
				}
				// Try parsing as integer first
				lengthInt, errConv := strconv.Atoi(field.Length)
				if errConv == nil {
					// It's an integer
					if lengthInt <= 0 {
						return fmt.Errorf("validation error in struct '%s': field '%s' has invalid non-positive integer 'Length: %s'", structName, field.Name, field.Length)
					}
					// Positive integer length is valid
				} else {
					// Not an integer, assume it's an expression. Perform basic checks.
					if !isValidLengthExpression(field.Length) {
						return fmt.Errorf("validation error in struct '%s': field '%s' has invalid 'Length: %s'. Must be a positive integer or a valid Go expression (cannot be empty or '...')", structName, field.Name, field.Length)
					}

					// Try to validate the expression with govaluate
					_, errExpr := govaluate.NewEvaluableExpressionWithFunctions(field.Length, GetExpressionFunctions())
					if errExpr != nil {
						log.Printf("Warning: Field '%s.%s' has expression length '%s' that may not be valid: %v",
							structName, field.Name, field.Length, errExpr)
					}

					// Passes basic expression checks
					log.Printf("Info: Field '%s.%s' uses expression length '%s'. Ensure the expression is valid Go code referencing struct fields with 's.'.", structName, field.Name, field.Length)
				}

			case "uint8", "uint16", "uint32", "uint64", "int8", "int16", "int32", "int64", "float32", "float64":
				// Fixed-size types should not have a Length specified
				if field.Length != "" {
					// Log a warning, but don't necessarily fail generation
					log.Printf("Warning: struct '%s': field '%s' of fixed-size type '%s' has an unnecessary 'Length: %s' specification. It will be ignored.", structName, field.Name, field.Type, field.Length)
				}
			default:
				// If it's a custom struct type, Length might not apply directly.
				// If Length is present, maybe warn?
				if field.Length != "" {
					log.Printf("Warning: struct '%s': field '%s' of type '%s' has a 'Length: %s' specification. Its usage depends on custom logic or nested struct handling.", structName, field.Name, field.Type, field.Length)
				}
			}

			// Validate Condition field if present
			if field.IsConditional() {
				trimmedCondition := strings.TrimSpace(field.Condition)
				if trimmedCondition == "" {
					return fmt.Errorf("validation error in struct '%s': conditional field '%s' has an empty 'Condition'", structName, field.Name)
				}
				// Basic check: ensure condition doesn't look obviously wrong (e.g., just operators)
				// A more robust check could involve parsing, but might be overkill.
				log.Printf("Info: Field '%s.%s' uses condition '%s'. Ensure the expression is valid Go code referencing struct fields with 's.'.", structName, field.Name, field.Condition)

			}
		}
		// Validate against stub expectations if available (Keep this existing logic)
		if expected, exists := expectedStructs[structName]; exists {
			// Check for extra fields in YAML that aren't in stub
			yamlFields := make(map[string]bool)
			for _, field := range structDef.Fields {
				yamlFields[field.Name] = true
				if _, existsInStub := expected.FieldTypes[field.Name]; !existsInStub {
					log.Printf("Warning: Field '%s' in YAML struct '%s' is not present in stub file '%s'", field.Name, structName, targetStubName)
				}
			}
			// Check for missing fields in YAML that are in stub
			for _, expectedFieldName := range expected.FieldOrder {
				if _, existsInYaml := yamlFields[expectedFieldName]; !existsInYaml {
					log.Printf("Warning: Field '%s' from stub file '%s' is missing in YAML struct '%s'", expectedFieldName, targetStubName, structName)
				}
			}

			// Check field order matches stub
			if len(expected.FieldOrder) == len(structDef.Fields) {
				for i, field := range structDef.Fields {
					if i < len(expected.FieldOrder) && field.Name != expected.FieldOrder[i] {
						log.Printf("Warning: Field order mismatch in struct '%s' at index %d - expected '%s' (from stub) but found '%s' (in YAML)",
							structName, i, expected.FieldOrder[i], field.Name)
					}
				}
			} else if len(expected.FieldOrder) > 0 { // Only warn if stub had fields
				log.Printf("Warning: Field count mismatch in struct '%s'. Stub file '%s' has %d fields, YAML has %d fields.",
					structName, targetStubName, len(expected.FieldOrder), len(structDef.Fields))
			}
		}
	}
	log.Println("YAML validation complete.")
	// --- END YAML VALIDATION ---

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to ensure output directory %s exists: %w", outputDir, err)
	}

	// 3. Parse the main template once... (rest of the function remains the same)
	tmpl := template.New("struct").Funcs(template.FuncMap{
		"atoi": atoi,
		"isExpressionLength": func(f application_structs.Field) bool {
			// Keep the original logic, validation happens before this
			_, err := strconv.Atoi(f.Length)
			return err != nil && f.Length != "" // It's not an int and not empty
		},
		"isConditional": func(f application_structs.Field) bool {
			return f.IsConditional()
		},
		"generateConditionCheck": func(condition string) string {
			// Assume validation ensured the condition is reasonable
			return condition
		},
	})
	tmpl, err = tmpl.Parse(StructTemplate)
	if err != nil {
		return fmt.Errorf("error parsing base template: %w", err)
	}
	log.Println("Successfully parsed base template.")

	// 4. For each struct defined in the YAML, execute the template
	for structName, structDef := range fileFormat.Structs {
		// ... (rest of the loop generating code remains the same) ...
		log.Printf("Generating code for struct: %s", structName)

		// 4A. Determine required imports, build field map, AND check variable needs
		requiredImports := map[string]bool{"io": true}
		fieldMap := make(map[string]string)
		needsBinary := false
		needsFmt := false
		needsErrVarRead := false  // Flag specific to Read function
		needsErrVarWrite := false // Flag specific to Write function
		needsBVar := false
		needsGovaluate := false // Flag for govaluate import

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
				// Validation ensures Length exists and is valid here
				// Check if length is > 0 int or expression for err usage
				_, errConv := strconv.Atoi(field.Length)
				if errConv == nil { // It's an int (already validated > 0)
					fieldUsesErrRead = true // io.ReadFull uses err
				} else { // It's an expression
					fieldUsesErrRead = true // Assume expression read uses err
					needsGovaluate = true   // Need govaluate for expression evaluation
				}
				fieldUsesErrWrite = true // Write always uses err for string

			case "[]byte":
				needsFmt = true
				// Validation ensures Length exists and is valid here
				fieldUsesErrWrite = true // Write always uses err for []byte
				// Check if length is > 0 int or expression for err usage
				_, errConv := strconv.Atoi(field.Length)
				if errConv == nil { // It's an int (already validated > 0)
					fieldUsesErrRead = true // io.ReadFull uses err
				} else { // It's an expression
					fieldUsesErrRead = true // Assume expression read uses err
					needsGovaluate = true   // Need govaluate for expression evaluation
				}

			default:
				// For custom struct types, we might need 'fmt' for errors if Read/Write are called
				// Let's assume fmt is needed if there's any field.
				needsFmt = true
				// We can't easily determine err usage for nested structs automatically here.
				// Assume they might use err for now.
				// TODO: Refine this if nested struct generation is added.
				log.Printf("Info: Field '%s.%s' has potentially unsupported type '%s' for automatic Read/Write generation.", structName, field.Name, field.Type)

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
		// Always include fmt if there are fields or potential errors
		if needsFmt || len(structDef.Fields) > 0 {
			requiredImports["fmt"] = true
		}
		// Add govaluate import if needed for expression evaluation
		if needsGovaluate {
			requiredImports["github.com/knetic/govaluate"] = true
			// Also need our helper functions
			requiredImports["FormatModules/generator"] = true
		}

		// Convert map keys to sorted slice for consistent import order
		importsList := make([]string, 0, len(requiredImports))
		for imp := range requiredImports {
			importsList = append(importsList, imp)
		}
		sort.Strings(importsList)

		// 4B. Prepare data for the template, including the new flags
		templateData := TemplateData{
			PackageName:      packageName,
			Imports:          importsList,
			StructName:       structName,
			Fields:           structDef.Fields,
			FieldMap:         fieldMap,
			VersionFieldPath: fileFormat.VersionFieldPath, // Pass the version field path
			NeedsErrVarRead:  needsErrVarRead,             // Pass specific flags
			NeedsErrVarWrite: needsErrVarWrite,
			NeedsBVar:        needsBVar,
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
		goFileName := strings.Title(structName) + ".go" // Use Title case for filename
		outputPath := filepath.Join(outputDir, goFileName)

		formattedOutput, errFmt := format.Source(output.Bytes())
		if errFmt != nil {
			log.Printf("Warning: Failed to format generated code for %s: %v. Writing unformatted code.", structName, errFmt)
			// Log the unformatted code if formatting fails, might help debug template issues
			// log.Printf("Unformatted code for %s:\n%s", structName, output.String())
			formattedOutput = output.Bytes() // Fallback
		}

		err = ioutil.WriteFile(outputPath, formattedOutput, 0644)
		if err != nil {
			return fmt.Errorf("error writing generated code to file %s: %w", outputPath, err)
		}
		log.Printf("Generated %s", outputPath)

	} // End loop through structs

	// --- START STUB REMOVAL STEP ---
	// ... (Keep stub removal logic as is) ...
	if _, err := os.Stat(stubFilePath); err == nil {
		log.Printf("Removing stub file: %s", stubFilePath)
		if err := os.Remove(stubFilePath); err != nil {
			log.Printf("Warning: Failed to remove stub file %s: %v", stubFilePath, err)
		}
	} else if !os.IsNotExist(err) {
		log.Printf("Warning: Error checking for stub file %s: %v", stubFilePath, err)
	}

	// 6. Log success and return nil only after the loop finishes
	log.Println("Code generation completed successfully.")
	return nil
}
