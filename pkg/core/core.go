package core

import (
	"bytereel/pkg/logger"
	"bytereel/pkg/workers"
	"context"
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
