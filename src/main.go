package main

import (
	"bytereel/pkg/core"
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"math"
	"os"
	"sync"

	"github.com/schollz/progressbar/v3"
)

// Amount of bits in 1 4k frame
// 3840*2160/4 = 2073600
const frameSizeBits = 2073600

var wg sync.WaitGroup

func main() {
	// read cmd line args
	args := os.Args[1:]
	if len(args) < 2 {
		log.Fatal("d dir - decode pics in a dir, e file - encode a file")
	}
	command := args[0]
	arg := args[1]

	// get command
	if command == "d" {
		fmt.Println("Decoding")
		err := core.Decode(arg)
		if err != nil {
			log.Fatalf("Error decoding video: %v", err)
		}
	} else if command == "e" {
		fmt.Println("Encoding")
		encode(arg)
		// err := processor.Encode(arg)
		// if err != nil {
		// 	log.Fatalf("Error encoding video: %v", err)
		// }
	}

	wg.Wait()
}

func encode(filename string) {

	// open a file and read to bytes
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// TODO: stream file data, not copy to buffer
	var buf bytes.Buffer
	_, err = io.Copy(&buf, file)
	if err != nil {
		log.Fatal(err)
	}

	b := buf.Bytes()

	// print the size
	fmt.Print("File size: ")
	if len(b) > 1024 {
		fmt.Printf("%d %s\n", len(b)/1024, "KB")
	} else if len(b) > 1024*1024 {
		fmt.Printf("%d %s\n", len(b)/1024/1024, "MB")
	} else if len(b) > 1024*1024*1024 {
		fmt.Printf("%d %s\n", len(b)/1024/1024/1024, "GB")
	} else {
		fmt.Printf("%d %s\n", len(b), "Bytes")
	}

	// calc amount of frames and frame size
	totalFramesCnt := uint64(math.Ceil(float64(len(b)) / float64(frameSizeBits/8)))
	fmt.Println("Frames:", totalFramesCnt)
	totalFrameBytes := int(totalFramesCnt) * frameSizeBits / 8
	// fmt.Println("Frames bytes size:", totalFrameBytes)
	// digits := int(math.Log10(float64(totalFramesCnt))) + 1 // Calculate number of digits
	// fmt.Println("Digits:", digits)

	// init progress bar
	bar := progressbar.NewOptions(int(totalFramesCnt),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetDescription("Encoding..."),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))

	bitIndex := 0
	var bitsBuffer [frameSizeBits]bool
	// range over all frames, more then file len!
	for i := 0; i < totalFrameBytes; i++ {
		// for every byte, range over all bits
		for j := 0; j < 8; j++ {
			frameNumber := i / (frameSizeBits / 8)
			shift := frameNumber * frameSizeBits
			bitIndex = i*8 + j - shift // should reset to 0 on every frame
			// if we have more bytes than needed, fill the rest with 0
			if i >= len(b) {
				bitsBuffer[bitIndex] = false
			} else {
				bitsBuffer[bitIndex] = (b[i] & (1 << uint(j))) != 0
			}

			// detect the end of the file or the end of the frame
			// proccess the image, save, reset the buffer
			// send a copy of bits buffer to goroutine to proccess
			// panic on errors - missed frames are not allowed
			if bitIndex == len(bitsBuffer)-1 || bitIndex == len(b)*8-1 {
				// create filename
				// prefix filename with dynamic leading zeroes
				// fileName := fmt.Sprintf("tmp/out_%0"+strconv.Itoa(digits)+"d.png", frameNumber)
				fileName := fmt.Sprintf("tmp/out/out_%d.png", frameNumber)

				wg.Add(1)
				go func(bitsBuffer [frameSizeBits]bool, fn string) {
					defer wg.Done()
					// fmt.Println("Proccessing frame in G:", fn)
					img := encodeFrame(bitsBuffer)
					save(fileName, img)
					// fmt.Println("Frame done:", fn)
					bar.Add(1)
				}(bitsBuffer, fileName)
			}
		}
	}
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
