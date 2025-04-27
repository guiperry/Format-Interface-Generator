package generator

import (
	"bytes"
	"fmt"
	"go/format"
	"io/ioutil"
	"log"
	"path/filepath"
	"sort"
	"text/template"

	"gopkg.in/yaml.v2"
)

func generateTestScript(reformedYamlPath, outputDir, packageName, goModulePath string) error {
	// 1. Parse the reformed YAML to find struct names
	yamlData, err := ioutil.ReadFile(reformedYamlPath)
	if err != nil {
		return fmt.Errorf("failed to read reformed YAML %s: %w", reformedYamlPath, err)
	}

	// Use a temporary struct to get just the struct keys
	var tempFormat struct {
		Structs map[string]interface{} `yaml:"structs"`
	}
	err = yaml.Unmarshal(yamlData, &tempFormat)
	if err != nil {
		return fmt.Errorf("failed to parse structs from reformed YAML %s: %w", reformedYamlPath, err)
	}

	if len(tempFormat.Structs) == 0 {
		return fmt.Errorf("no structs found in reformed YAML %s, cannot generate test", reformedYamlPath)
	}

	// Get struct names and sort them alphabetically for consistency
	structNames := make([]string, 0, len(tempFormat.Structs))
	for name := range tempFormat.Structs {
		structNames = append(structNames, name)
	}
	sort.Strings(structNames)
	firstStructName := structNames[0] // Use the first one alphabetically

	// 2. Prepare data for the test template
	testData := TestTemplateData{
		PackageName:     packageName,
		FormatDir:       filepath.Base(filepath.Dir(outputDir)), // e.g., "formats"
		FirstStructName: firstStructName,
		GoModulePath:    goModulePath,
	}

	// 3. Parse the test template
	tmpl, err := template.New("test").Parse(TestFileTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse test template: %w", err)
	}

	// 4. Execute the template
	var output bytes.Buffer
	err = tmpl.Execute(&output, testData)
	if err != nil {
		return fmt.Errorf("failed to execute test template for %s: %w", packageName, err)
	}

	// 5. Format the generated code
	formattedOutput, errFmt := format.Source(output.Bytes())
	if errFmt != nil {
		log.Printf("Warning: Failed to format generated test code for %s: %v. Writing unformatted code.", packageName, errFmt)
		formattedOutput = output.Bytes() // Fallback
	}

	// 6. Write the test file
	testFilePath := filepath.Join(outputDir, fmt.Sprintf("%s_test.go", packageName))
	err = ioutil.WriteFile(testFilePath, formattedOutput, 0644)
	if err != nil {
		return fmt.Errorf("failed to write test file %s: %w", testFilePath, err)
	}

	return nil
}