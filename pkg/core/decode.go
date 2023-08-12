package core

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	cfg "github.com/1F47E/go-bytereel/pkg/config"
	"github.com/1F47E/go-bytereel/pkg/job"
	"github.com/1F47E/go-bytereel/pkg/logger"
	"github.com/1F47E/go-bytereel/pkg/meta"
	"github.com/1F47E/go-bytereel/pkg/storage"
	"github.com/1F47E/go-bytereel/pkg/tui"
	"github.com/1F47E/go-bytereel/pkg/video"
)

// 1. extract frames from video
// 2. decode frames into bytes by workers, send results to separage channel in resChs
// 3. write to result file continuously. Read from resChs in order from every worker
func (c *Core) Decode(videoFile string) (string, error) {
	log := logger.Log.WithField("scope", "core decode")
	var err error

	c.eventsCh <- tui.NewEventSpin("Decoding video...")

	// extract frames from video
	err = framesExtract(c.ctx, c.eventsCh, videoFile)
	if err != nil {
		return "", err
	}

	// scan dir for frames
	filesList, err := storage.ScanFrames()
	if err != nil {
		return "", err
	}
	log.Debugf("total frames: %d", len(filesList))

	c.eventsCh <- tui.NewEventSpin(fmt.Sprintf("Decoding %d frames...", len(filesList)))

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

	// read the frames channel and write results to a file
	out, err := framesWrite(c.ctx, resChs)
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

func framesExtract(ctx context.Context, eventsCh chan tui.Event, videoFile string) error {
	eventsCh <- tui.NewEventSpin("Decoding video...")
	// p.ProgressSpinner("Decoding video... ")

	// create dir to store frames
	framesDir, err := storage.CreateFramesDir()
	if err != nil {
		return fmt.Errorf("Error creating frames dir: %w", err)
	}

	// start frames progress reporter
	// p.ProgressReset(0, "Extracting frames... ")
	eventsCh <- tui.NewEventSpin("Extracting frames...")
	done := make(chan bool)

	// fill scan frames folder untill video finishes extracting
	// updates progress bar in a loop
	go scanFramesDir(framesDir, videoFile, done)

	err = video.ExtractFrames(ctx, videoFile, framesDir)
	if err != nil {
		return fmt.Errorf("Error extracting frames: %w", err)
	}

	// stop the progress reporter and dir scanner
	eventsCh <- tui.NewEventText("Done.")
	close(done)
	return nil
}

func framesWrite(ctx context.Context, resChs []chan job.JobDecRes) (string, error) {
	log := logger.Log
	var out string
	var bytesWritten int
	var metadata meta.Metadata
	// Create a temporary file in the same directory
	log.Debug("Reading res channels, writing to file")
	tmpFile, err := storage.CreateTempFile()
	if err != nil {
		log.Fatal("Cannot create temp file:", err)
	}

	// ranging over channels because work should be done in order
	// write results to file, blocking, in order
	for i, ch := range resChs {
	loop:
		for {
			select {
			case <-ctx.Done():
				log.Debug("Decoder exit")
				return out, ctx.Err()
			case fr := <-ch:
				log.Debugf("Waiting for the res from the worker #%d/%d", i+1, len(resChs))

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
				// p.Add(1)
				// TODO: add progress
				break loop
			}
		}
	}

	// check metadata
	if metadata.IsOk() {
		out = metadata.Filename
	} else {
		log.Warn("\n!!! No metadata found")
		out = "out_decoded.bin" // default filename if no metadata found, unlikely to happen
	}

	err = storage.SaveDecoded(tmpFile, out)
	if err != nil {
		log.Fatal("Cannot save decoded file:", err)
	}
	log.Infof("Decoded file saved: %s", out)
	return out, nil
}

// Decoding video to frames progress runner
// NOTE: total frames count is unknown at this point
// but the total size of all frames is about 3% less then a video (in a corrent compression case)
// so we can use the video file size to estimate the total frames count
func scanFramesDir(dir string, videoFile string, done <-chan bool) {
	log := logger.Log.WithField("scope", "core scanFramesDir")
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
	// p.Max(totalFramesCount)
	// TODO: add progress

	prevCount := 0
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
				// p.Set(l)
				// TODO: add progress
			}
			log.Debugf("Scanned %d/%d frames", l, totalFramesCount)
		case <-done:
			return
		}
	}
}
