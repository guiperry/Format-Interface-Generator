package utils

import (
	"fmt"
	"strings"

	"github.com/knetic/govaluate"
)

// GetExpressionFunctions defines functions usable in YAML Length expressions.
func GetExpressionFunctions() map[string]govaluate.ExpressionFunction {
	return map[string]govaluate.ExpressionFunction{
		// Example: BMP padding calculation
		"CalculatePaddedSize": func(args ...interface{}) (interface{}, error) {
			if len(args) != 3 {
				return nil, fmt.Errorf("CalculatePaddedSize expects 3 arguments (width, height, bitsPerPixel)")
			}

			// --- Careful argument conversion (govaluate often uses float64) ---
			var width, height, bitsPerPixel float64
			var ok bool

			width, ok = args[0].(float64)
			if !ok {
				return nil, fmt.Errorf("arg 1 (width) must be numeric for CalculatePaddedSize")
			}
			height, ok = args[1].(float64)
			if !ok {
				return nil, fmt.Errorf("arg 2 (height) must be numeric for CalculatePaddedSize")
			}
			bitsPerPixel, ok = args[2].(float64)
			if !ok {
				return nil, fmt.Errorf("arg 3 (bitsPerPixel) must be numeric for CalculatePaddedSize")
			}
			// --- End conversion ---

			if bitsPerPixel == 0 { // Avoid division by zero
				return nil, fmt.Errorf("bitsPerPixel cannot be zero")
			}

			bytesPerPixel := int(bitsPerPixel / 8)
			if bytesPerPixel <= 0 { // Handle cases like 1-bit, 4-bit BMPs if needed later
				return nil, fmt.Errorf("unsupported bitsPerPixel for simple calculation: %f", bitsPerPixel)
			}

			bytesPerRow := int(width) * bytesPerPixel
			paddingPerRow := (4 - (bytesPerRow % 4)) % 4 // Standard BMP padding logic
			paddedRowSize := bytesPerRow + paddingPerRow
			totalSize := int(height) * paddedRowSize

			return float64(totalSize), nil // Return as float64 for govaluate
		},
	}
}

// isValidLengthExpression remains the same (used by bootstrap now, but keep accessible)
func IsValidLengthExpression(expr string) bool {
	trimmed := strings.TrimSpace(expr)
	if trimmed == "" || trimmed == "..." {
		return false // Empty or placeholder is invalid here
	}
	return true
}
