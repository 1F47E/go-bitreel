package core

import (
	"bytereel/pkg/logger"

	"github.com/schollz/progressbar/v3"
)

var log = logger.Log

// NOTE: img pixels are writter from left to right, top to bottom

// 4k
const frameWidth = 3840
const frameHeight = 2160
const frameFileSize = 7684000 // estimated

// all sizes are in bytes
const sizeFrameWidth = 3840
const sizeFrameHeight = 2160
const sizeFrameFile = 7684000 // estimated
const sizePixel = 4
const sizeMetadata = 256

// 250kb on 4k
// on 4k 3840*2160/4/8 = 259200 bytes = about 250kb
const sizeFrame = sizeFrameWidth * sizeFrameHeight / sizePixel / 8

type Core struct {
	progress *progressbar.ProgressBar
	// Wg       sync.WaitGroup
}

func NewCore() *Core {
	return &Core{
		progress: nil,
		// Wg:       sync.WaitGroup{},
	}
}

// reinit progress bar becuase of some bug
func (c *Core) ResetProgress(max int, desc string) {
	if c.progress != nil {
		_ = c.progress.Clear()
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
