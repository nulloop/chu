package broker_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	gonats "github.com/nats-io/go-nats"
	"github.com/nats-io/nats-streaming-server/server"
	"github.com/rs/xid"

	"github.com/nulloop/chu"
	"github.com/nulloop/chu/binary"
	"github.com/nulloop/chu/broker"
)

const (
	clusterName = "dummy_server"
	subject     = "root.a.b.c.test2"
)

func TestMain(m *testing.M) {

	chu.GenID = func() string {
		return xid.New().String()
	}

	natsStreamServer, err := server.RunServer(clusterName)
	if err != nil {
		panic(err)
	}

	defer natsStreamServer.Shutdown()
	exitCode := m.Run()

	fmt.Printf("exit code: %d\n", exitCode)
}

type dummySub struct {
	msg chan string
	err chan error
}

func (d *dummySub) Topic() string {
	return "a.b.c"
}
func (d *dummySub) Durable() bool {
	return true
}
func (d *dummySub) Group() string {
	return ""
}
func (d *dummySub) HandleEvent(event chu.ReceivedEvent) bool {
	msg := &message{}

	err := event.Message(msg)
	if err != nil {
		d.err <- err
	}

	d.msg <- fmt.Sprint(event.ID(), event.AggregateID(), msg.Message)

	return true
}

type message struct {
	Message string
}

func (m *message) MsgEncode() ([]byte, error) {
	return binary.DefaultEncode(m)
}

func (m *message) MsgDecode(data []byte) error {
	return binary.DefaultDecode(data, m)
}

func TestBroker(t *testing.T) {
	nats, err := broker.NewNats(&broker.NatsOptions{
		Addr:      gonats.DefaultURL,
		ClusterID: clusterName,
		ClientID:  "foo",
	})
	if err != nil {
		t.Fatal(err)
	}

	defer nats.Close()

	msgChan := make(chan string, 1)
	errChan := make(chan error, 1)

	subscription, err := nats.Subscribe(&dummySub{
		err: errChan,
		msg: msgChan,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer subscription.Unsubscribe()

	event, err := nats.CreateEvent(chu.EventOptions{
		Topic: "a.b.c",
		Message: &message{
			Message: "Hello World",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	nats.Wait()

	err = nats.Publish(event)
	if err != nil {
		t.Fatal(err)
	}

	select {
	case msg := <-msgChan:
		if strings.Index(msg, "Hello World") == -1 {
			t.Fatalf("expected %s but got %s", "12341234Hello World", msg)
		}
	case err := <-errChan:
		t.Fatal(err)
	case <-time.After(5 * time.Second):
		t.Fatal("got not message")
	}
}

type Sub struct {
}

func (d *Sub) Topic() string {
	return "a.b.c"
}
func (d *Sub) Durable() bool {
	return true
}
func (d *Sub) Group() string {
	return ""
}
func (d *Sub) HandleEvent(event chu.ReceivedEvent) bool {
	fmt.Println(event.ID())
	return true
}

func TestBroker2(t *testing.T) {
	client1, err := broker.NewNats(&broker.NatsOptions{
		Addr:      gonats.DefaultURL,
		ClusterID: clusterName,
		ClientID:  "client1",
	})
	if err != nil {
		t.Fatal(err)
	}

	defer client1.Close()

	client2, err := broker.NewNats(&broker.NatsOptions{
		Addr:      gonats.DefaultURL,
		ClusterID: clusterName,
		ClientID:  "client2",
	})
	if err != nil {
		t.Fatal(err)
	}

	defer client2.Close()

	client3, err := broker.NewNats(&broker.NatsOptions{
		Addr:      gonats.DefaultURL,
		ClusterID: clusterName,
		ClientID:  "client3",
	})
	if err != nil {
		t.Fatal(err)
	}

	defer client3.Close()

	sub1, err := client1.Subscribe(&Sub{})
	if err != nil {
		t.Fatal(err)
	}

	event1, err := client2.CreateEvent(chu.EventOptions{
		Topic: "a.b.c",
	})
	if err != nil {
		t.Fatal(err)
	}

	err = client2.Publish(event1)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(1 * time.Second)

	event2, err := client2.CreateEvent(chu.EventOptions{
		Topic: "a.b.c",
	})
	if err != nil {
		t.Fatal(err)
	}

	err = client2.Publish(event2)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(1 * time.Second)

	err = sub1.Close()
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(1 * time.Second)

	event3, err := client2.CreateEvent(chu.EventOptions{
		Topic: "a.b.c",
	})
	if err != nil {
		t.Fatal(err)
	}

	err = client2.Publish(event3)
	if err != nil {
		t.Fatal(err)
	}

	sub1, err = client1.Subscribe(&Sub{})
	if err != nil {
		t.Fatal(err)
	}

	defer sub1.Close()

	time.Sleep(1 * time.Second)

	sub1, err = client3.Subscribe(&Sub{})
	if err != nil {
		t.Fatal(err)
	}

	defer sub1.Close()

	time.Sleep(1 * time.Second)
}
