package chu

import (
	"context"
)

// AggregateIDGen is a function which generate aggregate id for each new message
// this function needs to be set based on desire state before the program sends or receives messages
var AggregateIDGen func() string

type Message interface {
	ID() string
	AggregateID() string
	Subject() string
	Body() []byte
	Sequence() uint64
	Timestamp() int64
	Context() context.Context
	WithContext(context.Context) Message
	// Extend accepts id, subject, and body and preserve aggregate
	// make sure AggregateIDGen is set
	Extend(string, string, []byte) Message
}

type Handler interface {
	ServeMessage(msg Message) error
}

type HandlerFunc func(msg Message) error

func (f HandlerFunc) ServeMessage(msg Message) error {
	return f(msg)
}

type Sender interface {
	Send(msg Message, middlewares ...func(Handler) Handler) error
}

type Receiver interface {
	Use(middlewares ...func(Handler) Handler)
	Route(path string, fn func(Receiver)) Receiver
	// Group can be use to apply midddleware to only group of handlers
	Group(fn func(Receiver)) Receiver
	// Handle is similar to Fanout. Everyone will receive the copy of message
	// This is ideal for replication of events that needs to be stored on multiple instances of
	// same service
	Handle(subject string, h HandlerFunc)
	// HandlerQueue is used for LoadBalancing the work
	// This is ideal for distribute the load on multiple instances of same service
	HandleQueue(subject string, h HandlerFunc)
}

type Provider interface {
	Sender() Sender
	Receiver() Receiver
	Close() error
}
