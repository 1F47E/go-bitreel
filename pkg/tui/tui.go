package tui

import (
	"context"

	"github.com/1F47E/go-bytereel/pkg/logger"
)

type eventType int

const (
	eventTypeSpin eventType = iota
	eventTypeBar
)

type Event struct {
	eventType eventType
	text      string
	percent   float64
}

func NewEventSpin(text string) Event {
	return Event{
		eventType: eventTypeSpin,
		text:      text,
	}
}

func NewEventBar(text string, percent float64) Event {
	return Event{
		eventType: eventTypeBar,
		text:      text,
		percent:   percent,
	}
}

type TUI struct {
	ctx      context.Context
	eventsCh chan Event
}

func New(eventsCh chan Event, ctx context.Context) *TUI {
	return &TUI{ctx, eventsCh}
}

func (t *TUI) Run() {
	log := logger.Log

	loader := NewProgress()
	go loader.Run()

	for {
		select {
		// exit TUI
		case <-t.ctx.Done():
			log.Warn("tui ctx done")
			return

		case event := <-t.eventsCh:
			// log.Warnf("event: %+v", event)
			if event.eventType == eventTypeSpin {
				loader.UpdateSpinner(event.text)
			} else if event.eventType == eventTypeBar {
				loader.UpdateProgress(event.text, event.percent)
			}
			// switch event.Type {
			// case "spinner":
		}
	}
}
