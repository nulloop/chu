package nats

import (
	"context"
	"errors"
	"fmt"
	"strings"

	stan "github.com/nats-io/go-nats-streaming"
	"github.com/nulloop/chu"
)

var (
	ErrPathRequired         = errors.New("path can't be empty string.")
	ErrPathStartedWithPoint = errors.New("path should not start with '.'.")
	ErrPathEndedWithPoint   = errors.New("path should not end with '.'.")
)

type NatsReceiver struct {
	provier     *NatsProvider
	path        string
	opts        *NatsOptions
	middlewares []func(chu.Handler) chu.Handler
}

var _ chu.Receiver = &NatsReceiver{}

func checkPath(path string) error {
	if path == "" {
		return ErrPathRequired
	}

	if strings.Index(path, ".") == 0 {
		return ErrPathStartedWithPoint
	}

	if strings.LastIndex(path, ".") == len(path)-1 {
		return ErrPathEndedWithPoint
	}

	return nil
}

func mergePath(base, path string) string {
	err := checkPath(path)
	if err != nil {
		panic(err)
	}

	if base != "" {
		path = fmt.Sprintf("%s.%s", base, path)
	}

	return path
}

func (n *NatsReceiver) Use(middlewares ...func(chu.Handler) chu.Handler) {
	n.middlewares = append(n.middlewares, middlewares...)
}

func (n *NatsReceiver) Route(path string, fn func(chu.Receiver)) chu.Receiver {
	path = mergePath(n.path, path)

	// need to copy middle ware from parent
	// middlewares should be stateless
	middlewares := make([]func(chu.Handler) chu.Handler, 0)
	middlewares = append(middlewares, n.middlewares...)

	receiver := &NatsReceiver{
		path:        path,
		provier:     n.provier,
		opts:        n.opts,
		middlewares: middlewares,
	}

	fn(receiver)

	return receiver
}

func (n *NatsReceiver) Group(fn func(chu.Receiver)) chu.Receiver {
	// need to copy middle ware from parent
	// middlewares should be stateless
	middlewares := make([]func(chu.Handler) chu.Handler, 0)
	middlewares = append(middlewares, n.middlewares...)

	receiver := &NatsReceiver{
		path:        n.path,
		provier:     n.provier,
		opts:        n.opts,
		middlewares: middlewares,
	}

	fn(receiver)

	return receiver
}

func (n *NatsReceiver) Handle(subject string, h chu.HandlerFunc) {
	path := mergePath(n.path, subject)

	options := []stan.SubscriptionOption{
		stan.SetManualAckMode(),
	}

	if n.opts.getSequence != nil {
		options = append(options, stan.StartAtSequence(n.opts.getSequence(path)))
	}

	if n.opts.genDurableName != nil {
		durableName := n.opts.genDurableName(path)
		if durableName != "" {
			stan.DurableName(path)
		}
	}

	// Here's the final handler is built up based on
	// layers of layers of middleware
	var handler chu.Handler = h
	for _, middleware := range n.middlewares {
		handler = middleware(handler)
	}

	n.provier.conn.Subscribe(path, func(msg *stan.Msg) {
		message := &NatsMessage{}

		// this will decode id and message as bytes
		err := message.decode(msg.Data)
		if err != nil {
			panic(err)
		}

		message.sequence = msg.Sequence
		message.subject = msg.Subject
		message.ctx = context.Background()
		message.timestamp = msg.Timestamp

		err = handler.ServeMessage(message)

		if err == nil {
			err = msg.Ack()
		}

		if err != nil && n.opts.logError != nil {
			n.opts.logError(err)
		}

		if err == nil {
			if n.opts.updateSequence != nil {
				n.opts.updateSequence(path, msg.Sequence)
			}
		}

	}, options...)
}

func (n *NatsReceiver) HandleQueue(subject string, h chu.HandlerFunc) {
	path := mergePath(n.path, subject)

	options := []stan.SubscriptionOption{
		stan.SetManualAckMode(),
	}

	if n.opts.getSequence != nil {
		options = append(options, stan.StartAtSequence(n.opts.getSequence(path)))
	}

	if n.opts.genDurableName != nil {
		durableName := n.opts.genDurableName(path)
		if durableName != "" {
			stan.DurableName(path)
		}
	}

	if n.opts.genQueueName == nil {
		panic("genQueueName options must be set")
	}

	queueName := n.opts.genQueueName(path)

	if queueName == "" {
		panic("queue name is empy")
	}

	// Here's the final handler is built up based on
	// layers of layers of middleware
	var handler chu.Handler = h
	for _, middleware := range n.middlewares {
		handler = middleware(handler)
	}

	n.provier.conn.QueueSubscribe(path, queueName, func(msg *stan.Msg) {
		message := &NatsMessage{}

		// this will decode id and message as bytes
		err := message.decode(msg.Data)
		if err != nil {
			panic(err)
		}

		message.sequence = msg.Sequence
		message.subject = msg.Subject
		message.ctx = context.Background()
		message.timestamp = msg.Timestamp

		err = handler.ServeMessage(message)
		if err == nil {
			err = msg.Ack()
		}

		if err != nil && n.opts.logError != nil {
			n.opts.logError(err)
		}

		if err == nil {
			if n.opts.updateSequence != nil {
				n.opts.updateSequence(path, msg.Sequence)
			}
		}

	}, options...)
}
