package core

import (
	"context"

	"github.com/1F47E/go-bytereel/pkg/tui"
	"github.com/1F47E/go-bytereel/pkg/workers"
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
