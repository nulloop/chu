package nats

import (
	"errors"

	"github.com/nulloop/chu"
)

type NatsSender struct {
	provier *NatsProvider
	opts    *NatsOptions
}

var _ chu.Sender = &NatsSender{}

func (s *NatsSender) Send(msg chu.Message, middlewares ...func(chu.Handler) chu.Handler) error {
	natsMsg, ok := msg.(*NatsMessage)
	if !ok {
		return errors.New("message is not NatsMessage")
	}

	var handler chu.Handler
	for _, middlewares := range middlewares {
		handler = middlewares(handler)
	}

	if handler != nil {
		err := handler.ServeMessage(natsMsg)
		if err != nil {
			return err
		}
	}

	data, err := natsMsg.encode()
	if err != nil {
		return err
	}

	return s.provier.conn.Publish(msg.Subject(), data)
}
