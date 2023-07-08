package meta

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"log"
	"reflect"
	"strings"
	"testing"
	"time"
)

const metadataMaxFilenameLen = 524
const sizeMetadata = 256

type Metadata struct {
	filename  string
	timestamp int64
}

func New(path string) Metadata {
	return Metadata{
		filename:  encodeFilename(path),
		timestamp: time.Now().Unix(),
	}
}

func (m *Metadata) Print() string {
	return fmt.Sprintf("Filename: %s, Timestamp: %d", m.filename, m.timestamp)
}

func (m *Metadata) Hash(bytes []byte) []bool {

	header := make([]byte, sizeMetadata)
	checksum := generateChecksum(&bytes)
	checksumBytes := convertUint64ToBytes(checksum)

	// copy checksum to header
	s := 0
	l := len(checksumBytes)
	copy(header[s:l], checksumBytes[:])

	// copy timestamp
	tsBytes := convertUint64ToBytes(uint64(m.timestamp))
	binary.BigEndian.PutUint64(tsBytes, uint64(m.timestamp))
	s = l
	l = s + 8
	copy(header[s:l], tsBytes[:])

	// copy filename
	fnBytes := make([]byte, len(m.filename))
	copy(fnBytes, []byte(m.filename))
	s = l
	l = s + len(m.filename)
	copy(header[s:l], fnBytes[:])

	return bytesToBits(header)
}

func generateChecksum(bytes *[]byte) uint64 {
	hasher := fnv.New64a()
	_, err := hasher.Write(*bytes)
	if err != nil {
		log.Println("META:Error writing to hasher:", err)
		panic("META:Error writing to hasher")
	}
	return hasher.Sum64()
}

func convertUint64ToBytes(num uint64) []byte {
	byteArray := make([]byte, 8)
	binary.BigEndian.PutUint64(byteArray, num)
	return byteArray
}

// METADATA - timestamp, 64 bits
// func encodeTimestamp() []byte {
// 	timestamp := time.Now().Unix()
// 	timeBytes := make([]byte, 8)
// 	binary.BigEndian.PutUint64(timeBytes, uint64(timestamp))
// 	return timeBytes
// }

// func (j *Job) Checksum() uint64 {
// 	return 0
// }

func encodeFilename(path string) string {
	// TODO: deal with too long filename
	filename := path[strings.LastIndex(path, "/")+1:]
	// add marker to the end of the filename so on decoding we know the end
	filename += "/"
	return filename
	// return bytesToBits([]byte(filename))
	// fmt.Println("filename", filename)
	// printBits(filenameBits)
}

// get metadata as bits
// func (m *Metadata) GetBits() []bool {
//
// 	// fill the metadata first
// 	s := 0
// 	l := len(checksumBits)
// 	copy(bufferBits[s:l], checksumBits[:])
// 	s = l
// 	l = s + len(timeBits)
// 	copy(bufferBits[s:l], timeBits[:])
// 	s = l
// 	l = s + len(filenameBits)
// 	copy(bufferBits[s:l], filenameBits[:])
// 	// // get metadata as bits
// 	// filenameBits := EncodeFilename(m.filename)
// 	// timestampBits := EncodeTimestamp()
// 	// checksumBits := EncodeChecksum()
// 	// // join all metadata bits
// 	// metadataBits := append(filenameBits, timestampBits...)
// 	// metadataBits = append(metadataBits, checksumBits...)
// 	// return metadataBits
// }

// get datetime in users format
func (m *Metadata) GetDatetime() string {
	t := time.Unix(m.timestamp, 0)
	localTime := t.Local()
	return localTime.Format(time.RFC822)
}

// METADATA - timestamp, 64 bits
// func EncodeTimestamp() []bool {
// 	timestamp := time.Now().Unix()
// 	timeBytes := make([]byte, 8)
// 	binary.BigEndian.PutUint64(timeBytes, uint64(timestamp))
// 	return bytesToBits(timeBytes)
// }

// METADATA - filename
// func encodeFilename(path string) []bool {
//
// 	filename := path[strings.LastIndex(path, "/")+1:]
// 	// cut too long filename
// 	ext := filepath.Ext(filename)
// 	maxLen := sizeMetadata - len(ext) - 2 // 2 for -- separator/ indicator of cut
// 	if len(filename) > maxLen {
// 		filename = fmt.Sprintf("%s--%s", filename[:maxLen], ext)
// 	}
// 	// add marker to the end of the filename so on decoding we know the length
// 	filename += "/"
// 	return bytesToBits([]byte(filename))
// 	// fmt.Println("filename", filename)
// 	// printBits(filenameBits)
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

func TestConvertUint64ToBytes(t *testing.T) {

	testCases := []struct {
		name string
		num  uint64
		want []byte
	}{
		{
			name: "Test 1",
			num:  1234567890,
			want: []byte{0x49, 0x96, 0x2d, 0x2, 0x0, 0x0, 0x0, 0x0},
		},
		{
			name: "Test 2",
			num:  9876543210,
			want: []byte{0x49, 0x96, 0x2d, 0x2e, 0x0, 0x0, 0x0, 0x0},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := convertUint64ToBytes(tc.num)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestBytesToBits(t *testing.T) {
	testCases := []struct {
		name  string
		bytes []byte
		want  []bool
	}{
		{
			name:  "Test case 1: Single byte",
			bytes: []byte{0x2}, // Binary: 00000010
			want:  []bool{false, true, false, false, false, false, false, false},
		},
		{
			name:  "Test case 2: Multiple bytes",
			bytes: []byte{0x2, 0x3}, // Binary: 00000010 00000011
			want:  []bool{false, true, false, false, false, false, false, false, true, true, false, false, false, false, false, false},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := bytesToBits(tc.bytes)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
