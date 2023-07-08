package core

import (
	cfg "bytereel/pkg/config"
	"bytereel/pkg/encoder"
	"bytereel/pkg/job"
	"bytereel/pkg/meta"
	"bytereel/pkg/video"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"
)

func Encode(path string) error {
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
	// c.ResetProgress(estimatedFrames, "Encoding...") // set as spinner
	// _ = c.progress.Add(1)
	progress.Describe("Encoding...")
	_ = progress.Add(1)

	// ===== START WORKERS

	jobs := make(chan job.JobEnc)
	numCpu := runtime.NumCPU()

	wg := sync.WaitGroup{}
	for i := 0; i <= numCpu; i++ {
		wg.Add(1)
		i := i
		go func() {
			encoder.WorkerEncode(i, jobs)
			wg.Done()
		}()
	}

	// init metadata with filename and timestamp
	md := meta.New(path)
	frameCnt := 1
	// job object will be updated with copy of the buffer and send to the channel
	j := job.New(md, frameCnt)
	// read file into the buffer by chunks
	for {
		n, err := file.Read(readBuffer)
		if err != nil {
			if err == io.EOF {
				log.Debug("EOF")
				break
			}
			log.Println("Error reading file:", err)
			return err
		}
		// copy the buffer explicitly
		j.Update(readBuffer, n, frameCnt)
		log.Debugf("Sending job for frame %d: %s\n", frameCnt, j.Print())
		// this will block untill available worker pick it up
		log.Debug(j.Print())
		jobs <- j
		// _ = c.progress.Add(1)
		_ = progress.Add(1)
		frameCnt++
	}

	// no more jobs to send, closing the channel
	// expected all the workers to finish and exit
	close(jobs)

	// wait for all the files to be processed
	wg.Wait()
	log.Debug("All workers done")

	// ====== VIDEO ENCODING

	// setup progress bar async, otherwise it wont animate
	// c.ResetProgress(-1, "Saving video...")
	progress.Describe("Saving video...")
	done := make(chan bool)
	go func(done <-chan bool) {
		ticker := time.NewTicker(time.Millisecond * 300)
		for {
			select {
			case <-ticker.C:
				// _ = c.progress.Add(1)
				_ = progress.Add(1)
			case <-done:
				return
			}
		}
	}(done)

	// Call ffmpeg to decode the video into frames
	err = video.EncodeFrames()
	if err != nil {
		log.Fatal("Error encoding frames into video:", err)
	}
	done <- true
	_ = progress.Clear()

	// clean up tmp/out dir
	err = os.RemoveAll("tmp/out")
	if err != nil {
		panic(fmt.Sprintf("Error removing tmp/out dir: %s", err))
	}
	log.Debug("\nVideo encoded")

	return nil
}
