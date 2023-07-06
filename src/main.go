package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"os"
	"path/filepath"
)

func main() {
	// read cmd line args
	args := os.Args[1:]
	if len(args) == 0 {
		log.Fatal("No file given")
	}
	file := args[0]

	// get file extension via filePath
	ext := filepath.Ext(file)
	if ext == ".png" {
		decode(file)
	} else {
		encode(file)
	}
}

func decode(filename string) {
	fmt.Println("Decoding")

	// read the image
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal("Cannot open file:", err)
	}
	defer file.Close()
	img, err := png.Decode(file)
	if err != nil {
		log.Fatal("Cannot decode file:", err)
	}

	bits := make([]bool, 0)

	// copy image to bytes
	// TODO: read all 4 pixels and decide if black or white on majority
	for x := 0; x < img.Bounds().Dx(); x += 2 {
		for y := 0; y < img.Bounds().Dy(); y += 2 {
			// get color of pixel
			col := img.At(x, y)
			r, g, b, _ := col.RGBA()

			// white = {255 255 255 255}
			// black = {0 0 0 255}
			isBlack := r == 0 && g == 0 && b == 0
			isWhite := r == 0xFFFF && g == 0xFFFF && b == 0xFFFF

			if isBlack {
				bits = append(bits, true)
			} else if isWhite {
				bits = append(bits, false)
			}
		}
		// fmt.Println("bits len:", len(bits))
		// fmt.Println("bits:", bits)
	}
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

	// write bytes to file
	ext := filepath.Ext(filename)
	outputFilename := fmt.Sprintf("decoded%s", ext)
	file, err = os.Create(outputFilename)
	if err != nil {
		log.Fatal("Cannot create file:", err)
	}
	defer file.Close()
	_, err = file.Write(bytes)
	if err != nil {
		log.Fatal("Cannot write to file:", err)
	}
	log.Println("Done")

}

func encode(filename string) {
	fmt.Println("Encoding")

	// open a file and read to bytes
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, file)
	if err != nil {
		log.Fatal(err)
	}

	b := buf.Bytes()
	fmt.Println("Read bytes:", len(b))
	bits := make([]bool, len(b)*8)
	for i := 0; i < len(b); i++ {
		// for every byte convert it to 8 bits
		for j := 0; j < 8; j++ {
			bits[i*8+j] = (b[i] & (1 << uint(j))) != 0
		}
	}
	fmt.Println("bits done")

	// TODO: encode frame in a func, allow many frames

	// generate an image
	img := image.NewNRGBA(image.Rect(0, 0, 3840, 2160)) // 4K resolution

	// generate image
	fmt.Println("filling the image")
	// rand.Seed(time.Now().UnixNano())
	k := 0
	for x := 0; x < img.Bounds().Dx(); x += 2 {
		for y := 0; y < img.Bounds().Dy(); y += 2 {
			var col color.Color
			if k < len(bits) {
				if bits[k] {
					col = color.Black
				} else {
					col = color.White
				}
				k++
			} else {
				// red color as end
				col = color.RGBA{255, 0, 0, 255}
			}
			// // generate random int
			// if rand.Intn(2)%2 == 0 {
			// 	col = color.White
			// } else {
			// 	col = color.Black
			// }
			// Set a 2x2 block of pixels to the color.
			img.Set(x, y, col)
			img.Set(x+1, y, col)
			img.Set(x, y+1, col)
			img.Set(x+1, y+1, col)
		}
	}
	save("test.png", img)
}

func save(filePath string, img *image.NRGBA) {
	imgFile, err := os.Create(filePath)
	defer imgFile.Close()
	if err != nil {
		log.Println("Cannot create file:", err)
	}
	png.Encode(imgFile, img.SubImage(img.Rect))
}
