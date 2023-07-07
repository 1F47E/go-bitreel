package core

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"log"
	"os"
	"sort"
	"strings"
	"time"
)

func (c *Core) Decode(dir string) error {
	c.ResetProgress(-1, "Decoding...") // spinner
	// scan the directory for files
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	// get list of files
	filesList := make([]string, 0, len(files))
	for _, file := range files {
		// get file path
		path := dir + "/" + file.Name()
		fmt.Println(path)
		// check if the name has "output_" prefix
		if strings.HasPrefix(file.Name(), "out_") {
			// add to the list
			filesList = append(filesList, path)
		}
	}
	if len(filesList) == 0 {
		log.Fatal("No files to decode")
	}
	fmt.Println("total files:", len(filesList))
	sort.Strings(filesList)

	// filename with timestamp
	// outputFilename := "tmp/decoded.bin.txt"
	// outputFilenameTmp := fmt.Sprintf("tmp/decoded_%d.bin", time.Now().Unix())

	// setup progress bar
	c.ResetProgress(len(filesList), "Decoding...")

	// create a output file
	// TODO: get output filename from metadata
	// f, err := os.Create(outputFilenameTmp)
	// if err != nil {
	// 	log.Fatalf("Cannot create file: %s - %v", outputFilenameTmp, err)
	// }

	// Create a temporary file in the same directory
	tmpFile, err := os.CreateTemp("", "decoded-")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tmpFile.Name()) // clean up

	// Create the destination file, owerwriting existing ones
	// tempFilePath := fmt.Sprintf("tmp/decoded_%d.bin", time.Now().Unix())
	//
	// err = os.MkdirAll(filepath.Dir(tempFilePath), 0755)
	// if err != nil {
	// 	log.Fatal("Cannot create dir:", err)
	// }
	// destFile, err := os.Create(outputPath)
	// if err != nil {
	// 	log.Fatal("Cannot create file:", err)
	// }
	// defer destFile.Close()

	// Open the file for appending. with unixtime
	// tmpFile, err := os.OpenFile(tempFilePath, os.O_APPEND|os.O_WRONLY, 0644)
	// if err != nil {
	// 	log.Fatal("Cannot open file:", err)
	// }
	// // defer tmpFile.Close()
	// defer os.Remove(tmpFile.Name()) // clean up

	var bytesWritten, pixelErrorsCount int
	var metaTime int64
	var metaDatetime string
	var metaFilename string
	var metaChecksum uint64
	for _, file := range filesList {
		// fmt.Println("Decoding", file)
		frameBytes, cnt := decodeFrame(file)
		pixelErrorsCount += cnt

		// cut metadata
		data := frameBytes[metadataSizeBits/8:]
		meta := frameBytes[:metadataSizeBits/8]

		// write data to the file
		written, err := tmpFile.Write(data)
		if err != nil {
			log.Fatal("Cannot write to file:", err)
		}
		bytesWritten += written
		err = c.progress.Add(1)
		if err != nil {
			log.Fatal("Cannot update progress bar:", err)
		}

		// METADATA parsing

		// check checksum
		if metaChecksum == 0 {
			checksumBytes := meta[:8]
			// convert bytes to uint64
			metaChecksum = binary.BigEndian.Uint64(checksumBytes)
		}

		if metaTime == 0 {
			timeBytes := meta[8:16]
			metaTime = int64(binary.BigEndian.Uint64(timeBytes))
			fmt.Println("METADATA FOUND: time:", metaTime)
		}
		if metaFilename == "" {
			metaFilenameBuff := meta[16:]
			bStr := string(metaFilenameBuff)
			fmt.Println("METADATA FOUND: filename buf:", bStr, "len", len(bStr))
			// cut filename to size
			// search for the market "end of filename" - byte "/"
			delimiterIndex := strings.Index(bStr, "/")
			if delimiterIndex != -1 {
				metaFilename = bStr[:delimiterIndex]
				fmt.Println("METADATA FOUND: filename cut:", metaFilename, "len", len(metaFilename))

				// CHECK FILENAME
				compareString := "test.png"
				compareBytes := []byte(compareString)

				minLen := len(metaFilename)
				if len(compareBytes) < minLen {
					minLen = len(compareBytes)
				}

				for i := 0; i < minLen; i++ {
					if metaFilename[i] != compareBytes[i] {
						fmt.Printf("Bytes differ at index %d: metaFilename byte is %d, compareString byte is %d\n",
							i, metaFilename[i], compareBytes[i])
					}
				}

				if len(metaFilename) != len(compareBytes) {
					fmt.Println("Lengths differ: metaFilename length is", len(metaFilename), "compareString length is", len(compareBytes))
				}
			}
		}
	}

	// close the file so we can rename it
	// Ensure data is written to disk
	// err = tmpFile.Sync()
	// if err != nil {
	// 	log.Fatal("Cannot sync file:", err)
	// }
	//
	// Close the file before renaming/moving it
	err = tmpFile.Close()
	if err != nil {
		log.Fatal("Cannot close file:", err)
	}

	// check metadata
	// report time from metadata
	if metaTime != 0 {
		metaunix := time.Unix(metaTime, 0)
		// format to datetime
		fmt.Println("Time from metadata:", metaunix.Format("2006-01-02 15:04:05"))
		metaDatetime = metaunix.Format("2006-01-02_15-04-05")
	}
	if metaFilename != "" {
		fmt.Println("Filename found in metadata:", metaFilename)
		// rename tmp file to the original filename
		if metaDatetime != "" {
			metaFilename = fmt.Sprintf("%s_%s", metaDatetime, metaFilename)
		}
		outputFilename := fmt.Sprintf("decoded_%s", metaFilename)
		// log filename from metadata by bytes
		// fmt.Println("Filename from metadata1:", string(metaFilename))
		// fmt.Println("Filename from metadata2:", metaFilename)
		// fmt.Println("Filename from metadata2:", []rune(metaFilename))
		// fmt.Printf("Filename from metadata runes:")
		// for _, r := range []rune(metaFilename) {
		// 	fmt.Printf("%c", r)
		// }
		// fmt.Println()
		// outputFilename = "2023-07-07_10-12-35_test.png"
		// outputFilename = "decoded_2023-07-07_10-12-35_test.png"
		fmt.Println("Renaming", tmpFile.Name(), "to", outputFilename)
		// create path to the file
		err = os.Rename(tmpFile.Name(), outputFilename)
		if err != nil {
			log.Fatal("Cannot rename file:", err)
		}

		// // check curr dir
		// dir, err := os.Getwd()
		// if err != nil {
		// 	log.Fatal(err)
		// }
		// fmt.Println("Current dir:", dir)
		//
		// // outputPath := dir + "/tmp2/" + outputFilename
		// outputPath := dir + "/tmp2/test.bin"
		//
		// err = os.MkdirAll(filepath.Dir(outputPath), 0755)
		// if err != nil {
		// 	log.Fatal("Cannot create dir:", err)
		// }
		//
		// // Create the destination file, owerwriting existing ones
		// destFile, err := os.Create(outputPath)
		// if err != nil {
		// 	log.Fatal("Cannot create file:", err)
		// }
		// defer destFile.Close()
		//
		// // Copy the file contents from source to destination
		// _, err = io.Copy(destFile, tmpFile)
		// if err != nil {
		// 	log.Fatal("Cannot copy file:", err)
		// }
		log.Printf("\n\nWrote %d bytes to %s\n", bytesWritten, outputFilename)
		if pixelErrorsCount > 0 {
			log.Printf("Pixel errors corrected: %d\n", pixelErrorsCount)
		}

		fmt.Println("Renamed", tmpFile.Name(), "to", outputFilename)
	} else {
		fmt.Println("No filename found in metadata")
	}
	return nil
}

// TODO: make this a worker
func decodeFrame(filename string) ([]byte, int) {

	// read the image
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal("Cannot open file:", err)
	}
	defer file.Close()
	imgRaw, err := png.Decode(file)
	if err != nil {
		log.Fatal("Cannot decode file:", err)
	}
	// TODO: test later is this needed
	// Create an empty NRGBA image with the same size as the source image.
	img := image.NewNRGBA(imgRaw.Bounds())
	// Draw the source image onto the new NRGBA image.
	draw.Draw(img, img.Bounds(), imgRaw, imgRaw.Bounds().Min, draw.Src)

	var bits [frameSizeBits]bool

	// copy image to bytes
	var k, cntBlack, cntWhite uint
	var pixelErrorsCount int
	for x := 0; x < img.Bounds().Dx(); x += 2 {
		for y := 0; y < img.Bounds().Dy(); y += 2 {
			// error detection
			// count black and white pixels in a 2x2 square
			for i := 0; i < 2; i++ {
				for j := 0; j < 2; j++ {
					col := img.At(x+i, y+j)
					// this will return 0-65535 range
					r, g, b, _ := col.RGBA()
					// shift 8 bits to the right to have 0-255 range
					r8, g8, b8 := r>>8, g>>8, b>>8

					if r8 > 128 && g8 > 128 && b8 > 128 {
						cntWhite++
					} else {
						cntBlack++
					}
				}
			}
			if cntBlack > cntWhite {
				bits[k] = true
				if cntBlack != 4 {
					// fmt.Println("Error at black ", x, y, cntBlack, cntWhite)
					pixelErrorsCount++
				}
			} else {
				bits[k] = false
				if cntWhite != 4 {
					// fmt.Println("Error at white ", x, y, cntBlack, cntWhite)
					pixelErrorsCount++
				}
			}
			cntBlack = 0
			cntWhite = 0
			k++
		}
	}

	// 4k video is 3840x2160 = 8294400 pixels = 2073600 4px blocks
	// every frame should have 2073600 bits
	// convert bits to bytes
	bytes := make([]byte, len(bits)/8)
	for i := 0; i < len(bits); i += 8 {
		var b byte
		for j := 0; j < 8; j++ {
			if bits[i+j] {
				b |= 1 << uint(j)
			}
		}
		bytes[i/8] = b
	}
	// fmt.Println("bytes len:", len(bytes))

	return bytes, pixelErrorsCount

}
