package core

import (
	"context"

	"github.com/1F47E/go-bytereel/pkg/logger"
	"github.com/1F47E/go-bytereel/pkg/workers"
)

var log = logger.Log

type Core struct {
	ctx    context.Context
	logCh  chan string
	worker *workers.Worker
}

func NewCore(ctx context.Context) *Core {
	return &Core{
		ctx:    ctx,
		logCh:  make(chan string),
		worker: workers.NewWorker(ctx),
	}
}
