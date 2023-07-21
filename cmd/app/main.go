package main

import (
	"bytereel/pkg/core"
	"bytereel/pkg/logger"
	"flag"
	"fmt"
	"os"
	"runtime/pprof"

	"github.com/urfave/cli"
)

var app = cli.NewApp()
var log = logger.Log
var pprofFlag = flag.Bool("pprof", false, "enable pprof profiling")

// global vars to be filled via build args and later used in api
var version string

func init() {
	cr := core.NewCore()
	app.Name = "bytereel"
	app.Usage = "A file to video converter"
	app.UsageText = "bytereel [command] filename"
	app.HideHelp = true
	app.HideVersion = false
	app.Version = version
	app.ArgsUsage = ""
	app.EnableBashCompletion = true
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
				return cr.Encode(filename)
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
				_, err = cr.Decode(filename)
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
				same, err := cr.Compare(filename)
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
	flag.Parse()

	args := os.Args
	if *pprofFlag {
		if len(args) < 2 {
			log.Fatal("pprof filename is required")
		}
		filename := args[1]
		fmt.Println("Profiling enabled")
		f, err := os.Create(fmt.Sprintf("%s.pprof", filename))
		if err != nil {
			log.Fatal(err)
		}
		_ = pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
		// cut off the pprof flag
		args = args[2:]
	}

	err := app.Run(args)
	if err != nil {
		log.Fatal(err)
	}
}
