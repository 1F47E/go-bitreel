package core

import (
	"context"

	"github.com/1F47E/go-bitreel/internal/tui"
	"github.com/1F47E/go-bitreel/internal/workers"
)

type Core struct {
	ctx      context.Context
	logCh    chan string
	eventsCh chan tui.Event
	worker   *workers.Worker
}

func NewCore(ctx context.Context, eventsCh chan tui.Event) *Core {
	return &Core{
		ctx:      ctx,
		logCh:    make(chan string),
		eventsCh: eventsCh,
		worker:   workers.NewWorker(ctx),
	}
}
