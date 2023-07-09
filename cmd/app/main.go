package main

import (
	"bytereel/pkg/core"
	"bytereel/pkg/logger"
	"fmt"
	"os"

	"github.com/urfave/cli"
)

var app = cli.NewApp()
var log = logger.Log

func init() {
	app.Name = "bytereel"
	app.Usage = "A file to video converter"
	app.UsageText = "bytereel [command] filename"
	app.HideHelp = true
	app.HideVersion = true
	app.ArgsUsage = ""
	app.Commands = []cli.Command{
		{
			Name:    "encode",
			Aliases: []string{"e"},
			Usage:   "Encode a file",
			Action: func(c *cli.Context) error {
				filename, err := getFilename(c)
				if err != nil {
					return err
				}
				return core.Encode(filename)
			},
		},
		{
			Name:    "decode",
			Aliases: []string{"d"},
			Usage:   "Decode a video",
			Action: func(c *cli.Context) error {
				filename, err := getFilename(c)
				if err != nil {
					return err
				}
				_, err = core.Decode(filename)
				return err
			},
		},
		{
			Name:    "test",
			Aliases: []string{"t"},
			Usage:   "Run encode+decode and compare files",
			Action: func(c *cli.Context) error {
				filename, err := getFilename(c)
				if err != nil {
					return err
				}
				same, err := core.Compare(filename)
				if err != nil {
					return fmt.Errorf("Error comparing video: %v", err)
				}
				if !same {
					return fmt.Errorf("Files are different")
				}
				log.Info("Files are the same")
				return nil
			},
		},
	}
}

func getFilename(c *cli.Context) (string, error) {
	f := c.Args().Get(0)
	if f == "" {
		return "", fmt.Errorf("Filename is required")
	}
	return f, nil
}

func main() {
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
