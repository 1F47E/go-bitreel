package core

import (
	"bytereel/pkg/job"
	"bytereel/pkg/meta"
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"
)

// // job for the worker
// type frameJob struct {
// 	file string
// 	idx  int
// }
//
// // res from the worker
// type frameRes struct {
// 	data []byte
// 	meta meta.Metadata
// }

func (c *Core) Decode(videoFile string) (string, error) {
	var err error
	var out string
	var bytesWritten int
	// var pixelErrorsCount int
	var metadata meta.Metadata

	c.ResetProgress(-1, "Decoding video...") // spinner

	// ===== VIDEO DECODING

	// create dir to store frames
	framesDir := "tmp/frames"
	err = os.MkdirAll(framesDir, os.ModePerm)
	if err != nil {
		// panic(fmt.Sprintf("Error creating dir: %s", err))
		return out, fmt.Errorf("Error creating frames dir: %s", err)
	}

	// NOTE: total frames count is unknown at this point
	// but the total size of all frames is about 3% less then a video (in a corrent compression case)
	// so we can use the video file size to estimate the total frames count

	// start reporter
	done := make(chan bool)
	go c.framerReporter(framesDir, videoFile, done)

	framesPath := framesDir + "/out_%08d.png"
	// Call ffmpeg to decode the video into frames
	cmdStr := fmt.Sprintf("ffmpeg -y -i %s %s", videoFile, framesPath)
	cmdList := strings.Split(cmdStr, " ")
	log.Debugf("Running ffmpeg command:", cmdStr)
	cmd := exec.Command(cmdList[0], cmdList[1:]...)
	err = cmd.Run()
	if err != nil {
		panic(fmt.Sprintf("Error running ffmpeg: %s", err))
	}

	close(done)

	// ===== DECODING FRAMES

	// scan the directory for files
	files, err := os.ReadDir(framesDir)
	if err != nil {
		return out, err
	}
	// filter out files
	filesList := make([]string, 0, len(files))
	for _, file := range files {
		// check if the name has right prefix
		if strings.HasPrefix(file.Name(), "out_") {
			// add to the list
			filesList = append(filesList, framesDir+"/"+file.Name())
		}
	}
	if len(filesList) == 0 {
		log.Fatal("No files to decode")
	}
	log.Debugf("total frames: %d", len(filesList))
	sort.Strings(filesList)

	// setup progress bar
	c.ResetProgress(len(filesList), "Decoding frames...")

	// Create a temporary file in the same directory
	tmpFile, err := os.CreateTemp("", "decoded-")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tmpFile.Name()) // clean up

	// star the workers
	numCpu := runtime.NumCPU()

	framesCh := make(chan job.JobDec, numCpu) // buff by G count
	resChs := make([]chan job.JobDecRes, len(filesList))

	// create res channels
	for i := 0; i < len(filesList); i++ {
		resChs[i] = make(chan job.JobDecRes, 1)
	}

	log.Debugf("Starting %d workers", numCpu)
	for i := 0; i <= numCpu; i++ {
		i := i
		go workerDecode(i+1, framesCh, resChs)
	}

	// send all the jobs, in batches of G cnt
	go func() {
		for i, file := range filesList {
			framesCh <- job.JobDec{File: file, Idx: i}
			log.Debugf("Sent file %d/%d", i+1, len(filesList))
		}
	}()

	// read results, blocking, in order
	log.Debug("Reading res channels")
	for i, ch := range resChs {
		log.Debugf("Waiting for the res from the worker #%d/%d", i+1, len(resChs))
		fr := <-ch

		// set metadata if not set already
		if fr.Meta.IsOk() && !metadata.IsOk() {
			metadata = fr.Meta
			log.Warnf("Metadata found: %s", metadata.Print())
		}

		log.Debugf("Got the res from the worker #%d/%d - %d", i+1, len(resChs), len(fr.Data))
		written, err := tmpFile.Write(fr.Data)
		if err != nil {
			log.Fatal("Cannot write to file:", err)
		}
		bytesWritten += written
		if os.Getenv("DEBUG") != "" {
			_ = c.progress.Add(1)
		}
	}
	log.Debug("Closing res channels")
	for _, ch := range resChs {
		close(ch)
	}
	log.Debug("Closing frames channel")
	close(framesCh)

	// check metadata
	if metadata.IsOk() {
		out = metadata.Filename
	} else {
		log.Warn("\n!!! No metadata found")
		out = "out_decoded.bin" // default filename if no metadata found, unlikely to happen
	}

	// Write the data to the file and clear tmp folder with frames
	err = tmpFile.Sync()
	if err != nil {
		log.Fatal("Cannot sync file:", err)
	}
	err = tmpFile.Close()
	if err != nil {
		log.Fatal("Cannot close file:", err)
	}
	err = os.Rename(tmpFile.Name(), out)
	if err != nil {
		log.Error("Cant rename a file to ", out)
		return out, err
	}
	log.Infof("\n\nWrote %d bytes to %s\n", bytesWritten, out)
	err = os.RemoveAll(framesDir)
	if err != nil {
		log.Warn("!!! Cannot remove frames dir:", err)
	}
	return out, nil
}

func workerDecode(id int, fCh <-chan job.JobDec, resChs []chan job.JobDecRes) {
	log.Debugf("G %d started\n", id)
	defer log.Debugf("G %d finished\n", id)
	for {
		frame, ok := <-fCh
		if !ok {
			return
		}
		file := frame.File
		log.Debugf("G %d got %d-%s\n", id, frame.Idx, file)

		frameBytes, fileBytesCnt := decodeFrame(file)
		log.Debugf("G %d decoded %s\n", id, file)

		// split frameBytes to header and data
		fileBytesCnt -= sizeMetadata
		header := frameBytes[:sizeMetadata]
		m, err := meta.Parse(header)
		if err != nil {
			log.Warnf("\n!!! metadata broken in file %s: %s\n", file, err)
		}
		log.Debugf("G %d parsed metadata in %s\n", id, file)
		data := frameBytes[sizeMetadata : sizeMetadata+fileBytesCnt]

		// validate
		isValid := m.Validate(data)
		if !isValid {
			log.Errorf("\n!!! frame checksum and metadata checksum mismatch in file %s\n", file)
		}
		log.Debugf("G %d validated %s\n", id, file)
		resChs[frame.Idx] <- job.JobDecRes{
			Data: data,
			Meta: m,
		}

		log.Debugf("G %d sent res %s\n", id, file)
	}
}

// decoding video to frames progress runner
func (c *Core) framerReporter(dir string, videoFile string, done <-chan bool) {
	// get video file size
	fileInfo, err := os.Stat(videoFile)
	if err != nil {
		log.Fatal("Error opening file:", err)
	}
	videoFileSize := fileInfo.Size()
	totalFramesCount := int(videoFileSize/frameFileSize - 1) // 3% error
	log.Debug("Total frames count estimated:", totalFramesCount)

	ticker := time.NewTicker(time.Second / 10)
	defer ticker.Stop()
	prevCount := 0
	c.ResetProgress(totalFramesCount, "Extracting frames... ")
	for {
		select {
		case <-ticker.C:
			// scan dir
			files, err := os.ReadDir(dir)
			if err != nil {
				log.Warn("scanning dir error:", err)
				break
			}
			// count files
			l := len(files)
			if l > prevCount {
				prevCount = l
				_ = c.progress.Set(l)
			}
		case <-done:
			return
		}

	}
}

// NOTE:
// when decoder reached red area if will no longer write bits to the frameBytes
// so we need fileBytesCnt to know how many bytes to write to the file
// fileBytesCnt will include metadata size!
func decodeFrame(filename string) ([]byte, int) {

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

	var fileBits [sizeFrame * 8]bool
	// fileBits := make([]bool, sizeFrame*8)

	// copy image to bytes
	var writeIdx int
	// black = 1, white = 0, red = EOF
	var cntBlack, cntWhite, cntRed uint
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

					// detect red first
					if r8 > 128 && g8 < 128 && b8 < 128 {
						cntRed++
					} else if r8 > 128 && g8 > 128 && b8 > 128 {
						cntWhite++
					} else {
						cntBlack++
					}
				}
			}

			// skip if reached red section which is EOF
			// TODO: skip after some time, do not scan till the end of the frame
			if cntRed > cntBlack && cntRed > cntWhite {
				// reset counters
				cntRed = 0
				cntBlack = 0
				cntWhite = 0
				continue
			}

			if cntBlack > cntWhite {
				fileBits[writeIdx] = true
				if cntBlack != 4 {
					// fmt.Println("Error at black ", x, y, cntBlack, cntWhite)
					pixelErrorsCount++
				}
			} else {
				fileBits[writeIdx] = false
				if cntWhite != 4 {
					// fmt.Println("Error at white ", x, y, cntBlack, cntWhite)
					pixelErrorsCount++
				}
			}
			cntRed = 0
			cntBlack = 0
			cntWhite = 0
			writeIdx++
		}
	}
	if pixelErrorsCount > 0 {
		log.Warnf("Pixel errors (%d) corrected in frame: %s\n", pixelErrorsCount, filename)
	}

	// 4k video is 3840x2160 = 8294400 pixels = 2073600 4px blocks
	// every frame should have 2073600 bits
	// convert bits to bytes
	bytes := make([]byte, len(fileBits)/8)
	for i := 0; i < len(fileBits); i += 8 {
		var b byte
		for j := 0; j < 8; j++ {
			if fileBits[i+j] {
				b |= 1 << uint(j)
			}
		}
		bytes[i/8] = b
	}
	// fmt.Println("bytes len:", len(bytes))

	writtenBytes := writeIdx / 8
	return bytes, writtenBytes
}
