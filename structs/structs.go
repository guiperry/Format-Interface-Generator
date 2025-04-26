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
	// It's primarily used for fixed-size strings or byte slices.
	Length string `yaml:"length,omitempty"`
	// We might need more fields later if we support complex array logic,
	// but let's keep it simple for now.
	// isArray, arrayLengthField, stringLength removed for clarity unless needed
	StringLength string `yaml:"stringLength,omitempty"`
	// isArray bool `yaml:"isArray,omitempty"`
	

}

func (f *Field) GetLength() (int, error) {
	// If Length is empty, return 0
	if f.Length == "" {
		return 0, nil
	}

	// Convert Length to an integer
	length, err := strconv.Atoi(f.Length)
	if err != nil {
		return 0, fmt.Errorf("invalid length for field %s: %w", f.Name, err)
	}

	return length, nil
}
func (f *Field) GetStringLength() (int, error) {
	// If StringLength is empty, return 0
	if f.StringLength == "" {
		return 0, nil
	}

	// Convert StringLength to an integer
	stringLength, err := strconv.Atoi(f.StringLength)
	if err != nil {
		return 0, fmt.Errorf("invalid string length for field %s: %w", f.Name, err)
	}

	return stringLength, nil
}