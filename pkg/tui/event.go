package tui

type eventType int

const (
	eventTypeSpin eventType = iota
	eventTypeBar
	eventTypeText
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

func NewEventText(text string) Event {
	return Event{
		eventType: eventTypeText,
		text:      text,
	}
}
