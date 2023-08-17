package meta

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"path/filepath"
	"strings"
	"time"

	cfg "github.com/1F47E/go-bitreel/internal/config"
	"github.com/1F47E/go-bitreel/internal/logger"
)

type Metadata struct {
	Filename  string
	timestamp int64
	checksum  uint64
}

func New(path string) Metadata {
	return Metadata{
		Filename:  encodeFilename(path),
		timestamp: time.Now().Unix(),
	}
}

// METADATA parsing
func Parse(header []byte) (Metadata, error) {
	log := logger.Log.WithField("scope", "meta parser")
	log.Debug("Parsing metadata")
	log.Debug("Header len: ", len(header))
	log.Debugf("Header: %v\n", header)

	checksumBytes := header[:8]
	timestampBytes := header[8:16]
	timestamp := int64(binary.BigEndian.Uint64(timestampBytes))

	filenameBytes := header[16:]
	// find end of the filename by marker
	end := strings.Index(string(filenameBytes), cfg.MetadataEOFMarker)
	filename := string(filenameBytes[:end])

	checksum := binary.BigEndian.Uint64(checksumBytes)
	m := Metadata{
		Filename:  filename,
		timestamp: timestamp,
		checksum:  checksum,
	}
	return m, nil
}

func (m *Metadata) IsOk() bool {
	if len(m.Filename) > 0 && m.timestamp > 0 {
		return true
	}
	return false
}

func (m *Metadata) Print() string {
	return fmt.Sprintf("Filename: %s, Timestamp: %d (%s)", m.Filename, m.timestamp, m.FormatDatetime())
}

func (m *Metadata) FormatDatetime() string {
	t := time.Unix(m.timestamp, 0)
	localTime := t.Local()
	return localTime.Format(time.RFC822)
}

func (m *Metadata) Checksum() uint64 {
	return m.checksum
}

// validate
func (m *Metadata) Validate(buff []byte) (bool, error) {
	checksum, err := generateChecksum(&buff)
	if err != nil {
		return false, err
	}
	return checksum == m.checksum, nil
}

func (m *Metadata) Hash(bytes []byte) ([]bool, error) {
	log := logger.Log.WithField("scope", "meta hasher")

	header := make([]byte, cfg.SizeMetadata)
	checksum, err := generateChecksum(&bytes)
	if err != nil {
		return nil, err
	}

	checksumBytes := convertUint64ToBytes(checksum)

	// copy checksum to header
	s := 0
	l := len(checksumBytes)
	copy(header[s:l], checksumBytes[:])
	log.Debugf("META:Checksum bytes: %v\n", checksumBytes)

	// copy timestamp
	tsBytes := convertUint64ToBytes(uint64(m.timestamp))
	binary.BigEndian.PutUint64(tsBytes, uint64(m.timestamp))
	log.Debugf("META:Timestamp bytes: %v\n", tsBytes)
	s = l
	l = s + 8
	copy(header[s:l], tsBytes[:])

	// copy filename
	fnBytes := make([]byte, len(m.Filename))
	copy(fnBytes, []byte(m.Filename))
	log.Debugf("META:Filename bytes: %v\n", fnBytes)
	s = l
	l = s + len(m.Filename)
	copy(header[s:l], fnBytes[:])

	return bytesToBits(header), nil
}

// get datetime in users format
func (m *Metadata) GetDatetime() string {
	t := time.Unix(m.timestamp, 0)
	localTime := t.Local()
	return localTime.Format(time.RFC822)
}

func generateChecksum(bytes *[]byte) (uint64, error) {
	hasher := fnv.New64a()
	_, err := hasher.Write(*bytes)
	if err != nil {
		return 0, fmt.Errorf("META:Error writing to hasher")
	}
	return hasher.Sum64(), nil
}

func convertUint64ToBytes(num uint64) []byte {
	byteArray := make([]byte, 8)
	binary.BigEndian.PutUint64(byteArray, num)
	return byteArray
}

func encodeFilename(path string) string {
	filename := path[strings.LastIndex(path, "/")+1:]
	if len(filename) > cfg.MetadataMaxFilenameLen {
		ext := filepath.Ext(filename) // with a dot
		maxLen := cfg.MetadataMaxFilenameLen - len(ext) - len(cfg.MetadataFilenameCutDelimeter) - len(cfg.MetadataEOFMarker)
		filename = filename[:maxLen] + cfg.MetadataFilenameCutDelimeter + ext
	}
	// add marker to the end of the filename so on decoding we know the end
	filename += cfg.MetadataEOFMarker
	return filename
}

func bytesToBits(bytes []byte) []bool {
	bits := make([]bool, 8*len(bytes))
	for i, b := range bytes {
		for j := 0; j < 8; j++ {
			bits[i*8+j] = (b & (1 << uint(j))) != 0
		}
	}
	return bits
}
