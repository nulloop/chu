package nats_test

import (
	"context"
	"sync"
	"testing"

	"github.com/nulloop/choo"

	gonats "github.com/nats-io/go-nats"
	"github.com/nats-io/nats-streaming-server/server"

	"github.com/nulloop/choo/nats"
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

	wg.Add(1)

	r.Route("root", func(r choo.Receiver) {
		// r.Use(func(h choo.Handler) choo.Handler {
		// 	return choo.HandlerFunc(func(msg choo.Message) error {
		// 		fmt.Println("got the middleware", msg.ID())
		// 		return nil
		// 	})
		// })

		r.Handle("test", func(msg choo.Message) error {
			defer wg.Done()
			if string(msg.Body()) != "Hello World!" {
				t.Fatal("wrong message")
			}
			return nil
		})
	})

	s := provider.Sender()
	err = s.Send(nats.NewMessage(context.Background(), "1", "root.test", []byte("Hello World!")))
	if err != nil {
		t.Error(err)
	}

	wg.Wait()
}
