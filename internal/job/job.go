package job

import (
	"fmt"

	"github.com/1F47E/go-bitreel/internal/meta"
)

// job for the decoding worker
type JobDec struct {
	File string
	Idx  int
}

// res from the decoding worker
type JobDecRes struct {
	Data []byte
	Meta meta.Metadata
}

// job for the encoding worker
type JobEnc struct {
	Buffer   []byte
	Metadata meta.Metadata
	FrameNum int
}

func New(m meta.Metadata, fn int) JobEnc {
	return JobEnc{
		Metadata: m,
		FrameNum: fn,
	}
}

func (j *JobEnc) Print() string {
	return fmt.Sprintf("Job: FrameNum: %d, Meta: %s, Buffer len: %d", j.FrameNum, j.Metadata.Print(), len(j.Buffer))
}

func (j *JobEnc) Update(buf []byte, bufLen int, frameNum int) {
	j.Buffer = append([]byte{}, buf[:bufLen]...)
	j.FrameNum = frameNum
}
