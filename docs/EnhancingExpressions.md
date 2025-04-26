## Enhancing the Generator to Handle Complex Length Expressions

Enhancing the generator to handle more complex Length expressions, especially those depending on data outside the immediate struct (like fields from a previously read header), is a significant but powerful upgrade.

The most robust way to achieve this involves introducing a context mechanism and leveraging a Go expression evaluation library.

Here's a breakdown of the steps and necessary code changes:

**1. Introduce a Context Parameter to Read Methods**

*   **Goal:** Allow the code calling `Read` to pass necessary information (like a previously read header) into the `Read` method.
*   **Change:** Modify the `Read` method signature in the template.

```go
// generator/templates.go (Modify the Read method signature)

// Read populates the struct fields by reading from an io.Reader, using optional context.
// The context can be used by dynamic length calculations.
func (s *{{.StructName}}) Read(r io.Reader, ctx interface{}) error { // Added ctx interface{} parameter
    {{if .NeedsErrVarRead}}var err error{{end}}
    {{if .NeedsBVar}}var b []byte{{end}}
    // ... rest of the Read method template ...
}
```

*Why `interface{}`?* It offers maximum flexibility. The calling code decides what concrete type to pass (e.g., `*bmp.InfoHeader`, a custom context struct, or `nil` if no context is needed). The expression evaluation will handle accessing fields from it.

**2. Integrate an Expression Evaluation Library**

*   **Goal:** Safely parse and execute the `Length` expressions from YAML at runtime, using the current struct (`s`) and the provided context (`ctx`).
*   **Recommendation:** Use a library like `govaluate` (`github.com/knetic/govaluate`) or `expr` (`github.com/expr-lang/expr`). We'll use `govaluate` for this example.
*   **Add Dependency:** `bash go get github.com/knetic/govaluate`
*   **Modify Template (`generator/templates.go`):** Replace the simple `size := int(s.{{$field.Length}})` logic with evaluation logic for `string` and `[]byte` fields where `isExpressionLength` is true.

```go
// generator/templates.go (Inside the Read method, within the range loop)

// Add govaluate import to the generated file's imports
// (You'll need to adjust the import logic in generator.go to add this conditionally
// if isExpressionLength is used anywhere)

// --- Replace existing dynamic length logic for string ---
{{if eq $field.Type "string"}}
    {{if $field.Length}}
        {{if isExpressionLength $field}}
// Dynamic length string field: {{$field.Name}} using expression: {{$field.Length}}
expressionStr := `{{$field.Length}}` // Use backticks for raw string literal
expression, errExpr := govaluate.NewEvaluableExpressionWithFunctions(expressionStr, GetExpressionFunctions()) // Assuming GetExpressionFunctions exists
if errExpr != nil {
    return fmt.Errorf("parsing length expression for {{$field.Name}} ('%s'): %w", expressionStr, errExpr)
}
parameters := map[string]interface{}{
    "s":   s,   // Pass the current struct instance
    "ctx": ctx, // Pass the context
}
evalResult, errEval := expression.Evaluate(parameters)
if errEval != nil {
    return fmt.Errorf("evaluating length expression for {{$field.Name}} ('%s'): %w", expressionStr, errEval)
}
// Convert result to int (handle potential float64 from evaluator)
var size int
switch v := evalResult.(type) {
case float64: size = int(v)
case float32: size = int(v)
case int: size = v
case int64: size = int(v)
case int32: size = int(v)
case uint: size = int(v)
case uint64: size = int(v)
case uint32: size = int(v)
case uint16: size = int(v)
case uint8: size = int(v)
default:
    return fmt.Errorf("length expression for {{$field.Name}} ('%s') evaluated to non-numeric type %T", expressionStr, evalResult)
}
if size < 0 {
     return fmt.Errorf("length expression for {{$field.Name}} ('%s') evaluated to negative size %d", expressionStr, size)
}

b = make([]byte, size) // Uses b var
_, err = io.ReadFull(r, b) // Uses err var
if err != nil {
    return fmt.Errorf("reading {{$field.Name}} (string[dynamic length %s]): %w", expressionStr, err)
}
s.{{$field.Name}} = string(b)
        {{else}}
            // --- Keep existing fixed-length string logic here ---
            {{$length := $field.Length | atoi}}
            // ... (make, ReadFull, assign) ...
        {{end}}
    {{else}}
        // Error: string field needs length
    {{end}}
// --- Replace existing dynamic length logic for []byte ---
{{else if eq $field.Type "[]byte"}}
    {{if $field.Length}}
        {{if isExpressionLength $field}}
// Dynamic length []byte field: {{$field.Name}} using expression: {{$field.Length}}
expressionStr := `{{$field.Length}}` // Use backticks
expression, errExpr := govaluate.NewEvaluableExpressionWithFunctions(expressionStr, GetExpressionFunctions())
if errExpr != nil {
    return fmt.Errorf("parsing length expression for {{$field.Name}} ('%s'): %w", expressionStr, errExpr)
}
parameters := map[string]interface{}{
    "s":   s,
    "ctx": ctx,
}
evalResult, errEval := expression.Evaluate(parameters)
if errEval != nil {
    return fmt.Errorf("evaluating length expression for {{$field.Name}} ('%s'): %w", expressionStr, errEval)
}
// Convert result to int
var size int
switch v := evalResult.(type) {
case float64: size = int(v)
case float32: size = int(v)
case int: size = v
case int64: size = int(v)
case int32: size = int(v)
case uint: size = int(v)
case uint64: size = int(v)
case uint32: size = int(v)
case uint16: size = int(v)
case uint8: size = int(v)
default:
    return fmt.Errorf("length expression for {{$field.Name}} ('%s') evaluated to non-numeric type %T", expressionStr, evalResult)
}
 if size < 0 {
     return fmt.Errorf("length expression for {{$field.Name}} ('%s') evaluated to negative size %d", expressionStr, size)
}

s.{{$field.Name}} = make([]byte, size)
_, err = io.ReadFull(r, s.{{$field.Name}}) // Uses err var
if err != nil {
    return fmt.Errorf("reading {{$field.Name}} ([]byte[dynamic length %s]): %w", expressionStr, err)
}
        {{else}}
            // --- Keep existing fixed-length []byte logic here ---
            {{$length := $field.Length | atoi}}
            // ... (make, ReadFull) ...
        {{end}}
    {{else}}
        // Error: []byte field needs length
    {{end}}
// --- Other field types ---
{{else if eq $field.Type "uint8"}}
    // ... existing uint8 logic ...
// ... etc for other types ...
{{end}} // End of main field type switch
```

**3. Define Helper Functions for Expressions**

*   **Goal:** Provide reusable calculation logic that can be called from YAML expressions.
*   **Change:** Create a function (e.g., in `generator/helpers.go` or even within the generated package file itself, though that's less clean) that returns the map of functions for `govaluate`.

```go
// generator/helpers.go (or similar)
package generator

import (
    "fmt"
    "github.com/knetic/govaluate"
    // Import other necessary packages if helpers need them
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
            if !ok { return nil, fmt.Errorf("arg 1 (width) must be numeric for CalculatePaddedSize") }
            height, ok = args[1].(float64)
            if !ok { return nil, fmt.Errorf("arg 2 (height) must be numeric for CalculatePaddedSize") }
            bitsPerPixel, ok = args[2].(float64)
            if !ok { return nil, fmt.Errorf("arg 3 (bitsPerPixel) must be numeric for CalculatePaddedSize") }
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

        // Add other common helper functions here...
        // "AnotherHelper": func(args ...interface{}) (interface{}, error) { ... },
    }
}

// NOTE: This GetExpressionFunctions needs to be accessible by the generated code.
// You might need to:
// a) Put this function directly into the template (less ideal).
// b) Put it in a shared package imported by the generated code.
// c) Generate this helper function into each output package. (Option B is often cleanest).
```

**4. Update Generator Logic (`generator.go`)**

*   **Goal:** Ensure the `govaluate` import and potentially the `GetExpressionFunctions` helper are available to the generated code.
*   **Change:**
    *   Modify the import calculation logic to add `"github.com/knetic/govaluate"` and the package containing `GetExpressionFunctions` (if separate) whenever `isExpressionLength` is true for any field.
    *   Ensure `GetExpressionFunctions` is accessible (see note in step 3). If placing it in a shared package (e.g., `FormatModules/exprhelpers`), add that import too.

```go
// generator/generator.go (Inside the loop processing structs)

// 4A. Determine required imports...
needsGovaluate := false // Add this flag
needsExprHelpers := false // Add this flag if helpers are in separate package

// Inside the loop iterating through fields:
for _, field := range structDef.Fields {
    // ... existing logic ...
    if isExpressionLength(field) { // Check if any field uses expressions
        needsGovaluate = true
        needsExprHelpers = true // Assume helpers are needed if expressions are used
    }
    // ... existing logic ...
}

// Determine imports based on flags
if needsBinary { requiredImports["encoding/binary"] = true }
if needsFmt || len(structDef.Fields) > 0 { requiredImports["fmt"] = true }
if needsGovaluate { requiredImports["github.com/knetic/govaluate"] = true }
if needsExprHelpers { requiredImports["FormatModules/exprhelpers"] = true } // Adjust path if needed

// ... rest of import list generation ...

// --- Ensure GetExpressionFunctions is available ---
// If you choose option C (generate helpers into each package), you'd add
// the helper function definition to the StructTemplate itself, outside the methods.
// If using option B (shared package), the import logic above handles it.
```

**5. Update YAML Definitions**

*   **Goal:** Use the new expression capabilities.
*   **Change:** Modify `Length` fields in your YAML files (e.g., `bmp.yml`).

```yaml
# bmp.yml (Example for ImageData)
structs:
  FileHeader:
    # ...
  InfoHeader:
    # ... fields like Width, Height, BitsPerPixel ...
  ImageData:
    fields:
      - Name: PixelData
        Type: "[]byte"
        Description: "Actual pixel data (BGR format, padded rows)"
        # Use the helper function, accessing fields from context (ctx)
        # Assumes the InfoHeader struct/pointer is passed as ctx
        Length: "CalculatePaddedSize(ctx.Width, ctx.Height, ctx.BitsPerPixel)"
```

***Self-Correction:*** `govaluate` might require explicit casting or careful handling if `ctx` is `interface{}`. Accessing `ctx.Width` directly might fail. You might need to pass specific types or use type assertions within the expression if the library supports it, or adjust the helper function to handle `interface{}` args more robustly. A common pattern is to pass a `map[string]interface{}` as the context.

***Revised YAML/Context Approach:*** Pass `InfoHeader` specifically.

```yaml
# bmp.yml (Revised)
Length: "CalculatePaddedSize(ctx.Width, ctx.Height, ctx.BitsPerPixel)" # Keep expression simple
```

Calling Code Change (Test): Pass `&readInfoHeader` as `ctx`. The `govaluate` parameters map (`"ctx": ctx`) makes the fields accessible.

**6. Update Calling Code (e.g., `bmp_test.go`)**

*   **Goal:** Pass the necessary context when calling the generated `Read` methods.
*   **Change:** Modify how `Read` is called.

```go
// bmp_test.go (Inside the Read Phase)

// ... readHeader.Read(readFile, nil) // Pass nil context if not needed ...

// Read InfoHeader
if readErr = readInfoHeader.Read(readFile, nil); readErr != nil { // Pass nil context
    readErr = fmt.Errorf("error reading info header: %w", readErr)
    return
}
log.Println("-> InfoHeader read.")

// Prepare context for ImageData - pass the InfoHeader pointer
imageDataCtx := &readInfoHeader // The context IS the InfoHeader

// Create ImageData instance and Read using context
readImageData := bmp.ImageData{}
if readErr = readImageData.Read(readFile, imageDataCtx); readErr != nil { // Pass the context
     readErr = fmt.Errorf("error reading image data: %w", readErr)
     return
}
log.Println("-> ImageData read.")
readPixelData = readImageData.PixelData // Get data from struct

// ... rest of test ...
```

This approach significantly enhances the generator's power, allowing complex, context-dependent length calculations defined declaratively in YAML, while keeping the generated Go code relatively clean by using an expression evaluator. Remember to handle potential errors during expression parsing and evaluation gracefully.
```