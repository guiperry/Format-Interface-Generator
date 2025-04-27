# FormatModule - Binary Format Code Generator

## Overview

FormatModule is a Go application designed to automate the creation of Go code for reading and writing binary file formats. By defining the structure of a file format in a simple YAML file, this tool generates Go structs along with corresponding `Read` and `Write` methods, significantly reducing boilerplate code and potential errors when dealing with binary data serialization and deserialization.

## Features

*   **YAML-Based Definitions:** Define complex binary formats using an intuitive YAML structure.
*   **Go Code Generation:** Automatically generates Go struct definitions based on the YAML.
*   **Read/Write Methods:** Generates `Read(io.Reader, interface{}) error` and `Write(io.Writer) error` methods for each struct, handling the binary encoding/decoding according to the definition.
*   **Type Handling:** Supports standard fixed-size Go types (e.g., `uint8`, `uint16`, `int32`, `float64`) using `encoding/binary`.
*   **String and Byte Slices:** Handles fixed-length `string` and `[]byte` fields.
*   **Dynamic Lengths:** Supports `string` and `[]byte` fields whose lengths are determined at runtime using Go expressions (e.g., based on previously read fields or context).
*   **Context Passing:** `Read` methods accept an `interface{}` context, allowing dynamic length calculations based on data external to the current struct (e.g., a previously read header).
*   **Conditional Fields:** Define fields that are only read or written if a specific Go expression (referencing other fields) evaluates to true.
*   **YAML Validation & Reformation:** Includes a bootstrap phase that:
    *   Validates YAML definitions against expected structure and rules.
    *   Handles case-insensitivity for field attribute keys (e.g., `Name` vs `name`).
    *   Replaces placeholder `length: ...` with `length: NEEDS_MANUAL_LENGTH` to flag areas requiring manual logic.
    *   Saves a validated/reformed version of the YAML for use during code generation.
*   **Configuration Management:** Uses a `formats.json` file to manage configured formats.
*   **Test Script Generation:** Optionally generates a basic `_test.go` file template for each format, providing a starting point for testing the generated code.
*   **Organized Structure:** Uses dedicated directories for source YAML (`sources/`) and generated code (`formats/`).

## Prerequisites

*   Go (version 1.18 or later recommended)

## Setup

1.  Clone the repository:
    ```bash
    git clone <your-repository-url>
    cd FormatModule
    ```
2.  Ensure dependencies are downloaded (if any beyond the standard library and included vendors):
    ```bash
    go mod tidy
    ```

## Usage

The tool operates in two main modes: Bootstrap and Generation.

### 1. Defining a Format (YAML)

Create a `.yml` file (e.g., `myformat.yml`) inside the `sources/` directory. Define your format structure:

```yaml
# sources/myformat.yml

# Optional: Top-level name and description
# name: MyFormat
# description: Describes my custom binary format.

structs:
  MyHeader:
    fields:
      - name: signature       # Use lowercase keys
        type: string
        length: 4           # Fixed length string
        description: "File signature (e.g., 'MYF\\x00')"
      - name: version
        type: uint16
        description: "Format version"
      - name: payloadSize
        type: uint32
        description: "Size of the upcoming payload"
      - name: flags
        type: uint8
        description: "Various flags"
        tags: 'json:"format_flags"' # Optional Go struct tags

  MyPayload:
    fields:
      - name: data
        type: "[]byte"
        description: "The main data payload"
        # Dynamic length based on header field passed via context
        length: "ctx.payloadSize - 4" # Assumes MyHeader passed as ctx
      - name: checksum
        type: uint32
        description: "Optional checksum"
        # Conditional field based on flags in the *same* struct
        condition: "(s.flags & 0x01) != 0" # Example: Read only if first flag bit is set

  # Add other structs as needed...
```

## YAML Field Attributes:

*   **`name`:** (Required) The name of the field in the generated Go struct.
*   **`type`:** (Required) The Go type (e.g., `uint8`, `string`, `[]byte`, `MyOtherStruct`).
*   **`description`:** (Optional) A comment added to the generated struct field.
*   **`length`:** (Required for `string`, `[]byte`) Specifies the length.
    *   Can be a positive integer (e.g., `5`).
    *   Can be a Go expression string evaluating to an integer. Use `s.` to refer to fields within the same struct (e.g., `"s.Count * 4"`). Use `ctx.` to refer to fields from the context passed to the `Read` method (e.g., `"ctx.HeaderSize - 2"`).
    *   Use `NEEDS_MANUAL_LENGTH` if the length requires complex logic not expressible here; the generator will insert TODO comments.
*   **`condition`:** (Optional) A Go expression string. If present, the field is only read/written if the condition evaluates to true at runtime. Use `s.` to refer to fields within the same struct.
*   **`tags`:** (Optional) A string containing Go struct tags to be added to the generated field (e.g., ``json:"myName" xml:"name"``).

## Bootstrapping Formats

The bootstrap phase validates your source YAML, reforms it (handling case and placeholders), saves the reformed version, and updates the `formats.json` configuration.

*   **Run the bootstrap command:**

    ```bash
    go run . -bootstrap
    ```

*   The tool will scan the `sources/` directory.
*   You will be prompted to select which YAML file(s) to bootstrap.
*   For each selected file:
    *   Validation and reformation occur. Errors will be reported.
    *   The reformed YAML is saved (e.g., `formats/myformat/myformat.yml`).
    *   `formats.json` is updated with the format's configuration.

## Generating Code

This phase generates the Go code based on the reformed YAML files listed in `formats.json`.

*   **Run the generation command:**

    ```bash
    go run .
    ```

*   The tool will read `formats.json`.
*   You will be prompted to select which configured format(s) to generate code for.
*   For each selected format:
    *   Existing `.go` files in the target directory (e.g., `formats/myformat/`) are removed (the reformed YAML is kept).
    *   The generator reads the reformed YAML (e.g., `formats/myformat/myformat.yml`).
    *   Go files (e.g., `formats/myformat/myheader.go`, `formats/myformat/mypayload.go`) are generated.
    *   You will be asked if you want to generate a basic test script.

## Generating Test Scripts (Optional)

If you answer `'y'` during the code generation phase:

*   A basic test file (e.g., `formats/myformat/myformat_test.go`) is generated.
*   This file uses the first struct found in the YAML as an example and follows a `Write -> Read -> Verify` pattern.
*   **Important:** You *must* adapt this generated test file. Fill in realistic sample data, implement the correct sequence of `Write` and `Read` calls for your specific format, and add appropriate verification logic using `reflect.DeepEqual` or `bytes.Equal`.

**Directory Structure**

*   `main.go`: Main application entry point, handles flags and orchestrates bootstrap/generation.
*   `validator.go`: Contains YAML validation and reformation logic.
*   `reset_generator.go`: Contains logic to clean generated files before regeneration.
*   `generator/`: Package containing the core code generation logic.
    *   `generator.go`: Parses YAML and executes templates.
    *   `templates.go`: Go code template for generated structs and methods.
    *   `test_template.go`: Go code template for generated test files.
    *   `helpers.go`: Contains helper functions (e.g., `GetExpressionFunctions`) used by generated code.
*   `application_structs/`: Defines the Go structs that represent the YAML structure.
*   `sources/`: (You create this) Place your source `.yml` format definition files here.
*   `formats/`: (Generated) Contains subdirectories for each generated format's Go code and reformed YAML.
    *   `formats/myformat/`: Example directory for `myformat`.
        *   `myformat.yml`: The validated and reformed YAML file.
        *   `myheader.go`: Generated code for `MyHeader`.
        *   `mypayload.go`: Generated code for `MyPayload`.
        *   `myformat_test.go`: Generated test script template.
*   `formats.json`: (Generated/Updated) Configuration file listing known formats, their source YAML paths, and output directories.

**Limitations and TODOs**

*   **Complex Repeating Structures:** The generator currently doesn't automatically handle fields that represent a variable number of repeating structures based on a count field. This often requires manual loops within the `Read`/`Write` methods after generation.
*   **End-of-Stream Reads:** Formats where data continues until the end of the stream or a specific marker (like JPEG entropy-coded data) cannot be handled automatically by the `length` attribute and require manual implementation.
*   **Advanced Validation:** The static validation of `length` and `condition` expressions is basic. Complex expressions might pass validation but fail at runtime if incorrect. Runtime error handling in generated code is present but could be enhanced.
*   **Error Handling:** While basic error checking is generated, more nuanced error handling might be needed for production use.

**Contributing**

Contributions are welcome! Please feel free to submit issues or pull requests.
