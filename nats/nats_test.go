package nats_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	gonats "github.com/nats-io/go-nats"
	"github.com/nats-io/nats-streaming-server/server"

	"github.com/nulloop/chu"
	"github.com/nulloop/chu/nats"
)

func runDummyServer(clusterName string) (*server.StanServer, error) {
	s, err := server.RunServer(clusterName)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func TestRunDummyServer(t *testing.T) {
	server, err := runDummyServer("dummy_server")
	if err != nil {
		t.Error(err)
	}
	server.Shutdown()
}

func TestCreateNewProvider(t *testing.T) {
	server, err := runDummyServer("dummy_server")
	if err != nil {
		t.Error(err)
	}
	defer server.Shutdown()

	provider, err := nats.NewProvider(nats.OptClientAddr("client", "dummy_server", gonats.DefaultURL))
	if err != nil {
		t.Fatal(err)
	}

	provider.Close()
}

func TestCreateSender(t *testing.T) {
	server, err := runDummyServer("dummy_server")
	if err != nil {
		t.Error(err)
	}
	defer server.Shutdown()

	provider, err := nats.NewProvider(nats.OptClientAddr("client", "dummy_server", gonats.DefaultURL))
	if err != nil {
		t.Fatal(err)
	}

	defer provider.Close()

	sender := provider.Sender()
	sender.Send(nil)
}

func TestSendNilMessage(t *testing.T) {
	server, err := runDummyServer("dummy_server")
	if err != nil {
		t.Error(err)
	}
	defer server.Shutdown()

	provider, err := nats.NewProvider(nats.OptClientAddr("client", "dummy_server", gonats.DefaultURL))
	if err != nil {
		t.Fatal(err)
	}

	defer provider.Close()

	sender := provider.Sender()

	err = sender.Send(nil)
	if err == nil {
		t.Error("should error out")
	}

	if err.Error() != "message is not NatsMessage" {
		t.Error("should be 'message is not NatsMessage' error")
	}
}

func TestRoute(t *testing.T) {
	var wg sync.WaitGroup

	server, err := runDummyServer("dummy_server")
	if err != nil {
		t.Error(err)
	}
	defer server.Shutdown()

	provider, err := nats.NewProvider(nats.OptClientAddr("client", "dummy_server", gonats.DefaultURL))
	if err != nil {
		t.Fatal(err)
	}

	defer provider.Close()

	r := provider.Receiver()

	wg.Add(2)

	r.Route("root.a.b.c", func(r chu.Receiver) {
		r.Use(func(h chu.Handler) chu.Handler {
			return chu.HandlerFunc(func(msg chu.Message) error {
				fmt.Println("got the middleware", msg.ID())
				return h.ServeMessage(msg)
			})
		})

		r.Handle("test2", func(msg chu.Message) error {
			defer wg.Done()
			return nil
		})

		r.Handle("test", func(msg chu.Message) error {
			defer wg.Done()
			if string(msg.Body()) != "Hello World!" {
				t.Fatal("wrong message")
			}
			return nil
		})
	})

	s := provider.Sender()
	err = s.Send(nats.NewMessage(context.Background(), "1", "a:1", "root.a.b.c.test", []byte("Hello World!")))
	if err != nil {
		t.Error(err)
	}

	err = s.Send(nats.NewMessage(context.Background(), "2", "a:2", "root.a.b.c.test2", []byte("Hello World!")))
	if err != nil {
		t.Error(err)
	}

	wg.Wait()
}
