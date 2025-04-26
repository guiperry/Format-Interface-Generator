// bmp_stubs.go - Stub file
// IMPORTANT: This file contains stub definitions for compile-time checks.
//            It should NOT be removed by the generator.
package bmp

import "io"


// --- Stub for FileHeader ---
type FileHeader struct {
    
    Signature string // BMP Signature (BM)
    
    FileSize uint32 // Total file size
    
    Reserved1 uint16 // Reserved (0)
    
    Reserved2 uint16 // Reserved (0)
    
    DataOffset uint32 // Offset to image data
    
}

// Dummy Read method
func (s *FileHeader) Read(r io.Reader) error {
	return nil
}

// Dummy Write method
func (s *FileHeader) Write(w io.Writer) error {
	return nil
}



// --- Stub for InfoHeader ---
type InfoHeader struct {
    
    HeaderSize uint32 // Size of the information header (40)
    
    Width uint32 // Image width
    
    Height uint32 // Image height
    
    Planes uint16 // Number of color planes (always 1)
    
    BitsPerPixel uint16 // Bits per pixel (e.g., 24 for RGB)
    
    Compression uint32 // Compression method (0 for uncompressed)
    
    ImageSize uint32 // Size of the raw pixel data (can be 0 for uncompressed)
    
    XPixelsPerMeter int32 // Horizontal resolution (pixels per meter)
    
    YPixelsPerMeter int32 // Vertical resolution (pixels per meter)
    
    ColorsUsed uint32 // Number of colors in the color palette (0 for true-color images)
    
    ImportantColors uint32 // Number of important colors (0 for all colors important)
    
}

// Dummy Read method
func (s *InfoHeader) Read(r io.Reader) error {
	return nil
}

// Dummy Write method
func (s *InfoHeader) Write(w io.Writer) error {
	return nil
}



// --- Stub for ImageData ---
type ImageData struct {
    
    PixelData []byte // RGB pixel data
    
}

// Dummy Read method
func (s *ImageData) Read(r io.Reader) error {
	return nil
}

// Dummy Write method
func (s *ImageData) Write(w io.Writer) error {
	return nil
}



