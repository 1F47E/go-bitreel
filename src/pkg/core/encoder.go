package core

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (c *Core) Encode(path string) error {
	c.ResetProgress(-1, "Encoding...") // set as spinner

	// get only filename
	// open a file
	file, err := os.Open(path)
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
		n, err := file.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Println("Error reading file:", err)
			return err
		}

		// METADATA - CHECKSUM, 64 bits
		// create checksum hash - 8bytes, 64bits
		hasher := fnv.New64a() // FNV-1a hash
		// Pass sliced buffer slice to hasher, no copy
		// also important to pass n - number of bytes read in case of last chunk
		_, err = hasher.Write(buffer[:n])
		if err != nil {
			log.Println("Error writing to hasher:", err)
			return err
		}
		checksum := hasher.Sum64()
		// Convert uint64 to a byte slice
		checksumBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(checksumBytes, checksum)
		// checksumBits := make([]bool, 64)
		checksumBits := bytesToBits(checksumBytes)
		fmt.Println("checksum", checksum)
		printBits(checksumBits)

		// METADATA - timestamp, 64 bits
		timestamp := time.Now().Unix()
		timeBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(timeBytes, uint64(timestamp))
		timeBits := bytesToBits(timeBytes)
		fmt.Println("timestamp", timestamp)
		printBits(timeBits)

		// FILENAME
		filename := path[strings.LastIndex(path, "/")+1:]
		// cut too long filename
		ext := filepath.Ext(filename)
		maxLen := metadataMaxFilenameLen - len(ext) - 2 // 2 for -- separator/ indicator of cut
		if len(filename) > maxLen {
			filename = fmt.Sprintf("%s--%s", filename[:maxLen], ext)
		}
		filenameBits := bytesToBits([]byte(filename))
		fmt.Println("filename", filename)
		printBits(filenameBits)

		// create bits buffer
		var bufferBits [frameBufferSizeBits]bool
		// fill the metadata first
		s := 0
		l := len(checksumBits)
		copy(bufferBits[s:l], checksumBits[:])
		s = l
		l = s + len(timeBits)
		copy(bufferBits[s:l], timeBits[:])
		s = l
		l = s + len(filenameBits)
		copy(bufferBits[s:l], filenameBits[:])
		// panic("debug")
		fmt.Println("metadata header bits:")
		printBits(bufferBits[:metadataSizeBits])

		// start filling data after medata size
		var bitIndex int
		// NOTE: n is the number of bytes read from the file in the last chunk.
		// if not slice with n, the last chunk will be filled with previous data
		// because we reuse the buffer
		for i := 0; i < len(buffer[:n]); i++ {
			// for every byte, range over all bits
			for j := 0; j < 8; j++ {
				bitIndex = metadataSizeBits + i*8 + j
				// get the current bit of the byte
				// buf[i]:     0 1 1 0 1 0 0 1
				// (1<<j):     0 0 0 0 1 0 0 0  (1 has been shifted left 3 times)
				//					   ^
				// --------------------------------
				// result:     0 0 0 0 1 0 0 0  (bitwise AND operation)
				//					   ^
				// write the bit to the buffer
				bufferBits[bitIndex] = (buffer[i] & (1 << uint(j))) != 0
			}
		}
		// bitesWriten := bitIndex - metadataSizeBits

		// send copy of the buffer to pixel processing
		c.Wg.Add(1)
		go func(buf [frameBufferSizeBits]bool, bi int, fn int) {
			// bi - bit index needed to know how many bits to process
			// at the end this number will be < frameBufferSizeBits
			// so we can mark the end of the file
			defer c.Wg.Done()

			_ = c.progress.Add(1)

			// fmt.Println("Proccessing frame in G:", fn)
			now := time.Now()

			// TODO: split into separate goroutines
			// Encoding bits to image - around 1.5s
			fmt.Println("Frame start:", fn)
			img := encodeFrame(buf, bi)
			if bi < frameBufferSizeBits-1 {
				// log the end
				fmt.Println("END OF FILE DETECTED. frame:", fn, ", bits processed:", bi, "buff size:", frameBufferSizeBits)
			}
			fmt.Println("Frame done:", fn, "time:", time.Since(now), "bits processed:", bi)

			// Saving image to file - around 7s
			now = time.Now()
			fmt.Println("Save start:", fn)
			fileName := fmt.Sprintf("tmp/out/out_%08d.png", fn)
			save(fileName, img)
			fmt.Println("Save done. Took time:", time.Since(now))
			// fmt.Println("Frame done:", fn)

		}(bufferBits, bitIndex, frameNumber)

		frameNumber++
	}

	return nil
}

func printBits(bits []bool) {
	for _, b := range bits {
		if b {
			fmt.Print("1")
		} else {
			fmt.Print("0")
		}
	}
	fmt.Println()
}

func bytesToBits(bytes []byte) []bool {
	bits := make([]bool, 8*len(bytes))
	for i, b := range bytes {
		for j := 0; j < 8; j++ {
			bits[i*8+j] = (b & (1 << uint(j))) != 0
		}
	}
	return bits
}

// TODO: make this a worker
func encodeFrame(bits [frameBufferSizeBits]bool, bitIndex int) *image.NRGBA {
	// fmt.Println("Encoding frame")

	// create empty image
	img := image.NewNRGBA(image.Rect(0, 0, frameWidth, frameHeight))

	// generate image
	// fmt.Println("filling the image")
	writeIdx := 0
	var col color.Color
	for x := 0; x < img.Bounds().Dx(); x += 2 {
		for y := 0; y < img.Bounds().Dy(); y += 2 {
			// var col color.Color
			// set red color as default background
			// col := color.NRGBA{255, 0, 0, 255}
			// TODO: check if the end of the file
			// detect end of file
			// if k < bitIndex { // BUG: always true
			// 	col = color.NRGBA{255, 0, 0, 255}
			// } else {
			// 	col = color.NRGBA{0, 255, 0, 255}
			// }

			// detect end of file
			if writeIdx < bitIndex { // BUG: always true
				if bits[writeIdx] {
					// black color
					col = color.NRGBA{0, 0, 0, 255}
				} else {
					// white color
					col = color.NRGBA{255, 255, 255, 255}
				}
			} else {
				col = color.NRGBA{255, 0, 0, 255}
				// fmt.Println("END")
			}
			// Set a 2x2 block of pixels to the color.
			img.Set(x, y, col)
			img.Set(x+1, y, col)
			img.Set(x, y+1, col)
			img.Set(x+1, y+1, col)
			writeIdx++
		}
	}

	// fmt.Println("Encoding frame done")
	return img
}

// TODO: make this a worker
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
