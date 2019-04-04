// Package chu is the main interface. It contains all the types and
// data structures to work with chu project
package chu

import (
	"errors"
	"time"
)

var (
	ErrGenIDNotDefined = errors.New("GenID function not defined")
)

// GenID needs to be set prior to use this library
// It basically generates ID and AggregateID
var GenID func() string

// Message is a base for data which is being sent by user
type Message interface {
	MsgEncode() ([]byte, error)
	MsgDecode(data []byte) error
}

type Codec interface {
	Encode(ptr interface{}) error
	Decode(ptr interface{}) error
}

type Event interface {
	ID() string
	AggregateID() string
	Topic() string
}

type ReceivedEvent interface {
	Event
	CreatedAt() time.Time
	// Message will be used parse the message from body of event
	Message(ptr Message) error
}

type EventEncoder interface {
	// EvtEncode will be convert the entire Event into bytes
	// This method will be called inside Publish method
	EvtEncode() ([]byte, error)
}

type EventDecoder interface {
	// EvtDecode will be called as soon as message is being received by handler
	// This will decode back the information from bytes to Event
	EvtDecode(data []byte) error
}

type Subscriber interface {
	Topic() string
	Durable() bool
	// Group should be used if round robin is considered. Otherwise return an empty string
	Group() string
	// Once event.Message(msg) is being called, event is passing internal message bytes to
	// msg.MsgDecode(bytes) to populate the Message.
	HandleEvent(event ReceivedEvent) bool
}

type Subscription interface {
	Unsubscribe() error
	Close() error
}

type EventOptions struct {
	AggregateID string  // optional
	Topic       string  // required
	Message     Message // optional
}

type Broker interface {
	Subscribe(sub Subscriber) (Subscription, error)
	// Publish only need to access Two method, "Topic" and "EvtEncode"
	// internall it calls EvtEncode to convert event to bytes and publish
	// that bytes to topic.
	Publish(event Event) error
	// Internally, CreateEvent will call Message.MsgEncode to encode given message to
	// bytes and saves that to internal variable
	CreateEvent(eventOpts EventOptions) (Event, error)
	Wait() error
	Close() error
}
