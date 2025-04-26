package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
)

const (
	width  = 256
	height = 256
)

func generateTestBMP() error {
	f, err := os.Create("test.bmp")
	if err != nil {
		return fmt.Errorf("failed to create: %v", err)
	}
	defer f.Close()

	var (
		fileSize   = uint32(14 + 40 + width*height*3) // size of file
		reserved   = uint32(0)                //not used
		dataOffset = uint32(54)               // start of image data
		headerSize = uint32(40)
		planes     = uint16(1)
		bits       = uint16(24)
		compression = uint32(0)
		imageSize    = uint32(width * height * 3)
		xPixelsPerM = int32(2835)
		yPixelsPerM = int32(2835)
		colorsUsed = uint32(0)
		importantColors = uint32(0)
	)

	// BMP Header
	_, err = f.WriteString("BM")
	if err != nil {
		log.Fatalf("WriteString failed: %v", err)
	}
	err = binary.Write(f, binary.LittleEndian, fileSize)
	if err != nil {
		log.Fatalf("binary.Write failed: %v", err)
	}
	err = binary.Write(f, binary.LittleEndian, &reserved)
	if err != nil {
		log.Fatalf("binary.Write failed: %v", err)
	}
	err = binary.Write(f, binary.LittleEndian, &dataOffset)
	if err != nil {
		log.Fatalf("binary.Write failed: %v", err)
	}

	// DIB Header
		err = binary.Write(f, binary.LittleEndian, &headerSize)
	if err != nil {
		log.Fatalf("binary.Write failed: %v", err)
	}
		err = binary.Write(f, binary.LittleEndian, uint32(width))
	if err != nil {
		log.Fatalf("binary.Write failed: %v", err)
	}
		err = binary.Write(f, binary.LittleEndian, uint32(height))
	if err != nil {
		log.Fatalf("binary.Write failed: %v", err)
	}
		err = binary.Write(f, binary.LittleEndian, &planes)
	if err != nil {
		log.Fatalf("binary.Write failed: %v", err)
	}
		err = binary.Write(f, binary.LittleEndian, &bits)
	if err != nil {
		log.Fatalf("binary.Write failed: %v", err)
	}
		err = binary.Write(f, binary.LittleEndian, &compression)
	if err != nil {
		log.Fatalf("binary.Write failed: %v", err)
	}
		err = binary.Write(f, binary.LittleEndian, &imageSize)
	if err != nil {
		log.Fatalf("binary.Write failed: %v", err)
	}
		err = binary.Write(f, binary.LittleEndian, &xPixelsPerM)
	if err != nil {
		log.Fatalf("binary.Write failed: %v", err)
	}
		err = binary.Write(f, binary.LittleEndian, &yPixelsPerM)
	if err != nil {
		log.Fatalf("binary.Write failed: %v", err)
	}
		err = binary.Write(f, binary.LittleEndian, &colorsUsed)
	if err != nil {
		log.Fatalf("binary.Write failed: %v", err)
	}
		err = binary.Write(f, binary.LittleEndian, &importantColors)
	if err != nil {
		log.Fatalf("binary.Write failed: %v", err)
	}

	// Pixel Data
	for i := 0; i < width*height; i++ {
		err = binary.Write(f, binary.LittleEndian, []uint8{255, 255, 255})
		if err != nil {
			log.Fatalf("binary.Write failed: %v", err)
		}
	}
	return nil
}