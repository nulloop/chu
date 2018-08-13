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

func (s *NatsSender) Send(msg choo.Message) error {
	natsMsg, ok := msg.(*NatsMessage)
	if !ok {
		return errors.New("message is not NatsMessage")
	}

	data, err := natsMsg.encode()
	if err != nil {
		return err
	}

	return s.provier.conn.Publish(msg.Subject(), data)
}
