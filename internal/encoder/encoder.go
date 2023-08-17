package encoder

import (
	// cfg "github.com/1F47E/go-bitreel/internal/config"
	"image"
	"image/color"

	"github.com/1F47E/go-bitreel/internal/logger"
	"github.com/1F47E/go-bitreel/internal/meta"
	"github.com/1F47E/go-bitreel/internal/storage"
)

const pixelSize = 4

type FrameEncoder struct {
	width    int
	height   int
	sizeBits int
}

func NewFrameEncoder(width, height int) *FrameEncoder {
	// calc frame size in bits based on our pixel size
	sizeBits := width * height / pixelSize
	return &FrameEncoder{width, height, sizeBits}
}

func (f *FrameEncoder) EncodeFrame(data []byte, m meta.Metadata) *image.NRGBA {
	log := logger.Log.WithField("scope", "frame encoder")
	log.Debug("Encoding frame")

	// craete buffer, get and copy metadata - filename, timestamp and checksum
	bufferBits := make([]bool, f.sizeBits)
	metadataBits, err := m.Hash(data)
	if err != nil {
		log.Fatal("Cannot hash metadata:", err)
	}
	copy(bufferBits, metadataBits)

	// range over data bit by bit and encode every bit as a pixel
	var bitIndex int
	for i := 0; i < len(data); i++ {
		for k := 0; k < 8; k++ {
			// shift by metadata size header
			bitIndex = len(metadataBits) + i*8 + k
			// get the current bit of the byte
			// buf[i]:     0 1 1 0 1 0 0 1
			// (1<<j):     0 0 0 0 1 0 0 0  (1 has been shifted left 3 times)
			//					   ^
			// --------------------------------
			// result:     0 0 0 0 1 0 0 0  (bitwise AND operation)
			//					   ^
			// write the bit to the buffer
			bufferBits[bitIndex] = (data[i] & (1 << uint(k))) != 0
		}
	}

	// generate image
	writeIdx := 0
	var col color.Color
	img := image.NewNRGBA(image.Rect(0, 0, f.width, f.height))
	for x := 0; x < f.width; x += 2 {
		for y := 0; y < f.height; y += 2 {
			// detect file end
			if writeIdx <= bitIndex {
				if bufferBits[writeIdx] {
					col = color.NRGBA{0, 0, 0, 255} // black
				} else {
					col = color.NRGBA{255, 255, 255, 255} // white
				}
			} else {
				col = color.NRGBA{255, 0, 0, 255} // red
			}
			// Set a 2x2 block of pixels to the color.
			img.Set(x, y, col)
			img.Set(x+1, y, col)
			img.Set(x, y+1, col)
			img.Set(x+1, y+1, col)
			writeIdx++
		}
	}

	log.Debug("Encoding frame done")
	return img
}

func (f *FrameEncoder) DecodeFrame(filename string) ([]byte, int) {
	log := logger.Log.WithField("scope", "frame decoder")
	img, err := storage.FrameRead(filename)
	if err != nil {
		log.Fatal("Cannot decode file:", err)
	}

	// copy image to bytes
	// black = 1, white = 0, red = EOF
	var writeIdx int
	var cntBlack, cntWhite, cntRed uint
	var pixelErrorsCount int
	// var fileBits [cfg.sizeBits]bool
	fileBits := make([]bool, f.sizeBits)
	for x := 0; x < f.width; x += 2 {
		for y := 0; y < f.height; y += 2 {
			// error detection
			// count black and white pixels in a 2x2 square
			for i := 0; i < 2; i++ {
				for j := 0; j < 2; j++ {
					col := img.At(x+i, y+j)
					// this will return 0-65535 range
					r, g, b, _ := col.RGBA()
					// shift 8 bits to the right to have 0-255 range
					r8, g8, b8 := r>>8, g>>8, b>>8

					// detect red first
					if r8 > 128 && g8 < 128 && b8 < 128 {
						cntRed++
					} else if r8 > 128 && g8 > 128 && b8 > 128 {
						cntWhite++
					} else {
						cntBlack++
					}
				}
			}

			// skip if reached red section which is EOF
			if cntRed > cntBlack && cntRed > cntWhite {
				// reset counters
				cntRed = 0
				cntBlack = 0
				cntWhite = 0
				continue
			}

			if cntBlack > cntWhite {
				fileBits[writeIdx] = true
				if cntBlack != 4 {
					// fmt.Println("Error at black ", x, y, cntBlack, cntWhite)
					pixelErrorsCount++
				}
			} else {
				fileBits[writeIdx] = false
				if cntWhite != 4 {
					// fmt.Println("Error at white ", x, y, cntBlack, cntWhite)
					pixelErrorsCount++
				}
			}
			cntRed = 0
			cntBlack = 0
			cntWhite = 0
			writeIdx++
		}
	}
	if pixelErrorsCount > 0 {
		log.Println()
		log.Warnf("Pixel errors (%d) corrected in frame: %s\n", pixelErrorsCount, filename)
	}

	// 4k video is 3840x2160 = 8294400 pixels = 2073600 4px blocks
	// every frame should have 2073600 bits
	// convert bits to bytes
	bytes := make([]byte, len(fileBits)/8)
	for i := 0; i < len(fileBits); i += 8 {
		var b byte
		for j := 0; j < 8; j++ {
			if fileBits[i+j] {
				b |= 1 << uint(j)
			}
		}
		bytes[i/8] = b
	}
	writtenBytes := writeIdx / 8
	return bytes, writtenBytes
}
