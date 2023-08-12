package tui

import (
	"context"
)

type TUI struct {
	ctx      context.Context
	eventsCh chan Event
}

func New(eventsCh chan Event, ctx context.Context) *TUI {
	return &TUI{ctx, eventsCh}
}

func (t *TUI) Run() {
	// init bubbletea spinner and progress bar
	widget := NewWidget()
	go widget.Run()

	// read events from channel and update spinner/progress bar
	for {
		select {
		case <-t.ctx.Done():
			return

		case event := <-t.eventsCh:
			switch event.eventType {
			case eventTypeSpin:
				widget.SetSpinner(event.text)
			case eventTypeBar:
				widget.SetProgress(event.text, event.percent)
			case eventTypeText:
				widget.SetText(event.text)
			}
		}
	}
}
