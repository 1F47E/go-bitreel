package job

import (
	"bytereel/pkg/meta"
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

const metadataMaxFilenameLen = 524
const metadataEOFMarker = "/"

type Job struct {
	Buffer   []byte
	metadata meta.Metadata
	FrameNum int
}

func New(m meta.Metadata, fn int) Job {
	return Job{
		metadata: m,
		FrameNum: fn,
	}
}

func (j *Job) Print() string {
	return fmt.Sprintf("Job: FrameNum: %d, Meta: %s, Buffer len: %d", j.FrameNum, j.metadata.Print(), len(j.Buffer))
}

// func (j *Job) GetBits() []bool {
// 	// fill the metadata first
// 	s := 0
// 	l := len(j.timestamp)
// 	copy(j.buffer[s:l], j.timestamp[:])
// 	s = l
// 	l = s + len(j.filename)
// 	copy(j.buffer[s:l], j.filename[:])
//
// 	// // get metadata as bits
// 	// filenameBits := EncodeFilename(m.filename)
// 	// timestampBits := EncodeTimestamp()
// 	// checksumBits := EncodeChecksum()
// 	// // join all metadata bits
// 	// metadataBits := append(filenameBits, timestampBits...)
// 	// metadataBits = append(metadataBits, checksumBits...)
// 	// return metadataBits
// }

func (j *Job) UpdateBuffer(b []byte, n int) {
	// copy buffer to avoid overwriting of the same buffer
	cp := make([]byte, n)
	_ = copy(cp, b[:n])
	j.Buffer = cp
}

// get datetime in users format
// func (j *Job) GetDatetime() string {
// 	t := time.Unix(j.timestamp, 0)
// 	localTime := t.Local()
// 	return localTime.Format(time.RFC822)
// }

// METADATA - timestamp, 64 bits
func encodeTimestamp() []byte {
	timestamp := time.Now().Unix()
	timeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timeBytes, uint64(timestamp))
	return timeBytes
}

// func (j *Job) Checksum() uint64 {
// 	return 0
// }

func encodeFilename(path string) string {

	filename := path[strings.LastIndex(path, "/")+1:]
	// add marker to the end of the filename so on decoding we know the end
	filename += "/"
	return filename
	// return bytesToBits([]byte(filename))
	// fmt.Println("filename", filename)
	// printBits(filenameBits)
}

// METADATA - CHECKSUM, 64 bits
// func (j *Job) checksum() ([]byte, error) {
//
// 	// create checksum hash - 8bytes, 64bits
// 	hasher := fnv.New64a() // FNV-1a hash
// 	// Pass sliced buffer slice to hasher, no copy
// 	// also important to pass n - number of bytes read in case of last chunk
// 	_, err := hasher.Write(j.bytes)
// 	if err != nil {
// 		log.Println("META:Error writing to hasher:", err)
// 		return nil, err
// 	}
// 	checksum := hasher.Sum64()
// 	checksumBytes := make([]byte, 8)
// 	binary.BigEndian.PutUint64(checksumBytes, checksum)
// 	return checksumBytes, nil
// }

func bytesToBits(bytes []byte) []bool {
	bits := make([]bool, 8*len(bytes))
	for i, b := range bytes {
		for j := 0; j < 8; j++ {
			bits[i*8+j] = (b & (1 << uint(j))) != 0
		}
	}
	return bits
}
