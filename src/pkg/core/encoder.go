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
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func (c *Core) Encode(path string) error {

	// get only filename
	// open a file
	file, err := os.Open(path)
	if err != nil {
		log.Println("Error opening file:", err)
		return err
	}
	defer file.Close()

	// NOTE: read into buffer smaller then a frame to leave space for metadata
	bufferSize := frameSizeBits/8 - metadataSizeBits/8
	readBuffer := make([]byte, bufferSize)

	// Progress bar with frames count progress
	// get total file size
	fileInfo, err := file.Stat()
	if err != nil {
		log.Println("Error getting file info:", err)
		return err
	}
	size := fileInfo.Size()
	estimatedFrames := int(int(size) / len(readBuffer))
	c.ResetProgress(estimatedFrames, "Encoding...") // set as spinner
	_ = c.progress.Add(1)

	// read file by chunks into the buffer
	var frameNumber int
	for {
		// read chunk of bytes into the buffer
		n, err := file.Read(readBuffer)
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
		_, err = hasher.Write(readBuffer[:n])
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
		// fmt.Println("checksum", checksum)
		// printBits(checksumBits)

		// METADATA - timestamp, 64 bits
		timestamp := time.Now().Unix()
		timeBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(timeBytes, uint64(timestamp))
		timeBits := bytesToBits(timeBytes)
		// fmt.Println("timestamp", timestamp)
		// printBits(timeBits)

		// FILENAME
		filename := path[strings.LastIndex(path, "/")+1:]
		// cut too long filename
		ext := filepath.Ext(filename)
		maxLen := metadataMaxFilenameLen - len(ext) - 2 // 2 for -- separator/ indicator of cut
		if len(filename) > maxLen {
			filename = fmt.Sprintf("%s--%s", filename[:maxLen], ext)
		}
		// add marker to the end of the filename so on decoding we know the length
		filename += "/"
		filenameBits := bytesToBits([]byte(filename))
		// fmt.Println("filename", filename)
		// printBits(filenameBits)

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
		// fmt.Println("metadata header bits:")
		// printBits(bufferBits[:metadataSizeBits])

		// start filling data after medata size
		var bitIndex int
		// NOTE: n is the number of bytes read from the file in the last chunk.
		// if not slice with n, the last chunk will be filled with previous data
		// because we reuse the buffer
		for i := 0; i < len(readBuffer[:n]); i++ {
			// for every byte, range over all bits
			for j := 0; j < 8; j++ {
				// shift by metadata size header
				// calc the bit index
				bitIndex = metadataSizeBits + i*8 + j
				// get the current bit of the byte
				// buf[i]:     0 1 1 0 1 0 0 1
				// (1<<j):     0 0 0 0 1 0 0 0  (1 has been shifted left 3 times)
				//					   ^
				// --------------------------------
				// result:     0 0 0 0 1 0 0 0  (bitwise AND operation)
				//					   ^
				// write the bit to the buffer
				bufferBits[bitIndex] = (readBuffer[i] & (1 << uint(j))) != 0
			}
		}
		// bitesWriten := bitIndex - metadataSizeBits

		// send copy of the buffer to pixel processing
		c.Wg.Add(1)
		go func(buf [frameBufferSizeBits]bool, bi int, fn int) {
			// bi - included metadata size
			// at the end this number will be < frameBufferSizeBits
			// so we can mark the end of the file
			defer c.Wg.Done()

			// fmt.Println("Proccessing frame in G:", fn)
			// now := time.Now()

			// TODO: split into separate goroutines
			// Encoding bits to image - around 1.5s
			// fmt.Println("Frame start:", fn)
			img := encodeFrame(buf, bi)
			// limit := frameBufferSizeBits
			// if bi < limit {
			// 	// log the end
			// 	fmt.Println("END OF FILE DETECTED. frame:", fn)
			// 	// lot bits and bytes proccessed on the frame
			// 	fmt.Printf("bits processed: %d, bytes: %d, frameSizeBits: %d, limit bits: %d, limit bytes: %d\n", bi, bi/8, frameBufferSizeBits, limit, limit/8)
			// }
			// fmt.Println("Frame done:", fn, "time:", time.Since(now), "bits processed:", bi)

			// Saving image to file - around 7s
			// now = time.Now()
			// fmt.Println("Save start:", fn)
			fileName := fmt.Sprintf("tmp/out/out_%08d.png", fn)
			err = save(fileName, img)
			if err != nil {
				log.Println("Error saving file:", err)
				// NOTE: no need to continue if we can't save the file
				panic(fmt.Sprintf("EXITING!\n\n\nError saving file: %s", err))
			}
			_ = c.progress.Add(1)
			// fmt.Println("Save done. Took time:", time.Since(now))
			// fmt.Println("Frame done:", fn)

		}(bufferBits, bitIndex, frameNumber)

		frameNumber++
	}
	// wait for all the files to be processed
	c.Wg.Wait()

	// VIDEO ENCODING
	// setup progress bar async, otherwise it wont animate
	c.ResetProgress(-1, "Saving video...")
	done := make(chan bool)
	go func(done <-chan bool) {
		ticker := time.NewTicker(time.Millisecond * 300)
		for {
			select {
			case <-ticker.C:
				_ = c.progress.Add(1)
			case <-done:
				return
			}
		}
	}(done)

	// Call ffmpeg to decode the video into frames
	videoPath := "tmp/out.mov"
	cmdStr := "ffmpeg -y -framerate 30 -i tmp/out/out_%08d.png -c:v prores -profile:v 3 -pix_fmt yuv422p10 " + videoPath
	cmdList := strings.Split(cmdStr, " ")
	// fmt.Println("Running ffmpeg command:", cmdStr)
	cmd := exec.Command(cmdList[0], cmdList[1:]...)
	err = cmd.Run()
	if err != nil {
		panic(fmt.Sprintf("Error running ffmpeg cmd: %s: %s", cmdStr, err))
	}
	done <- true
	_ = c.progress.Clear()

	// clean up tmp/out dir
	err = os.RemoveAll("tmp/out")
	if err != nil {
		panic(fmt.Sprintf("Error removing tmp/out dir: %s", err))
	}
	fmt.Println("\nVideo encoded")

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
			// detect end of file
			if writeIdx <= bitIndex {
				if bits[writeIdx] {
					col = color.NRGBA{0, 0, 0, 255} // black
				} else {
					col = color.NRGBA{255, 255, 255, 255} // white
				}
			} else {
				col = color.NRGBA{255, 0, 0, 255} // red
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
func save(filePath string, img *image.NRGBA) error {
	// make sure dir exists - create all
	err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
	if err != nil {
		log.Println("Cannot create dir:", err)
		// panic(fmt.Sprintf("Cannot create dir: %s", err))
		return fmt.Errorf("Cannot create tmp out dir for path %s: %s", filePath, err)
	}

	imgFile, err := os.Create(filePath)
	defer imgFile.Close()
	if err != nil {
		log.Println("Cannot create file:", err)
		// panic(fmt.Sprintf("Cannot create file: %s", err))
		return fmt.Errorf("Cannot create file: %s", err)
	}
	err = png.Encode(imgFile, img.SubImage(img.Rect))
	if err != nil {
		log.Println("Cannot encode to file:", err)
		// panic(fmt.Sprintf("Cannot encode to file: %s", err))
		return fmt.Errorf("Cannot encode to file: %s", err)
	}
	return nil
}
