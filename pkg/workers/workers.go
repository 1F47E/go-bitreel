package workers

import (
	cfg "bytereel/pkg/config"
	"bytereel/pkg/encoder"
	"bytereel/pkg/fs"
	"bytereel/pkg/job"
	"bytereel/pkg/logger"
	"bytereel/pkg/meta"
	"time"
)

var log = logger.Log

func WorkerEncode(g int, jobs <-chan job.JobEnc) {
	log.Debugf("Goroutine %d started\n", g)
	defer log.Debugf("Goroutine %d finished\n", g)

	var err error
	for {
		j, ok := <-jobs
		if !ok {
			return
		}
		log.Debugf("G #%d got job %s\n", g, j.Print())

		// Encoding bits to image - around 1.5s
		now := time.Now()
		log.Debugf("G #%d Frame start: %d\n", g, j.FrameNum)
		enc := encoder.NewFrameEncoder(cfg.SizeFrameWidth, cfg.SizeFrameHeight)
		img := enc.EncodeFrame(j.Buffer, j.Metadata)
		log.Debugf("G #%d Frame done. Took time: %s\n", g, time.Since(now))

		// Saving image to file - around 7s
		now = time.Now()
		log.Debugf("G #%d Save start: %d\n", g, j.FrameNum)
		err = fs.SaveFrame(j.FrameNum, img)
		if err != nil {
			log.Fatal("\nError saving frame:", err)
		}
		log.Debugf("G #%d Save done. Took time: %s\n", g, time.Since(now))
	}
}

func WorkerDecode(id int, fCh <-chan job.JobDec, resChs []chan job.JobDecRes) {
	log.Debugf("G %d started\n", id)
	defer log.Debugf("G %d finished\n", id)
	for {
		frame, ok := <-fCh
		if !ok {
			return
		}
		file := frame.File
		log.Debugf("G %d got %d-%s\n", id, frame.Idx, file)

		// decode frame file into bytes
		enc := encoder.NewFrameEncoder(cfg.SizeFrameWidth, cfg.SizeFrameHeight)
		frameBytes, fileBytesCnt := enc.DecodeFrame(file)
		log.Debugf("G %d decoded %s\n", id, file)

		// split frameBytes to header and data
		fileBytesCnt -= cfg.SizeMetadata
		header := frameBytes[:cfg.SizeMetadata]
		m, err := meta.Parse(header)
		if err != nil {
			log.Warnf("\n!!! metadata broken in file %s: %s\n", file, err)
		}
		log.Debugf("G %d parsed metadata in %s\n", id, file)
		data := frameBytes[cfg.SizeMetadata : cfg.SizeMetadata+fileBytesCnt]

		// validate checksum
		isValid := m.Validate(data)
		if !isValid {
			log.Warnf("\n!!! frame checksum and metadata checksum mismatch in file %s\n", file)
		}
		log.Debugf("G %d validated %s\n", id, file)
		resChs[frame.Idx] <- job.JobDecRes{
			Data: data,
			Meta: m,
		}

		log.Debugf("G %d sent res %s\n", id, file)
	}
}
