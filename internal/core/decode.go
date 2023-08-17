package core

import (
	"fmt"
	"os"
	"runtime"
	"time"

	cfg "github.com/1F47E/go-bitreel/internal/config"
	"github.com/1F47E/go-bitreel/internal/job"
	"github.com/1F47E/go-bitreel/internal/logger"
	"github.com/1F47E/go-bitreel/internal/meta"
	"github.com/1F47E/go-bitreel/internal/storage"
	"github.com/1F47E/go-bitreel/internal/tui"
	"github.com/1F47E/go-bitreel/internal/video"
)

// 1. extract frames from video
// 2. decode frames into bytes by workers, send results to separage channel in resChs
// 3. write to result file continuously. Read from resChs in order from every worker
func (c *Core) Decode(videoFile string) (string, error) {
	log := logger.Log.WithField("scope", "core decode")
	var err error

	c.eventsCh <- tui.NewEventSpin("Decoding video...")

	// extract frames from video
	err = c.framesExtract(videoFile)
	if err != nil {
		return "", err
	}

	c.eventsCh <- tui.NewEventSpin("Scanning frames...")

	// scan dir for frames
	filesList, err := storage.ScanFrames()
	if err != nil {
		return "", err
	}
	log.Debugf("total frames: %d", len(filesList))

	// list of channels to receive results from workers in order
	resChs := make([]chan job.JobDecRes, len(filesList))
	for i := 0; i < len(filesList); i++ {
		resChs[i] = make(chan job.JobDecRes, 1)
	}

	// create channels and start the workers
	cores := runtime.NumCPU()
	framesCh := make(chan job.JobDec, cores) // buff by G count
	log.Debugf("Starting %d workers", cores)
	for i := 0; i <= cores; i++ {
		i := i
		go c.worker.WorkerDecode(i+1, framesCh, resChs)
	}

	// send all the jobs
	go func() {
		for i, file := range filesList {
			framesCh <- job.JobDec{File: file, Idx: i}
			log.Debugf("Sent file %d/%d", i+1, len(filesList))
		}
	}()

	// Frames writer
	// will start when all the frames are extracted
	// Because its secuential and we need to write res file in order
	out, err := c.framesWrite(resChs)
	if err != nil {
		return "", err
	}

	// cleanup
	log.Debug("Closing res channels")
	for _, ch := range resChs {
		close(ch)
	}
	log.Debug("Closing frames channel")
	close(framesCh)

	return out, nil
}

func (c *Core) framesExtract(videoFile string) error {
	c.eventsCh <- tui.NewEventSpin("Decoding video...")

	// create dir to store frames
	framesDir, err := storage.CreateFramesDir()
	if err != nil {
		return fmt.Errorf("Error creating frames dir: %w", err)
	}

	// start frames progress reporter
	c.eventsCh <- tui.NewEventSpin("Extracting frames...")
	done := make(chan bool)

	// fill scan frames folder untill video finishes extracting
	// updates progress bar in a loop
	go c.scanFramesDir(framesDir, videoFile, done)

	err = video.ExtractFrames(c.ctx, videoFile, framesDir)
	if err != nil {
		return fmt.Errorf("Error extracting frames: %w", err)
	}

	close(done)
	return nil
}

func (c *Core) framesWrite(resChs []chan job.JobDecRes) (string, error) {
	log := logger.Log
	var out string
	var bytesWritten int
	var metadata meta.Metadata
	// Create a temporary file in the same directory
	log.Debug("Reading res channels, writing to file")
	tmpFile, err := storage.CreateTempFile()
	if err != nil {
		return "", fmt.Errorf("Cannot create temp file: %w", err)
	}

	c.eventsCh <- tui.NewEventSpin("Writing results...")

	// ranging over channels because work should be done in order
	// write results to file, blocking, in order
	for i, ch := range resChs {
	loop:
		for {
			select {
			case <-c.ctx.Done():
				log.Debug("Decoder exit")
				return out, c.ctx.Err()
			case fr := <-ch:
				log.Debugf("Waiting for the res from the worker #%d/%d", i+1, len(resChs))

				// set metadata if not set already
				// it may be lost in some frames, check untill found
				if fr.Meta.IsOk() && !metadata.IsOk() {
					metadata = fr.Meta
				}

				log.Debugf("Got the res from the worker #%d/%d - %d", i+1, len(resChs), len(fr.Data))
				written, err := tmpFile.Write(fr.Data)
				if err != nil {
					return "", fmt.Errorf("Cannot write to file: %w", err)
				}
				bytesWritten += written
				break loop
			}
		}
	}

	// check metadata
	statusMsg := ""
	if metadata.IsOk() {
		out = metadata.Filename
		statusMsg = metadata.Print()
	} else {
		// default filename if no metadata found, unlikely to happen
		out = "out_decoded.bin"
		statusMsg = fmt.Sprintf("Metadata not found, result file - %s", out)
	}
	c.eventsCh <- tui.NewEventText(statusMsg)

	err = storage.SaveDecoded(tmpFile, out)
	if err != nil {
		return "", fmt.Errorf("cannot save decoded file: %w", err)
	}
	return out, nil
}

// Decoding video to frames progress runner
// NOTE: total frames count is unknown at this point
// but the total size of all frames is about 3% less then a video (in a corrent compression case)
// so we can use the video file size to estimate the total frames count
func (c *Core) scanFramesDir(dir string, videoFile string, done <-chan bool) {
	log := logger.Log
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

	// update progress with estimated num of frames
	c.eventsCh <- tui.NewEventBar(fmt.Sprintf("Extracting frames... %d/%d", 0, totalFramesCount), 0)

	prevCount := 0
	for {
		select {
		case <-c.ctx.Done():
			return
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

				// update progress
				percent := float64(l) / float64(totalFramesCount)
				c.eventsCh <- tui.NewEventBar(fmt.Sprintf("Extracting frames... %d/%d", l, totalFramesCount), percent)
			}
			log.Debugf("Scanned %d/%d frames", l, totalFramesCount)
		case <-done:
			return
		}
	}
}
