package nats

import (
	"errors"

	"github.com/nulloop/chu"
)

var (
	ErrNotNatsMessage = errors.New("message is not NatsMessage")
)

type NatsSender struct {
	provier     *NatsProvider
	opts        *NatsOptions
	middlewares chu.Handler
}

var _ chu.Sender = &NatsSender{}

func (s *NatsSender) Send(msg chu.Message) error {
	natsMsg, ok := msg.(*NatsMessage)
	if !ok {
		return ErrNotNatsMessage
	}

	if s.middlewares != nil {
		err := s.middlewares.ServeMessage(natsMsg)
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
