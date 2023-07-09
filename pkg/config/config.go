package config

// NOTE: img pixels are writter from left to right, top to bottom
const (
	FrameWidth    = 3840
	FrameHeight   = 2160
	FrameFileSize = 7684000 // estimated

	// all sizes are in bytes
	SizeFrameWidth  = 3840
	SizeFrameHeight = 2160
	SizeFrameFile   = 7684000 // estimated
	SizePixel       = 4
	SizeMetadata    = 256

	// meta
	MetadataMaxFilenameLen       = 524 // size left in the meta header
	MetadataEOFMarker            = "/"
	MetadataFilenameCutDelimeter = "--"

	// 250kb on 4k
	// on 4k 3840*2160/4/8 = 259200 bytes = about 250kb
	SizeFrame = SizeFrameWidth * SizeFrameHeight / SizePixel / 8

	// Path
	PathFramesDir = "tmp/frames"
	PathVideoOut  = "tmp/out.mov"
)
