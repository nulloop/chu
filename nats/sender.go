package nats

import (
	"errors"

	"github.com/nulloop/choo"
)

type NatsSender struct {
	provier *NatsProvider
	opts    *NatsOptions
}

var _ choo.Sender = &NatsSender{}

func (s *NatsSender) Send(msg choo.Message, middlewares ...func(choo.Handler) choo.Handler) error {
	natsMsg, ok := msg.(*NatsMessage)
	if !ok {
		return errors.New("message is not NatsMessage")
	}

	var handler choo.Handler
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
