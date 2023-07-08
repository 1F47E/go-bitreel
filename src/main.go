package main

import (
	"bytereel/pkg/core"
	"os"

	"bytereel/pkg/logger"
)

var log = logger.Log

func main() {

	c := core.NewCore()
	// read cmd line args
	args := os.Args[1:]
	if len(args) < 2 {
		log.Error("d dir - decode pics in a dir, e file - encode a file")
	}
	command := args[0]
	arg := args[1]
	if command == "d" {
		_, err := c.Decode(arg)
		if err != nil {
			log.Fatalf("Error decoding video: %v", err)
		}
	} else if command == "e" {
		err := c.Encode(arg)
		if err != nil {
			log.Fatalf("Error encoding video: %v", err)
		}
	} else if command == "test" {
		// encode + decode + compare
		err := c.Encode(arg)
		if err != nil {
			log.Fatalf("Error encoding video: %v", err)
		}
		videoFile := "tmp/out.mov"
		out, err := c.Decode(videoFile)
		if err != nil {
			log.Fatalf("Error decoding video: %v", err)
		}
		// compare files
		same, err := c.Compare(arg, out)
		if err != nil {
			log.Fatalf("Error comparing files: %v", err)
		}
		// assert if not same
		if !same {
			log.Fatalf("Error: files are not the same")
		} else {
			log.Println("Success: files are the same")
		}
	}

}
