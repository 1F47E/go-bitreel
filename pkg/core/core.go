package core

import (
	"bytereel/pkg/logger"
	"bytereel/pkg/workers"
)

var log = logger.Log

type Core struct {
	logCh  chan string
	worker *workers.Worker
}

func NewCore() *Core {
	return &Core{
		logCh:  make(chan string),
		worker: workers.NewWorker(),
	}
}
