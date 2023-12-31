//
//
//  █▄▄ █ ▀█▀ █▀█ █▀▀ █▀▀ █░░
//  █▄█ █ ░█░ █▀▄ ██▄ ██▄ █▄▄
//
// convert any file to a video
//

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime/pprof"

	"github.com/1F47E/go-bitreel/internal/core"
	"github.com/1F47E/go-bitreel/internal/logger"
	"github.com/1F47E/go-bitreel/internal/printer"
	"github.com/1F47E/go-bitreel/internal/tui"

	"github.com/urfave/cli"
)

var app = cli.NewApp()
var pprofFlag = flag.Bool("pprof", false, "enable pprof profiling")

// to be filled on build
var version string

func init() {
	app.Name = "bitreel"
	app.Usage = "convert any file to a video"
	app.UsageText = "bitreel [command] filename"
	app.HideHelp = true
	app.HideVersion = false
	app.Version = version
	app.ArgsUsage = ""
	app.EnableBashCompletion = true
}

func main() {
	log := logger.Log
	printer.Banner()

	// profiling
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
		// cut off the pprof flag and filename
		args = args[2:]
	}

	// graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt)
		<-stop
		fmt.Println("Shutting down...")
		cancel()
	}()

	// TUI setup
	tuiEventsCh := make(chan tui.Event)
	t := tui.New(tuiEventsCh, ctx)
	go t.Run()

	// pass events channel to send all the events to the TUI
	appCore := core.NewCore(ctx, tuiEventsCh)

	// on encode command
	fEncode := func(c *cli.Context) error {
		filename, err := getFilename(c)
		if err != nil {
			return err
		}
		return appCore.Encode(filename)
	}

	// on decode command
	fDecode := func(c *cli.Context) error {
		filename, err := getFilename(c)
		if err != nil {
			return err
		}
		_, err = appCore.Decode(filename)
		return err
	}

	// on test command
	fCompare := func(c *cli.Context) error {
		filename, err := getFilename(c)
		if err != nil {
			return err
		}
		same, err := appCore.Compare(filename)
		if err != nil {
			return fmt.Errorf("Error comparing video: %v", err)
		}
		if !same {
			return fmt.Errorf("Files are different")
		}
		log.Info("Files are the same")
		return nil
	}

	app.Commands = []cli.Command{
		cmdBuilder("encode", "e", "Encode a file", fEncode),
		cmdBuilder("decode", "d", "Decode a video", fDecode),
		cmdBuilder("test", "t", "Run encode+decode and compare files", fCompare),
	}

	err := app.Run(args)
	if err != nil {
		log.Fatal(err)
	}
}

func getFilename(c *cli.Context) (string, error) {
	f := c.Args().Get(0)
	if f == "" {
		return "", fmt.Errorf("Filename is required")
	}
	return f, nil
}

func cmdBuilder(name, alias, descr string, f func(c *cli.Context) error) cli.Command {
	return cli.Command{
		Name:    name,
		Aliases: []string{alias},
		Usage:   descr,
		Action:  f,
	}
}
