package main

import (
	"bytereel/pkg/core"
	"bytereel/pkg/logger"
	"os"
)

var log = logger.Log

func main() {

	// TODO: make nice cli with flags

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
		same, err := core.Compare(arg)
		if err != nil {
			log.Fatalf("Error comparing video: %v", err)
		}
		if same {
			log.Info("Files are the same")
		} else {
			log.Error("Files are different")
		}

	}

}
