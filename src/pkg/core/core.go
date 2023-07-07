package core

import (
	"sync"

	"github.com/schollz/progressbar/v3"
)

// Amount of bits in 1 4k frame
// 3840*2160/4 = 2073600
// const frameSizeBits = 2073600 bits
const frameWidth = 3840
const frameHeight = 2160
const frameFileSize = 7684000 // estimated

const pixelSize = 4

// NOTE: img pixels are writter from left to right, top to bottom

// reserve first row for metadata
// in 4k its 2160 pixels / 4 = 540 bytes
// example
// 64 bits for checksum - 8 bytes
// unix timestamp - 8 bytes
// filename - the rest - 524 bytes
const metadataMaxFilenameLen = 524
const metadataSizeBits = frameWidth / pixelSize

// 250kb on 4k
const frameSizeBits = frameWidth * frameHeight / pixelSize
const frameBufferSizeBits = frameSizeBits + metadataSizeBits

type Core struct {
	progress *progressbar.ProgressBar
	Wg       sync.WaitGroup
}

func NewCore() *Core {
	return &Core{
		progress: nil,
		Wg:       sync.WaitGroup{},
	}
}

// reinit progress bar becuase of some bug
func (c *Core) ResetProgress(max int, desc string) {
	if c.progress != nil {
		c.progress.Clear()
		c.progress.Reset()
	}
	c.progress = progressbar.NewOptions(max,
		progressbar.OptionSetDescription(desc),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionSetDescription(desc))
}
