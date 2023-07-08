package job

import (
	"bytereel/pkg/meta"
	"fmt"
)

// job for the worker
type JobDec struct {
	File string
	Idx  int
}

// res from the worker
type JobDecRes struct {
	Data []byte
	Meta meta.Metadata
}

type JobEnc struct {
	Buffer   []byte
	metadata meta.Metadata
	FrameNum int
}

func New(m meta.Metadata, fn int) JobEnc {
	return JobEnc{
		metadata: m,
		FrameNum: fn,
	}
}

func (j *JobEnc) Print() string {
	return fmt.Sprintf("Job: FrameNum: %d, Meta: %s, Buffer len: %d", j.FrameNum, j.metadata.Print(), len(j.Buffer))
}

// get metadata bits
func (j *JobEnc) GetMetadataBits(buff []byte) []bool {
	return j.metadata.Hash(buff)
}

func (j *JobEnc) Update(buf []byte, bufLen int, frameNum int) {
	// copy buffer to avoid overwriting of the same buffer
	cp := make([]byte, bufLen)
	_ = copy(cp, buf[:bufLen])
	j.Buffer = cp
	j.FrameNum = frameNum
}
