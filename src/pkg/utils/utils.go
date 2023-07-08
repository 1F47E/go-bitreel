package utils

import (
	"bytereel/pkg/logger"
)

var log = logger.Log

func printBits(bits []bool) {
	for _, b := range bits {
		if b {
			log.Info("1")
		} else {
			log.Info("0")
		}
	}
	log.Debug()
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
