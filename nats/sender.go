package nats

import (
	"github.com/nulloop/choo"
)

type NatsSender struct {
	provier *NatsProvider
	opts    *NatsOptions
}

var _ choo.Sender = &NatsSender{}

func (s *NatsSender) Send(msg choo.Message) error {
	return nil
}
