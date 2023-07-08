package core

import (
	cfg "bytereel/pkg/config"
	"bytereel/pkg/encoder"
	"bytereel/pkg/fs"
	"bytereel/pkg/job"
	"bytereel/pkg/meta"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func Decode(videoFile string) (string, error) {
	var err error
	var out string
	var bytesWritten int
	var metadata meta.Metadata

	// .ResetProgress(-1, "Decoding video...") // spinner
	progress.Describe("Decoding video...")

	// ===== VIDEO DECODING

	// create dir to store frames
	framesDir, err := fs.CreateFramesDir()
	if err != nil {
		log.Fatal("Error creating frames dir:", err)
	}

	// NOTE: total frames count is unknown at this point
	// but the total size of all frames is about 3% less then a video (in a corrent compression case)
	// so we can use the video file size to estimate the total frames count

	// start reporter
	done := make(chan bool)
	go framerReporter(framesDir, videoFile, done)

	framesPath := framesDir + "/out_%08d.png"
	// Call ffmpeg to decode the video into frames
	cmdStr := fmt.Sprintf("ffmpeg -y -i %s %s", videoFile, framesPath)
	cmdList := strings.Split(cmdStr, " ")
	log.Debug("Running ffmpeg command:", cmdStr)
	cmd := exec.Command(cmdList[0], cmdList[1:]...)
	err = cmd.Run()
	if err != nil {
		panic(fmt.Sprintf("Error running ffmpeg: %s", err))
	}

	close(done)

	// ===== DECODING FRAMES

	// scan the directory for files
	filesList, err := fs.ScanFrames()
	if err != nil {
		log.Fatal("Error scanning frames dir:", err)
	}
	log.Debugf("total frames: %d", len(filesList))

	progress.Describe("Decoding frames...")

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
		go encoder.WorkerDecode(i+1, framesCh, resChs)
	}

	// send all the jobs, in batches of G cnt
	go func() {
		for i, file := range filesList {
			framesCh <- job.JobDec{File: file, Idx: i}
			log.Debugf("Sent file %d/%d", i+1, len(filesList))
		}
	}()

	// write results to file, blocking, in order
	// Create a temporary file in the same directory
	log.Debug("Reading res channels, writing to file")
	tmpFile, err := fs.CreateTempFile()
	if err != nil {
		log.Fatal("Cannot create temp file:", err)
	}
	for i, ch := range resChs {
		log.Debugf("Waiting for the res from the worker #%d/%d", i+1, len(resChs))
		fr := <-ch

		// set metadata if not set already
		// it may be lost in some frames, check untill found
		if fr.Meta.IsOk() && !metadata.IsOk() {
			metadata = fr.Meta
			log.Println()
			log.Warnf("Metadata found: %s", metadata.Print())
		}

		log.Debugf("Got the res from the worker #%d/%d - %d", i+1, len(resChs), len(fr.Data))
		written, err := tmpFile.Write(fr.Data)
		if err != nil {
			log.Fatal("Cannot write to file:", err)
		}
		bytesWritten += written
		if os.Getenv("DEBUG") != "" {
			_ = progress.Add(1)
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

	err = fs.SaveDecoded(tmpFile, out)
	if err != nil {
		log.Fatal("Cannot save decoded file:", err)
	}
	return out, nil
}

// decoding video to frames progress runner
func framerReporter(dir string, videoFile string, done <-chan bool) {
	// get video file size
	fileInfo, err := os.Stat(videoFile)
	if err != nil {
		log.Fatal("Error opening file:", err)
	}
	videoFileSize := fileInfo.Size()
	totalFramesCount := int(videoFileSize/cfg.FrameFileSize - 1) // 3% error
	log.Debug("Total frames count estimated:", totalFramesCount)

	ticker := time.NewTicker(time.Second / 10)
	defer ticker.Stop()
	prevCount := 0
	// c.ResetProgress(totalFramesCount, "Extracting frames... ")
	progress.Describe("Extracting frames... ")
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
				// _ = c.progress.Set(l)
				_ = progress.Set(l)
			}
		case <-done:
			return
		}

	}
}
