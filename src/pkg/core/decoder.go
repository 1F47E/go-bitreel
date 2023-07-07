package core

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"log"
	"os"
	"sort"
	"strings"
	"time"
)

func (c *Core) Decode(dir string) error {
	c.ResetProgress(-1, "Decoding...") // spinner
	// scan the directory for files
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	// get list of files
	filesList := make([]string, 0, len(files))
	for _, file := range files {
		// get file path
		path := dir + "/" + file.Name()
		fmt.Println(path)
		// check if the name has "output_" prefix
		if strings.HasPrefix(file.Name(), "output_") {
			// add to the list
			filesList = append(filesList, path)
		}
	}
	fmt.Println("total files:", len(filesList))
	sort.Strings(filesList)

	// filename with timestamp
	// outputFilename := "tmp/decoded.bin.txt"
	outputFilename := fmt.Sprintf("tmp/decoded_%d.txt", time.Now().Unix())

	// setup progress bar
	c.ResetProgress(len(filesList), "Decoding...")

	// create a output file
	// TODO: get output filename from metadata
	f, err := os.Create(outputFilename)
	if err != nil {
		log.Fatalf("Cannot create file: %s - %v", outputFilename, err)
	}
	defer f.Close()
	var bytesWritten, pixelErrorsCount int
	for _, file := range filesList {
		// fmt.Println("Decoding", file)
		bytes, cnt := decodeFrame(file)
		pixelErrorsCount += cnt

		written, err := f.Write(bytes)
		if err != nil {
			log.Fatal("Cannot write to file:", err)
		}
		bytesWritten += written
		err = c.progress.Add(1)
		if err != nil {
			log.Fatal("Cannot update progress bar:", err)
		}

	}
	log.Printf("\n\nWrote %d bytes to %s\n", bytesWritten, outputFilename)
	if pixelErrorsCount > 0 {
		log.Printf("Pixel errors corrected: %d\n", pixelErrorsCount)
	}
	return nil
}

// TODO: make this a worker
func decodeFrame(filename string) ([]byte, int) {

	// read the image
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal("Cannot open file:", err)
	}
	defer file.Close()
	imgRaw, err := png.Decode(file)
	if err != nil {
		log.Fatal("Cannot decode file:", err)
	}
	// TODO: test later is this needed
	// Create an empty NRGBA image with the same size as the source image.
	img := image.NewNRGBA(imgRaw.Bounds())
	// Draw the source image onto the new NRGBA image.
	draw.Draw(img, img.Bounds(), imgRaw, imgRaw.Bounds().Min, draw.Src)

	var bits [frameSizeBits]bool

	// copy image to bytes
	var k, cntBlack, cntWhite uint
	var pixelErrorsCount int
	for x := 0; x < img.Bounds().Dx(); x += 2 {
		for y := 0; y < img.Bounds().Dy(); y += 2 {
			// error detection
			// count black and white pixels in a 2x2 square
			for i := 0; i < 2; i++ {
				for j := 0; j < 2; j++ {
					col := img.At(x+i, y+j)
					// this will return 0-65535 range
					r, g, b, _ := col.RGBA()
					// shift 8 bits to the right to have 0-255 range
					r8, g8, b8 := r>>8, g>>8, b>>8

					if r8 > 128 && g8 > 128 && b8 > 128 {
						cntWhite++
					} else {
						cntBlack++
					}
				}
			}
			if cntBlack > cntWhite {
				bits[k] = true
				if cntBlack != 4 {
					// fmt.Println("Error at black ", x, y, cntBlack, cntWhite)
					pixelErrorsCount++
				}
			} else {
				bits[k] = false
				if cntWhite != 4 {
					// fmt.Println("Error at white ", x, y, cntBlack, cntWhite)
					pixelErrorsCount++
				}
			}
			cntBlack = 0
			cntWhite = 0
			k++
		}
	}

	// 4k video is 3840x2160 = 8294400 pixels = 2073600 4px blocks
	// every frame should have 2073600 bits
	// convert bits to bytes
	bytes := make([]byte, len(bits)/8)
	for i := 0; i < len(bits); i += 8 {
		var b byte
		for j := 0; j < 8; j++ {
			if bits[i+j] {
				b |= 1 << uint(j)
			}
		}
		bytes[i/8] = b
	}
	// fmt.Println("bytes len:", len(bytes))

	return bytes, pixelErrorsCount

}
