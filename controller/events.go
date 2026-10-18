package controller

type EventType string

const EVENT_TYPE_INPUT = "input"
const EVENT_TYPE_CONTROLLER = "controller-state"

type Event struct {
	Type       EventType
	Target     uint16
	Value      int32
	Normalized float64
	IsAxis     bool
}

var channel = make(chan Event)

func sendEvent(event Event) {
	channel <- event
}

func ListenForOneEvent() Event {
	msg := <-channel
	return msg
}
