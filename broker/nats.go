package broker

import (
	"crypto/tls"
	"errors"
	"fmt"
	"time"

	gonats "github.com/nats-io/go-nats"
	stan "github.com/nats-io/go-nats-streaming"

	"github.com/nulloop/chu"
	"github.com/nulloop/chu/binary"
	"github.com/nulloop/chu/heartbeat"
)

var _ chu.Event = &NatsEvent{}
var _ chu.Broker = &Nats{}

type NatsEvent struct {
	id          string
	aggregateID string
	body        []byte
	topic       string
	createdAt   time.Time
	codec       []chu.Codec
}

func (evt *NatsEvent) ID() string           { return evt.id }
func (evt *NatsEvent) AggregateID() string  { return evt.aggregateID }
func (evt *NatsEvent) Topic() string        { return evt.topic }
func (evt *NatsEvent) CreatedAt() time.Time { return evt.createdAt }

func (evt *NatsEvent) Message(msg chu.Message) error {
	if evt.body == nil || len(evt.body) == 0 {
		return errors.New("body is empty")
	}

	err := msg.MsgDecode(evt.body)
	if err != nil {
		return err
	}

	for _, c := range evt.codec {
		err = c.Decode(msg)
		if err != nil {
			return err
		}
	}

	return nil
}

func (evt *NatsEvent) EvtEncode() ([]byte, error) {
	var err error

	// only need to serialize id, aggregate and byte
	size := len(evt.id) + 8
	size += len(evt.aggregateID) + 8
	size += len(evt.body) + 8

	bin := binary.NewEncoding(size)

	err = bin.EncodeString(evt.id)
	if err != nil {
		return nil, err
	}

	err = bin.EncodeString(evt.aggregateID)
	if err != nil {
		return nil, err
	}

	err = bin.EncodeBytes(evt.body)
	if err != nil {
		return nil, err
	}

	return bin.Bytes(), nil
}

func (evt *NatsEvent) EvtDecode(data []byte) error {
	var err error

	bin := binary.NewDecoding(data)

	evt.id, err = bin.DecodeString()
	if err != nil {
		return err
	}

	evt.aggregateID, err = bin.DecodeString()
	if err != nil {
		return err
	}

	evt.body, err = bin.DecodeBytes()
	if err != nil {
		return err
	}

	return nil
}

type Nats struct {
	name             string
	ackTimeout       time.Duration
	wait             func()
	tick             func()
	done             func() <-chan struct{}
	uniqueMsgChecker func(id string) bool
	conn             stan.Conn
	codec            []chu.Codec
}

func (n *Nats) Publish(event chu.Event) error {
	v, ok := event.(chu.EventEncoder)
	if !ok {
		return errors.New("event is not EventEncoder type")
	}

	data, err := v.EvtEncode()
	if err != nil {
		return err
	}

	return n.conn.Publish(event.Topic(), data)
}

func (n *Nats) durableName(topic string) string {
	return fmt.Sprintf("%s.%s", n.name, topic)
}

func (n *Nats) Subscribe(sub chu.Subscriber) (chu.Subscription, error) {
	var err error

	options := []stan.SubscriptionOption{
		stan.SetManualAckMode(),
		stan.DeliverAllAvailable(),
	}

	if sub.Durable() {
		options = append(options, stan.DurableName(n.durableName(sub.Topic())))
	}

	if n.ackTimeout > 0 {
		options = append(options, stan.AckWait(n.ackTimeout))
	}

	group := sub.Group()
	isGroupHandler := group != ""

	handler := func(msg *stan.Msg) {
		n.tick()

		// this `select` is a necessary logic to prevent calling
		// queue handler during warm-up time. Queue handler should not be called
		// as they are design to generate more events or talk to external services
		// generating events are prohabited during warm-up time.
		select {
		case <-n.done():
			// ignore
		default:
			// default will be called because we are still in
			// warmup time and we want to make sure that if handler is a group handler
			// it should not be executed.
			if isGroupHandler {
				msg.Ack()
				return
			}
		}

		event := &NatsEvent{
			codec: n.codec,
		}

		// extract id, aggregate id and bytes from message
		err := event.EvtDecode(msg.Data)
		if err != nil {
			// Log the error here
			msg.Ack()
			return
		}

		if !n.uniqueMsgChecker(event.id) {
			msg.Ack()
			return
		}

		event.topic = sub.Topic()
		event.createdAt = time.Unix(msg.Timestamp, 0)
		event.codec = n.codec

		if sub.HandleEvent(event) {
			msg.Ack()
		}
	}

	var subscription stan.Subscription

	if isGroupHandler {
		subscription, err = n.conn.QueueSubscribe(sub.Topic(), group, handler, options...)
	} else {
		subscription, err = n.conn.Subscribe(sub.Topic(), handler, options...)
	}

	if err != nil {
		return nil, err
	}

	return subscription, nil
}

func (n *Nats) CreateEvent(eventOpts chu.EventOptions) (chu.Event, error) {
	if eventOpts.Topic == "" {
		return nil, errors.New("topic is required")
	}

	id := chu.GenID()
	aggregateID := eventOpts.AggregateID

	if aggregateID == "" {
		aggregateID = chu.GenID()
	}

	var body []byte
	var err error

	if eventOpts.Message != nil {
		body, err = eventOpts.Message.MsgEncode()
		if err != nil {
			return nil, err
		}
	} else {
		body = make([]byte, 0)
	}

	return &NatsEvent{
		id:          id,
		aggregateID: aggregateID,
		body:        body,
		topic:       eventOpts.Topic,
		codec:       n.codec,
	}, nil
}

func (n *Nats) Wait() error {
	n.wait()
	return nil
}

func (n *Nats) Close() error {
	return n.conn.Close()
}

type NatsOptions struct {
	ClientID         string
	ClusterID        string
	Addr             string
	Codec            []chu.Codec
	TLS              *tls.Config
	AckTimeout       time.Duration
	WarmUpTimeout    time.Duration
	UniqueMsgChecker func(id string) bool // Enable Idempotence
}

func NewNats(opt *NatsOptions) (*Nats, error) {
	broker := &Nats{
		name:             fmt.Sprintf("%s.%s", opt.ClusterID, opt.ClientID),
		ackTimeout:       opt.AckTimeout,
		uniqueMsgChecker: opt.UniqueMsgChecker,
		codec:            opt.Codec,
	}

	if broker.uniqueMsgChecker == nil {
		broker.uniqueMsgChecker = func(_ string) bool { return true }
	}

	natsOpts := make([]gonats.Option, 0)
	if opt.TLS != nil {
		natsOpts = append(natsOpts, gonats.Secure(opt.TLS))
	}

	nc, err := gonats.Connect(opt.Addr, natsOpts...)
	if err != nil {
		return nil, err
	}

	for {
		broker.conn, err = stan.Connect(opt.ClusterID, opt.ClientID, stan.NatsConn(nc))
		if err != nil {
			if err == stan.ErrConnectReqTimeout {
				time.Sleep(1 * time.Second)
				continue
			}
			return nil, err
		}

		break
	}

	broker.wait, broker.tick, broker.done = heartbeat.New(opt.WarmUpTimeout)

	return broker, nil
}
