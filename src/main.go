package main

import (
	"bytereel/pkg/core"
	"fmt"
	"log"
	"os"
)

func main() {
	c := core.NewCore()
	// read cmd line args
	args := os.Args[1:]
	if len(args) < 2 {
		log.Fatal("d dir - decode pics in a dir, e file - encode a file")
	}
	command := args[0]
	arg := args[1]
	if command == "d" {
		fmt.Println("Decoding", arg)
		err := c.Decode(arg)
		if err != nil {
			log.Fatalf("Error decoding video: %v", err)
		}
	} else if command == "e" {
		fmt.Println("Encoding")
		err := c.Encode(arg)
		if err != nil {
			log.Fatalf("Error encoding video: %v", err)
		}
	}

	c.Wg.Wait()
}
