package choo

import (
	"context"
)

type Message interface {
	ID() string
	Subject() string
	Body() []byte
	Sequence() uint64
	Timestamp() int64
	Context() context.Context
	WithContext(context.Context) Message
}

type Handler interface {
	ServeMessage(msg Message) error
}

type HandlerFunc func(msg Message) error

func (f HandlerFunc) ServeMessage(msg Message) error {
	return f(msg)
}

type Sender interface {
	Send(msg Message) error
}

type Receiver interface {
	Use(middlewares ...func(Handler) Handler)
	Route(path string, fn func(Receiver)) Receiver
	// Handle is similar to Fanout. Everyone will receive the copy of message
	// This is ideal for replication of events that needs to be stored on multiple instances of
	// same service
	Handle(subject string, h HandlerFunc)
	// HandlerQueue is used for LoadBalancing the work
	// This is ideal for distribute the load on multiple instances of same service
	HandleQueue(subject string, queueName string, h HandlerFunc)
}

type Provider interface {
	Sender() Sender
	Receiver() Receiver
	Close() error
}
