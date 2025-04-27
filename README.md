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
