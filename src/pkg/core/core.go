package core

import (
	"bytereel/pkg/logger"
	"os"

	"github.com/schollz/progressbar/v3"
)

var log = logger.Log

// TODO: extract to sep pkg
var progress = progressCreate(-1, "") // init as spinner

func ProgressSpinner(desc string) {
	ProgressReset(-1, desc)
	_ = progress.RenderBlank()
}

func ProgressReset(max int, desc string) {
	progress = progressCreate(max, desc)
	_ = progress.RenderBlank()
}

func progressCreate(max int, desc string) *progressbar.ProgressBar {
	return progressbar.NewOptions(max,
		progressbar.OptionSetDescription(desc),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]/[reset]",
			SaucerHead:    "[green]/[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))
}

// Compare files before and after decoding for test command
func Compare(file1, file2 string) (bool, error) {
	log.Info("Comparing files...")
	// read files
	b1, err := os.ReadFile(file1)
	if err != nil {
		return false, err
	}
	b2, err := os.ReadFile(file2)
	if err != nil {
		return false, err
	}
	// compare
	if len(b1) != len(b2) {
		log.Fatal("Files are not the same size")
		return false, nil
	}
	for i := 0; i < len(b1); i++ {
		if b1[i] != b2[i] {
			log.Info("Files are not the same at position", i)
			return false, nil
		}
	}
	return true, nil
}
