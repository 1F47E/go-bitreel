package workers

import (
	"context"
	"fmt"
	"time"

	cfg "github.com/1F47E/go-bytereel/pkg/config"
	"github.com/1F47E/go-bytereel/pkg/encoder"
	"github.com/1F47E/go-bytereel/pkg/job"
	"github.com/1F47E/go-bytereel/pkg/logger"
	"github.com/1F47E/go-bytereel/pkg/meta"
	"github.com/1F47E/go-bytereel/pkg/storage"
)

type Worker struct {
	ctx        context.Context
	encodingCh chan job.JobEnc
	decodingCh chan job.JobDec
	encoder    *encoder.FrameEncoder
}

func NewWorker(ctx context.Context) *Worker {
	return &Worker{
		ctx:        ctx,
		encodingCh: make(chan job.JobEnc),
		decodingCh: make(chan job.JobDec),
		encoder:    encoder.NewFrameEncoder(cfg.SizeFrameWidth, cfg.SizeFrameHeight),
	}
}

func (w *Worker) WorkerEncode(i int, jobs <-chan job.JobEnc) {
	log := logger.Log.WithField("scope", fmt.Sprintf("WorkerEncode #%d", i))
	name := fmt.Sprintf("WorkerEncode #%d", i)
	log.Debugf("%s started\n", name)
	defer log.Debugf("%s finished\n", name)

	var err error
	for {
		select {
		case <-w.ctx.Done():
			return
		case j, ok := <-jobs:
			if !ok {
				return
			}
			log.Debugf("%s got job %s\n", name, j.Print())

			// Encoding bits to image - about 1.5s
			now := time.Now()
			log.Debugf("%s Frame start: %d\n", name, j.FrameNum)
			img := w.encoder.EncodeFrame(j.Buffer, j.Metadata)
			log.Debugf("%s Frame done. Took time: %s\n", name, time.Since(now))

			// Saving image to file - about 5s
			now = time.Now()
			log.Debugf("%s Save start: %d\n", name, j.FrameNum)
			err = storage.SaveFrame(j.FrameNum, img)
			if err != nil {
				log.Fatalf("\n%s Error saving frame: %v\n", name, err)
			}
			log.Debugf("%s Saving done. Took time: %s\n", name, time.Since(now))
		}
	}
}

func (w *Worker) WorkerDecode(id int, fCh <-chan job.JobDec, resChs []chan job.JobDecRes) {
	log := logger.Log.WithField("scope", fmt.Sprintf("WorkerDecode #%d", id))
	log.Debug("started")
	defer log.Debug("finished")
	for {
		select {
		case <-w.ctx.Done():
			return
		case frame, ok := <-fCh:
			if !ok {
				return
			}
			file := frame.File
			log.Debugf(" got %d-%s\n", frame.Idx, file)

			// decode frame file into bytes
			frameBytes, fileBytesCnt := w.encoder.DecodeFrame(file)
			log.Debugf("decoded %s\n", file)

			// split frameBytes to header and data
			fileBytesCnt -= cfg.SizeMetadata
			header := frameBytes[:cfg.SizeMetadata]
			m, err := meta.Parse(header)
			if err != nil {
				log.Warnf("\n!!! metadata broken in file %s: %s\n", file, err)
			}
			log.Debugf("parsed metadata in %s\n", file)
			data := frameBytes[cfg.SizeMetadata : cfg.SizeMetadata+fileBytesCnt]

			// validate checksum
			isValid, err := m.Validate(data)
			if err != nil {
				log.Warnf("\n!!! checksum validation failed in file %s: %s\n", file, err)
			}
			if !isValid {
				log.Warnf("\n!!! frame checksum and metadata checksum mismatch in file %s\n", file)
			}
			log.Debugf("validated %s\n", file)
			resChs[frame.Idx] <- job.JobDecRes{
				Data: data,
				Meta: m,
			}

			log.Debugf("sent res %s\n", file)
		}
	}
}
