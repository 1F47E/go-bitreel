package core

import (
	"bytereel/pkg/job"
	"bytereel/pkg/meta"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

func workerEncode(g int, jobs <-chan job.Job) {
	fmt.Printf("Goroutine %d started\n", g)
	defer log.Printf("Goroutine %d finished\n", g)

	var err error
	for {
		select {
		case j := <-jobs:
			// TODO: detect by the size of the slice in a job is it last chunk or not

			fmt.Printf("%d G got job %+v\n", g, j)

			bufferBits := make([]bool, sizeFrame*8)
			metadataSizeBits := sizeMetadata * 8
			var bitIndex int
			// NOTE: n is the number of bytes read from the file in the last chunk.
			// if not slice with n, the last chunk will be filled with previous data
			// because we reuse the buffer
			for i := 0; i < len(j.Buffer); i++ {
				// for every byte, range over all bits
				for k := 0; k < 8; k++ {
					// shift by metadata size header
					// calc the bit index
					bitIndex = metadataSizeBits + i*8 + k
					// get the current bit of the byte
					// buf[i]:     0 1 1 0 1 0 0 1
					// (1<<j):     0 0 0 0 1 0 0 0  (1 has been shifted left 3 times)
					//					   ^
					// --------------------------------
					// result:     0 0 0 0 1 0 0 0  (bitwise AND operation)
					//					   ^
					// write the bit to the buffer
					bufferBits[bitIndex] = (j.Buffer[i] & (1 << uint(k))) != 0
				}
			}
			// bitesWriten := bitIndex - metadataSizeBits

			// pixel processing
			// c.Wg.Add(1)
			// go func(buf [frameBufferSizeBits]bool, bi int, fn int) {
			// bi - included metadata size
			// at the end this number will be < frameBufferSizeBits
			// so we can mark the end of the file
			// defer c.Wg.Done()

			// fmt.Println("Proccessing frame in G:", fn)
			// now := time.Now()

			// TODO: split into separate goroutines
			// Encoding bits to image - around 1.5s
			// fmt.Println("Frame start:", fn)
			img := encodeFrame(bufferBits, bitIndex)
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
			fileName := fmt.Sprintf("tmp/out/out_%08d.png", j.FrameNum)
			err = save(fileName, img)
			if err != nil {
				log.Println("Error saving file:", err)
				// NOTE: no need to continue if we can't save the file
				panic(fmt.Sprintf("EXITING!\n\n\nError saving file: %s", err))
			}
			// fmt.Println("Save done. Took time:", time.Since(now))
			// fmt.Println("Frame done:", fn)

			// }(bufferBits, bitIndex, frameNumber)
			// case <-ctx.Done():
			// 	fmt.Println("Goroutine %d canceled", g)
			// 	return
			// }
		}

	}
}

func (c *Core) Encode(path string) error {
	// open a file
	file, err := os.Open(path)
	if err != nil {
		log.Println("Error opening file:", err)
		return err
	}
	defer file.Close()

	// NOTE: read into buffer smaller then a frame to leave space for metadata
	readBuffer := make([]byte, sizeFrame-sizeMetadata)

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

	// ===== START WORKERS

	// create context to cancel goroutines on error
	// ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()

	jobs := make(chan job.Job) // fix array size, no buffer

	numCpu := runtime.NumCPU()

	wg := sync.WaitGroup{}
	for i := 0; i <= numCpu; i++ {
		wg.Add(1)
		i := i
		go func() {
			workerEncode(i, jobs)
			wg.Done()
		}()
	}

	time.Sleep(1 * time.Second)

	// feed the bytes from the file into the workers
	// TODO: make metadata with filename here
	// will hold filename and timestamp (now)
	// later will be updated with bytes and gen checksum from them
	// On processing will be added to the buffer as header
	md := meta.New(path)
	frameNumber := 0
	j := job.New(md, frameNumber)
	// read file into the buffer by chunks
	for {
		// NOTE: the buffer is reused on every loop so having n is important
		n, err := file.Read(readBuffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Println("Error reading file:", err)
			return err
		}
		// copy the buffer, do not send slice
		j.UpdateBuffer(readBuffer, n)
		// this will block untill available worker pick it up
		fmt.Println("Job: ", j.Print())
		panic("debug")
		jobs <- j
		_ = c.progress.Add(1)
		frameNumber++
	}
	// no more jobs to send, closing the channel
	// expected all the workers to finish and exit
	close(jobs)
	// wait for all the files to be processed
	wg.Wait()
	fmt.Println("All workers done")

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
func encodeFrame(bits []bool, bitIndex int) *image.NRGBA {
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
