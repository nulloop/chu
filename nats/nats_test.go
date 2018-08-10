package nats_test

import (
	"testing"

	"github.com/nulloop/choo"
	"github.com/nulloop/choo/nats"
)

func TestConn(t *testing.T) {

	p, err := nats.NewProvider()
	if err != nil {
		t.Fatal(err)
	}

	r := p.Receiver()

	r.Use()

	r.Route("session", func(r choo.Receiver) {
		r.Use()

		r.Handle("created", func(msg choo.Message) error {
			return nil
		})

		r.Handle("updated", func(msg choo.Message) error {
			return nil
		})
	})

}
