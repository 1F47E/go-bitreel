package core

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/schollz/progressbar/v3"
)

// Amount of bits in 1 4k frame
// 3840*2160/4 = 2073600
const frameSizeBits = 2073600

type Core struct {
	progress *progressbar.ProgressBar
	Wg       sync.WaitGroup
}

func NewCore() *Core {
	return &Core{
		progress: nil,
		Wg:       sync.WaitGroup{},
	}
}

// reinit progress bar becuase of some bug
func (c *Core) ResetProgress(max int, desc string) {
	c.progress = progressbar.NewOptions(max,
		progressbar.OptionSetDescription(desc),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionSetDescription(desc))
}

func (c *Core) Encode(filename string) error {
	c.ResetProgress(-1, "Encoding...") // set as spinner

	// open a file
	file, err := os.Open(filename)
	if err != nil {
		log.Println("Error opening file:", err)
		return err
	}
	defer file.Close()

	// read file by chunks into the buffer
	bufferSize := frameSizeBits / 8
	buffer := make([]byte, bufferSize)

	var frameNumber int
	for {
		// read chunk of bytes into the buffer
		_, err := file.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Println("Error reading file:", err)
			return err
		}

		// fill the bits buffer
		var bufferBits [frameSizeBits]bool
		bitIndex := 0
		for i := 0; i < len(buffer); i++ {
			// for every byte, range over all bits
			for j := 0; j < 8; j++ {
				bitIndex = i*8 + j
				// get the current bit of the byte
				// buf[i]:     0 1 1 0 1 0 0 1
				// (1<<j):     0 0 0 0 1 0 0 0  (1 has been shifted left 3 times)
				//					   ^
				// --------------------------------
				// result:     0 0 0 0 1 0 0 0  (bitwise AND operation)
				//					   ^
				bufferBits[bitIndex] = (buffer[i] & (1 << uint(j))) != 0
			}
		}

		// send copy of the buffer to pixel processing
		c.Wg.Add(1)
		go func(buf [frameSizeBits]bool, fn int) {
			defer c.Wg.Done()

			_ = c.progress.Add(1)

			// fmt.Println("Proccessing frame in G:", fn)
			now := time.Now()

			// TODO: split into separate goroutines
			// Encoding bits to image - around 1.5s
			fmt.Println("Frame start:", fn)
			img := encodeFrame(buf)
			fmt.Println("Frame done:", fn, "time:", time.Since(now))

			// Saving image to file - around 7s
			now = time.Now()
			fmt.Println("Save start:", fn)
			fileName := fmt.Sprintf("tmp/out/out_%08d.png", fn)
			save(fileName, img)
			fmt.Println("Save done. Took time:", time.Since(now))
			// fmt.Println("Frame done:", fn)

		}(bufferBits, frameNumber)

		frameNumber++
	}

	return nil
}

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

func encodeFrame(bits [frameSizeBits]bool) *image.NRGBA {
	// fmt.Println("Encoding frame")

	// generate an image
	img := image.NewNRGBA(image.Rect(0, 0, 3840, 2160)) // 4K resolution

	// generate image
	// fmt.Println("filling the image")
	k := 0
	var col color.Color
	for x := 0; x < img.Bounds().Dx(); x += 2 {
		for y := 0; y < img.Bounds().Dy(); y += 2 {
			// var col color.Color
			// set red color as default background
			// col := color.NRGBA{255, 0, 0, 255}
			// TODO: check if the end of the file
			if k < len(bits) { // BUG: always true
				if bits[k] {
					// col = color.Black
					col = color.NRGBA{0, 0, 0, 255}
				} else {
					// col = color.White
					col = color.NRGBA{255, 255, 255, 255}
				}
				k++
			} else {
				col = color.NRGBA{255, 0, 0, 255}
				fmt.Println("END")
			}
			// Set a 2x2 block of pixels to the color.
			img.Set(x, y, col)
			img.Set(x+1, y, col)
			img.Set(x, y+1, col)
			img.Set(x+1, y+1, col)
		}
	}

	// fmt.Println("Encoding frame done")
	return img
}

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

func save(filePath string, img *image.NRGBA) {
	imgFile, err := os.Create(filePath)
	defer imgFile.Close()
	if err != nil {
		log.Println("Cannot create file:", err)
		panic(fmt.Sprintf("Cannot create file: %s", err))
	}
	err = png.Encode(imgFile, img.SubImage(img.Rect))
	if err != nil {
		log.Println("Cannot encode to file:", err)
		panic(fmt.Sprintf("Cannot encode to file: %s", err))
	}
}
