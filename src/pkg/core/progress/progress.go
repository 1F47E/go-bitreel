package progress

import "github.com/schollz/progressbar/v3"

var Progress = progressCreate(-1, "") // init as spinner

func ProgressSpinner(desc string) {
	_ = Progress.Clear()
	ProgressReset(-1, desc)
	_ = Progress.RenderBlank()
}

func ProgressReset(max int, desc string) {
	Progress = progressCreate(max, desc)
}

func Add(n int) {
	_ = Progress.Add(n)
}

func Set(n int) {
	_ = Progress.Set(n)
}

func Max(n int) {
	Progress.ChangeMax(n)
}

func Finish() {
	_ = Progress.Finish()
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

