package main

import (
	"bytes" // Import bytes package for comparison
	"fmt"
	"io" // Import io for ReadFull
	"log"
	"os"
	"reflect" // Import reflect package for DeepEqual
	"testing" // Import testing package

	"FormatModules/fullbmp" // Assuming this is the correct import path
)

// Constants for test data
const (
	testWidth  = 2 // Keep it small for testing
	testHeight = 2
	testBits   = uint16(24) // 24 bits per pixel (RGB)
)

// TestGeneratedCode uses Go's testing framework.
// It writes a BMP file using generated structs and then reads it back for verification.
func TestGeneratedCode(t *testing.T) { // Changed to a test function
	log.Println("Starting generated code test (Write -> Read -> Verify)...")
	testFilename := "test_write_read.bmp"
	// Clean up the test file afterwards
	//defer os.Remove(testFilename)

	// --- Test Data Setup ---
	// Calculate sizes based on constants
	// Calculate pixel data size with padding (each row must be multiple of 4 bytes)
	bytesPerRow := testWidth * int(testBits/8)
	paddingPerRow := (4 - (bytesPerRow % 4)) % 4
	paddedRowSize := bytesPerRow + paddingPerRow
	pixelDataSize := uint32(testHeight * paddedRowSize)
	// Assuming standard header sizes based on common BMP format
	dibHeaderSize := uint32(40) // Standard BITMAPINFOHEADER size - Check if your YAML defines this struct differently
	fileHeaderSize := uint32(14)
	dataOffset := fileHeaderSize + dibHeaderSize // Adjust if your InfoHeader size differs
	fileSize := dataOffset + pixelDataSize

	// Create instances with test data
	originalHeader := fullbmp.FileHeader{
		Signature:  "BM",
		FileSize:   fileSize,
		Reserved1:  0, // Assuming Reserved1/2 based on typical BMP structure
		Reserved2:  0,
		DataOffset: dataOffset,
	}

	// Initialize InfoHeader with all standard fields
	originalInfoHeader := fullbmp.InfoHeader{
		HeaderSize:      dibHeaderSize, // Standard BITMAPINFOHEADER size (40)
		Width:          uint32(testWidth),
		Height:         uint32(testHeight),
		Planes:         1,
		BitsPerPixel:   testBits,
		Compression:    0, // BI_RGB (uncompressed)
		ImageSize:      pixelDataSize,
		XPixelsPerMeter: 2835, // Example value (~72 DPI)
		YPixelsPerMeter: 2835, // Example value (~72 DPI)
		ColorsUsed:     0,    // 0 for 24-bit
		ImportantColors: 0,   // 0 for all colors important
	}

	// Create simple pixel data
	// Create pixel data with proper padding (2 bytes per row for 2x2 24bpp image)
	originalPixelData := []byte{
		0, 0, 255, // Pixel (0,0) Red (BGR order)
		0, 255, 0, // Pixel (1,0) Green
		0, 0, // Padding for first row
		255, 0, 0, // Pixel (0,1) Blue
		255, 255, 255, // Pixel (1,1) White
		0, 0, // Padding for second row
	}
	if uint32(len(originalPixelData)) != pixelDataSize {
		// Use t.Fatalf for test failures
		t.Fatalf("internal test error: pixel data size mismatch (%d != %d)", len(originalPixelData), pixelDataSize)
	}

	// --- Write Phase ---
	log.Println("Phase 1: Writing test file", testFilename)
	writeFile, err := os.Create(testFilename)
	if err != nil {
		t.Fatalf("error creating write file '%s': %v", testFilename, err)
	}

	var writeErr error
	func() {
		defer func() {
			if cerr := writeFile.Close(); cerr != nil && writeErr == nil {
				writeErr = fmt.Errorf("error closing write file: %w", cerr)
			}
		}()

		if writeErr = originalHeader.Write(writeFile); writeErr != nil {
			writeErr = fmt.Errorf("error writing header: %w", writeErr)
			return
		}
		log.Println("-> Header written.")

		if writeErr = originalInfoHeader.Write(writeFile); writeErr != nil {
			writeErr = fmt.Errorf("error writing info header: %w", writeErr)
			return
		}
		log.Println("-> InfoHeader written.")

		bytesWritten, writeErr := writeFile.Write(originalPixelData)
		if writeErr != nil {
			writeErr = fmt.Errorf("error writing pixel data: %w", writeErr)
			return
		}
		if bytesWritten != len(originalPixelData) {
			writeErr = fmt.Errorf("incomplete pixel data write (%d / %d bytes)", bytesWritten, len(originalPixelData))
			return
		}
		log.Printf("-> PixelData written (%d bytes).", bytesWritten)

	}()

	if writeErr != nil {
		t.Fatalf("Write phase failed: %v", writeErr)
	}
	log.Println("Phase 1: Write completed successfully.")

	// --- Read Phase ---
	log.Println("Phase 2: Reading test file", testFilename)
	readFile, err := os.Open(testFilename)
	if err != nil {
		t.Fatalf("error opening read file '%s': %v", testFilename, err)
	}

	var readErr error
	readHeader := fullbmp.FileHeader{}
	readInfoHeader := fullbmp.InfoHeader{}
	var readPixelData []byte

	func() {
		defer func() {
			if cerr := readFile.Close(); cerr != nil && readErr == nil {
				readErr = fmt.Errorf("error closing read file: %w", cerr)
			}
		}()

		if readErr = readHeader.Read(readFile); readErr != nil {
			readErr = fmt.Errorf("error reading header: %w", readErr)
			return
		}
		log.Println("-> Header read.")

		if readErr = readInfoHeader.Read(readFile); readErr != nil {
			readErr = fmt.Errorf("error reading info header: %w", readErr)
			return
		}
		log.Println("-> InfoHeader read.")

		// Calculate expected size including padding (same logic as write phase)
		readBytesPerRow := int(readInfoHeader.Width) * int(readInfoHeader.BitsPerPixel/8)
		readPaddingPerRow := (4 - (readBytesPerRow % 4)) % 4
		readPaddedRowSize := readBytesPerRow + readPaddingPerRow
		expectedPixelDataSize := int(readInfoHeader.Height) * readPaddedRowSize
		if expectedPixelDataSize <= 0 {
			readErr = fmt.Errorf("invalid calculated pixel data size (with padding): %d", expectedPixelDataSize)
			return
		}
		readPixelData = make([]byte, expectedPixelDataSize)
		bytesRead, readErr := io.ReadFull(readFile, readPixelData)
		if readErr != nil {
			if readErr == io.ErrUnexpectedEOF || readErr == io.EOF {
				readErr = fmt.Errorf("error reading pixel data: unexpected end of file (read %d / %d bytes): %w", bytesRead, expectedPixelDataSize, readErr)
			} else {
				readErr = fmt.Errorf("error reading pixel data: %w", readErr)
			}
			return
		}
		log.Printf("-> PixelData read (%d bytes).", bytesRead)

	}()

	if readErr != nil {
		t.Fatalf("Read phase failed: %v", readErr)
	}
	log.Println("Phase 2: Read completed successfully.")

	// --- Verification Phase ---
	log.Println("Phase 3: Verifying data...")

	// Compare Headers
	if !reflect.DeepEqual(originalHeader, readHeader) {
		t.Errorf("Verification FAILED: Headers do not match.\nOriginal: %+v\nRead:     %+v", originalHeader, readHeader)
	} else {
		log.Println("-> Header verified.")
	}

	// Compare InfoHeaders
	if !reflect.DeepEqual(originalInfoHeader, readInfoHeader) {
		t.Errorf("Verification FAILED: InfoHeaders do not match.\nOriginal: %+v\nRead:     %+v", originalInfoHeader, readInfoHeader)
	} else {
		log.Println("-> InfoHeader verified.")
	}

	// Compare Pixel Data
	if !bytes.Equal(originalPixelData, readPixelData) {
		// Use t.Errorf for test failures
		t.Errorf("Verification FAILED: PixelData does not match.")
		if len(originalPixelData) != len(readPixelData) {
			log.Printf("  Length Mismatch: Original=%d, Read=%d", len(originalPixelData), len(readPixelData))
		}
	} else {
		log.Println("-> PixelData verified.")
	}

	// t.Error or t.Errorf would have already marked the test as failed if verification failed.
	log.Println("Phase 3: Verification finished.")
	log.Println("Generated code test completed.")
}
