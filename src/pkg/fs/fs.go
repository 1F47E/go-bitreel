// All files related functions
package fs

import (
	cfg "bytereel/pkg/config"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var framesDir = cfg.PathFramesDir

func CreateFramesDir() (string, error) {
	err := os.MkdirAll(framesDir, os.ModePerm)
	if err != nil {
		return framesDir, fmt.Errorf("Error creating frames dir: %s", err)
	}
	return framesDir, nil
}

func ScanFrames() ([]string, error) {
	files, err := os.ReadDir(framesDir)
	if err != nil {
		return nil, err
	}
	// filter out files
	filesList := make([]string, 0, len(files))
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "out_") {
			filesList = append(filesList, framesDir+"/"+file.Name())
		}
	}
	if len(filesList) == 0 {
		return nil, fmt.Errorf("No files to decode")
	}
	sort.Strings(filesList)
	return filesList, nil
}

func CreateTempFile() (*os.File, error) {
	// Create a temporary file in the same directory
	tmpFile, err := os.CreateTemp("", "decoded-")
	if err != nil {
		return nil, err
	}
	return tmpFile, nil
}

// Save decoded
// Write the data to the file and clear tmp folder with frames
func SaveDecoded(tmpFile *os.File, filename string) error {
	err := tmpFile.Sync()
	if err != nil {
		return err
	}
	err = tmpFile.Close()
	if err != nil {
		return err
	}
	err = os.Rename(tmpFile.Name(), filename)
	if err != nil {
		return err
	}
	err = os.RemoveAll(framesDir)
	if err != nil {
		return err
	}
	return nil
}

func SaveFrame(frameNum int, img *image.NRGBA) error {
	filePath := fmt.Sprintf("tmp/out/out_%08d.png", frameNum)
	// make sure dir exists - create all
	err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
	if err != nil {
		log.Println("Cannot create dir:", err)
		// panic(fmt.Sprintf("Cannot create dir: %s", err))
		return fmt.Errorf("Cannot create tmp out dir for path %s: %s", filePath, err)
	}

	imgFile, err := os.Create(filePath)
	defer imgFile.Close()
	if err != nil {
		log.Println("Cannot create file:", err)
		return fmt.Errorf("Cannot create file: %s", err)
	}
	err = png.Encode(imgFile, img.SubImage(img.Rect))
	if err != nil {
		log.Println("Cannot encode to file:", err)
		return fmt.Errorf("Cannot encode to file: %s", err)
	}
	return nil
}

func FrameRead(filaneme string) (image.Image, error) {
	// read the image
	file, err := os.Open(filaneme)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	img, err := png.Decode(file)
	if err != nil {
		return nil, err
	}
	return img, nil
}
