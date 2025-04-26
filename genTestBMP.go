package main

import (
	"encoding/binary"
	"log"
	"os"
)

const (
	width  = 256
	height = 256
)

// Helper function to handle writing and potential errors
func writeBinary(f *os.File, data interface{}) error {
	return binary.Write(f, binary.LittleEndian, data)
}

func getTestBMP() { // Changed signature: removed 'error' return type
	f, err := os.Create("test.bmp")
	if err != nil {
		// Handle error inside main: log and exit
		log.Fatalf("failed to create test.bmp: %v", err)
	}
	// Use a named variable for the close error to check it properly
	var closeErr error
	defer func() {
		closeErr = f.Close()
		if closeErr != nil {
			// Log the close error if it happens
			log.Printf("Error closing test.bmp: %v", closeErr)
		}
	}()

	// Define header values using constants and calculations
	const fileHeaderSize = 14
	const dibHeaderSize = 40 // This is implicitly 'int' here
	const bytesPerPixel = 3 // Assuming 24-bit color
	imageDataSize := uint32(width * height * bytesPerPixel)
	fileSize := uint32(fileHeaderSize + dibHeaderSize + imageDataSize)
	dataOffset := uint32(fileHeaderSize + dibHeaderSize)

	// BMP File Header
	if _, err = f.WriteString("BM"); err != nil {
		log.Fatalf("WriteString failed for signature: %v", err)
	}
	if err = writeBinary(f, fileSize); err != nil {
		log.Fatalf("binary.Write failed for fileSize: %v", err)
	}
	if err = writeBinary(f, uint16(0)); err != nil { // Reserved 1 (uint16)
		log.Fatalf("binary.Write failed for reserved1: %v", err)
	}
	if err = writeBinary(f, uint16(0)); err != nil { // Reserved 2 (uint16)
		log.Fatalf("binary.Write failed for reserved2: %v", err)
	}
	if err = writeBinary(f, dataOffset); err != nil {
		log.Fatalf("binary.Write failed for dataOffset: %v", err)
	}

	// DIB Header (BITMAPINFOHEADER)
	// --- FIX HERE: Explicitly cast dibHeaderSize to uint32 ---
	if err = writeBinary(f, uint32(dibHeaderSize)); err != nil {
		log.Fatalf("binary.Write failed for dibHeaderSize: %v", err)
	}
	// --- End of Fix ---
	if err = writeBinary(f, int32(width)); err != nil { // Width (int32)
		log.Fatalf("binary.Write failed for width: %v", err)
	}
	if err = writeBinary(f, int32(height)); err != nil { // Height (int32)
		log.Fatalf("binary.Write failed for height: %v", err)
	}
	if err = writeBinary(f, uint16(1)); err != nil { // Planes (uint16)
		log.Fatalf("binary.Write failed for planes: %v", err)
	}
	if err = writeBinary(f, uint16(24)); err != nil { // Bits per pixel (uint16)
		log.Fatalf("binary.Write failed for bits: %v", err)
	}
	if err = writeBinary(f, uint32(0)); err != nil { // Compression (uint32) - BI_RGB
		log.Fatalf("binary.Write failed for compression: %v", err)
	}
	if err = writeBinary(f, imageDataSize); err != nil { // Image Size (uint32)
		log.Fatalf("binary.Write failed for imageSize: %v", err)
	}
	if err = writeBinary(f, int32(2835)); err != nil { // X Pixels Per Meter (int32)
		log.Fatalf("binary.Write failed for xPixelsPerM: %v", err)
	}
	if err = writeBinary(f, int32(2835)); err != nil { // Y Pixels Per Meter (int32)
		log.Fatalf("binary.Write failed for yPixelsPerM: %v", err)
	}
	if err = writeBinary(f, uint32(0)); err != nil { // Colors Used (uint32)
		log.Fatalf("binary.Write failed for colorsUsed: %v", err)
	}
	if err = writeBinary(f, uint32(0)); err != nil { // Important Colors (uint32)
		log.Fatalf("binary.Write failed for importantColors: %v", err)
	}

	// Pixel Data (BGR order) - Writing white pixels
	pixel := []byte{255, 255, 255} // White pixel (BGR)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if _, err = f.Write(pixel); err != nil {
				log.Fatalf("failed writing pixel data at (%d, %d): %v", x, y, err)
			}
		}
	}

	// Check the defer'd close error before exiting
	if closeErr != nil {
		// Log the error and exit non-zero
		log.Fatalf("File close error occurred: %v", closeErr)
	}

	log.Println("Successfully generated test.bmp")
}

