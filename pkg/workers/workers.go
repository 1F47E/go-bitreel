package workers

import (
	cfg "bytereel/pkg/config"
	"bytereel/pkg/encoder"
	"bytereel/pkg/job"
	"bytereel/pkg/logger"
	"bytereel/pkg/meta"
	"bytereel/pkg/storage"
	"fmt"
	"time"
)

var log = logger.Log

type Worker struct {
	encodingCh chan job.JobEnc
	decodingCh chan job.JobDec
	encoder    *encoder.FrameEncoder
}

func NewWorker() *Worker {
	return &Worker{
		encodingCh: make(chan job.JobEnc),
		decodingCh: make(chan job.JobDec),
		encoder:    encoder.NewFrameEncoder(cfg.SizeFrameWidth, cfg.SizeFrameHeight),
	}
}

func (w *Worker) WorkerEncode(i int, jobs <-chan job.JobEnc) {
	name := fmt.Sprintf("WorkerEncode #%d", i)
	log.Debugf("%s started\n", name)
	defer log.Debugf("%s finished\n", name)

	var err error
	for {
		j, ok := <-jobs
		if !ok {
			return
		}
		log.Debugf("%s got job %s\n", name, j.Print())

		// Encoding bits to image - around 1.5s
		now := time.Now()
		log.Debugf("%s Frame start: %d\n", name, j.FrameNum)
		img := w.encoder.EncodeFrame(j.Buffer, j.Metadata)
		log.Debugf("%s Frame done. Took time: %s\n", name, time.Since(now))

		// Saving image to file - around 7s
		now = time.Now()
		log.Debugf("%s Save start: %d\n", name, j.FrameNum)
		err = storage.SaveFrame(j.FrameNum, img)
		if err != nil {
			log.Fatalf("\n%s Error saving frame: %v\n", name, err)
		}
		log.Debugf("%s Saving done. Took time: %s\n", name, time.Since(now))
	}
}

func (w *Worker) WorkerDecode(id int, fCh <-chan job.JobDec, resChs []chan job.JobDecRes) {
	name := fmt.Sprintf("WorkerDecode #%d", id)
	log.Debugf("%s started\n", name)
	defer log.Debugf("%s finished\n", name)
	for {
		frame, ok := <-fCh
		if !ok {
			return
		}
		file := frame.File
		log.Debugf("%s got %d-%s\n", name, frame.Idx, file)

		// decode frame file into bytes
		frameBytes, fileBytesCnt := w.encoder.DecodeFrame(file)
		log.Debugf("%s decoded %s\n", name, file)

		// split frameBytes to header and data
		fileBytesCnt -= cfg.SizeMetadata
		header := frameBytes[:cfg.SizeMetadata]
		m, err := meta.Parse(header)
		if err != nil {
			log.Warnf("\n%s !!! metadata broken in file %s: %s\n", name, file, err)
		}
		log.Debugf("%s parsed metadata in %s\n", name, file)
		data := frameBytes[cfg.SizeMetadata : cfg.SizeMetadata+fileBytesCnt]

		// validate checksum
		isValid := m.Validate(data)
		if !isValid {
			log.Warnf("\n%s !!! frame checksum and metadata checksum mismatch in file %s\n", name, file)
		}
		log.Debugf("%s validated %s\n", name, file)
		resChs[frame.Idx] <- job.JobDecRes{
			Data: data,
			Meta: m,
		}

		log.Debugf("%s sent res %s\n", name, file)
	}
}
