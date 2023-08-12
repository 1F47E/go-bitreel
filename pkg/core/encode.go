package core

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"

	cfg "github.com/1F47E/go-bytereel/pkg/config"
	"github.com/1F47E/go-bytereel/pkg/job"
	"github.com/1F47E/go-bytereel/pkg/logger"
	"github.com/1F47E/go-bytereel/pkg/meta"
	"github.com/1F47E/go-bytereel/pkg/tui"
	"github.com/1F47E/go-bytereel/pkg/video"
)

// 1. read file into buffer by chunks
// 2. encode chunks to images and write to files as png frames
// 3. encode frames into video
func (c *Core) Encode(path string) error {
	log := logger.Log.WithField("scope", "core encode")
	// open a file
	file, err := os.Open(path)
	if err != nil {
		log.Println("Error opening file:", err)
		return err
	}
	defer file.Close()

	// NOTE: read into buffer smaller then a frame to leave space for metadata
	readBuffer := make([]byte, cfg.SizeFrame-cfg.SizeMetadata)

	// Progress bar with frames count progress
	// get total file size
	fileInfo, err := file.Stat()
	if err != nil {
		log.Println("Error getting file info:", err)
		return err
	}
	size := fileInfo.Size()
	estimatedFrames := int(int(size) / len(readBuffer))
	log.Debug("Estimated frames:", estimatedFrames)

	// change TUI to progress bar mode and update title and percents
	c.eventsCh <- tui.NewEventBar(fmt.Sprintf("Encoding... %d frames", estimatedFrames), 0)

	// ===== START WORKERS

	jobs := make(chan job.JobEnc)
	numCpu := runtime.NumCPU()

	wg := sync.WaitGroup{}
	for i := 0; i <= numCpu; i++ {
		wg.Add(1)
		i := i
		go func() {
			c.worker.WorkerEncode(i, jobs)
			wg.Done()
		}()
	}

	// init metadata with filename and timestamp
	md := meta.New(path)
	frameCnt := 1

	// job object will be updated with copy of the buffer and send to the channel
	j := job.New(md, frameCnt)

	// read file into the buffer by chunks
loop:
	for {
		select {
		case <-c.ctx.Done():
			return c.ctx.Err()
		default:
			n, err := file.Read(readBuffer)
			if err != nil {
				if err == io.EOF {
					log.Debug("EOF")
					break loop
				}
				log.Println("Error reading file:", err)
				return err
			}
			// copy the buffer to the job
			j.Update(readBuffer, n, frameCnt)
			log.Debugf("Sending job for frame %d: %s\n", frameCnt, j.Print())
			// this will block untill available worker pick it up
			log.Debug(j.Print())
			jobs <- j

			// update progress bar
			percent := float64(frameCnt) / float64(estimatedFrames)
			c.eventsCh <- tui.NewEventBar(fmt.Sprintf("Encoding... %d frames", frameCnt), percent)

			frameCnt++
		}
	}

	// expected all the workers to finish and exit
	close(jobs)

	// wait for all the files to be processed
	wg.Wait()
	log.Debug("All workers done")

	// ====== VIDEO ENCODING

	// setup progress bar async, otherwise it wont animate
	// p.ProgressSpinner("Saving video... ")
	c.eventsCh <- tui.NewEventSpin("Saving video... ")

	// done := make(chan bool)
	// go func() {
	// 	cnt := 0
	// 	ticker := time.NewTicker(time.Millisecond * 300)
	// 	for {
	// 		select {
	// 		case <-ticker.C:
	// 			p.Add(1) // spin
	// 			c.eventsCh <- tui.Event{Type: "spinner", Data: fmt.Sprintf("Saving video... %d frames", cnt)}
	// 		case <-done:
	// 			return
	// 		}
	// 	}
	// }()

	// Call ffmpeg to encode frames into video
	err = video.EncodeFrames(c.ctx)
	if err != nil {
		log.Fatal("Error encoding frames into video:", err)
	}
	// close(done)

	// clean up tmp/out dir
	err = os.RemoveAll("tmp/out")
	if err != nil {
		panic(fmt.Sprintf("Error removing tmp/out dir: %s", err))
	}
	log.Debug("\nVideo encoded")

	return nil
}
