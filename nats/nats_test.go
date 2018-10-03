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

const (
	clusterName = "dummy_server"
	subject     = "root.a.b.c.test2"
)

func TestMain(m *testing.M) {
	natsStreamServer, err := server.RunServer(clusterName)
	if err != nil {
		panic(err)
	}

	defer natsStreamServer.Shutdown()
	exitCode := m.Run()

	fmt.Printf("exit code: %d\n", exitCode)
}

func TestCreateNewProvider(t *testing.T) {
	provider, err := nats.NewProvider(nats.OptClientAddr("client", clusterName, gonats.DefaultURL))
	if err != nil {
		t.Fatal(err)
	}

	provider.Close()
}

func TestCreateSender(t *testing.T) {
	provider, err := nats.NewProvider(nats.OptClientAddr("client", clusterName, gonats.DefaultURL))
	if err != nil {
		t.Fatal(err)
	}

	defer provider.Close()

	sender := provider.Sender()
	sender.Send(nil)
}

func TestSendNilMessage(t *testing.T) {
	provider, err := nats.NewProvider(nats.OptClientAddr("client", clusterName, gonats.DefaultURL))
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

	provider, err := nats.NewProvider(nats.OptClientAddr("client", clusterName, gonats.DefaultURL))
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

// create server
// send 3 messages
// create client and connect to server with QueueHandle with sequence 0
// disconnect client
// reconnect the same client with sequence number 3
// it should not receive any message
func TestSenario1(t *testing.T) {

	subject := "root.a.b.c.d.e"

	client1, err := nats.NewProvider(
		nats.OptClientAddr("client", clusterName, gonats.DefaultURL),
		nats.OptGenDurableName(func(path string) string {
			return fmt.Sprintf("durable1.%s", path)
		}),
		nats.OptGenQueueName(func(path string) string {
			return fmt.Sprintf("queue1.%s", path)
		}),
	)
	if err != nil {
		t.Fatal(err)
	}

	// r := client1.Receiver()
	// r.Handle(subject, func(msg chu.Message) error {
	// 	fmt.Println("got message")
	// 	return nil
	// })

	// send 3 messages

	sender := client1.Sender()

	for i := 0; i < 3; i++ {
		id := fmt.Sprintf("%d", i)
		aggregateID := fmt.Sprintf("a:%d", i)
		body := fmt.Sprintf("Hello Message %d", i)
		err := sender.Send(nats.NewMessage(context.Background(), id, aggregateID, subject, []byte(body)))
		if err != nil {
			t.Error(err)
		}
		fmt.Println("sending message")
	}

	err = client1.Close()
	if err != nil {
		t.Fatal(err)
	}

	client2, err := nats.NewProvider(
		nats.OptClientAddr("client2", clusterName, gonats.DefaultURL),
		nats.OptGenDurableName(func(path string) string {
			return fmt.Sprintf("durable2.%s", path)
		}),
		nats.OptGenQueueName(func(path string) string {
			return fmt.Sprintf("queue2.%s", path)
		}),
		nats.OptUpdateSequence(func(path string, sequence uint64) {

		}),
		nats.OptGetSequence(func(path string) uint64 {
			return 1
		}),
	)
	if err != nil {
		t.Fatal(err)
	}

	defer client2.Close()

	// create client and connect to server with QueueHandle with sequence 0
	var wg sync.WaitGroup
	wg.Add(3)
	r := client2.Receiver()
	r.HandleQueue(subject, func(msg chu.Message) error {
		defer wg.Done()

		fmt.Println("got a message for id: ", msg.ID())

		return nil
	})

	fmt.Println("waiting...")

	wg.Wait()

}
