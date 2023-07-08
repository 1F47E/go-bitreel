package main

import (
	"bytereel/pkg/core"
	"bytereel/pkg/logger"
	"os"
)

var log = logger.Log

func main() {

	// read cmd line args
	args := os.Args[1:]
	if len(args) < 2 {
		log.Info("e file - encode a file")
		log.Info("d dir - decode pics in a dir")
		log.Info("test file - encode, decode, compare")
		return
	}
	command := args[0]
	arg := args[1]
	if command == "d" {
		_, err := core.Decode(arg)
		if err != nil {
			log.Fatalf("Error decoding video: %v", err)
		}
	} else if command == "e" {
		err := core.Encode(arg)
		if err != nil {
			log.Fatalf("Error encoding video: %v", err)
		}
	} else if command == "test" {
		// encode + decode + compare
		err := core.Encode(arg)
		if err != nil {
			log.Fatalf("Error encoding video: %v", err)
		}
		videoFile := "tmp/out.mov"
		out, err := core.Decode(videoFile)
		if err != nil {
			log.Fatalf("Error decoding video: %v", err)
		}
		// compare files
		same, err := core.Compare(arg, out)
		if err != nil {
			log.Fatalf("Error comparing files: %v", err)
		}
		// assert if not same
		if !same {
			log.Fatalf("Error: files are not the same")
		} else {
			log.Println("Success: files are the same")
		}
		// cleanup
		_ = os.Remove(videoFile)
		_ = os.Remove(out)
	}

}
