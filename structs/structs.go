// project/structs/structs.go
package structs

import (
	"fmt"
	"strconv"
)

type FileFormat struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Structs     map[string]Struct `yaml:"structs"`
}

type Struct struct {
	Fields []Field `yaml:"fields"`
}

type Field struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
	// Length can be a number (as string) or an expression (e.g., "Width*Height*3")
	// It's used for fixed-size strings, byte slices, or dynamic lengths
	Length string `yaml:"length,omitempty"`
	

}

func (f *Field) GetLength() (int, error) {
	// If Length is empty, return 0
	if f.Length == "" {
		return 0, nil
	}

	// First try to convert Length to an integer (for fixed lengths)
	if length, err := strconv.Atoi(f.Length); err == nil {
		return length, nil
	}

	// If not a number, assume it's an expression that will be evaluated at runtime
	return 0, nil
}

// IsExpressionLength returns true if the length is an expression (not a fixed number)
func (f *Field) IsExpressionLength() bool {
	if f.Length == "" {
		return false
	}
	_, err := strconv.Atoi(f.Length)
	return err != nil
}
// Validate checks if required fields are present and valid
func (f *Field) Validate() error {
	if f.Name == "" {
		return fmt.Errorf("field name cannot be empty")
	}
	if f.Type == "" {
		return fmt.Errorf("field %s: type cannot be empty", f.Name)
	}
	
	// For []byte and string types, Length is required
	if f.Type == "[]byte" || f.Type == "string" {
		if f.Length == "" {
			return fmt.Errorf("field %s (%s): length must be specified", f.Name, f.Type)
		}
	}
	
	return nil
}