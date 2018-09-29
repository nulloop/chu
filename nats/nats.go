package nats

import (
	"crypto/tls"
	"sync"
	"time"

	gonats "github.com/nats-io/go-nats"
	stan "github.com/nats-io/go-nats-streaming"
	"github.com/nulloop/chu"
)

type NatsOptions struct {
	tlsConfig      *tls.Config
	addr           string
	clusterID      string
	clientID       string
	genDurableName func(string) string
	genQueueName   func(string) string
	updateSequence func(string, uint64)
	getSequence    func(string) uint64
	logError       func(error)
}

type NatsOption func(*NatsOptions) error

func OptTLSConfig(tlsConfig *tls.Config) NatsOption {
	return func(opts *NatsOptions) error {
		opts.tlsConfig = tlsConfig
		return nil
	}
}

func OptClientAddr(clientID, clusterID, addr string) NatsOption {
	return func(opts *NatsOptions) error {
		opts.clientID = clientID
		opts.clusterID = clusterID
		opts.addr = addr
		return nil
	}
}

// OptGenDurableName is an optional that creates a proper durable name
// if you provide this method, every channel will be created with durable name based on path
func OptGenDurableName(fn func(string) string) NatsOption {
	return func(opts *NatsOptions) error {
		opts.genDurableName = fn
		return nil
	}
}

// OptGenQueueName is required if QueueHandle is being used, if not provided panic will happen to stop the program.
// the given function receives path and it needs to return an string represents quene name.
// make sure that this method is pure function and predictable
func OptGenQueueName(fn func(string) string) NatsOption {
	return func(opts *NatsOptions) error {
		opts.genQueueName = fn
		return nil
	}
}

// OptUpdateSequence is an optional and a helper function which will be called
// for every receive message. this function can be used to implement warm up feature.
// As an example, a key value store can be used to store the last sequence id for given path.
// and OptGetSequence option can be use to retrive it.
func OptUpdateSequence(fn func(string, uint64)) NatsOption {
	return func(opts *NatsOptions) error {
		opts.updateSequence = fn
		return nil
	}
}

// OptGetSequence is an optional and a helper function which complements `OptUpdateSequence`
// it will be called upon registering the route to nats stream and it starts with given sequence.
// if it returns 0, it will ask Nats to send every single messages.
func OptGetSequence(fn func(string) uint64) NatsOption {
	return func(opts *NatsOptions) error {
		opts.getSequence = fn
		return nil
	}
}

// OptLogError is an optional and will be called if an internal error happens
func OptLogError(fn func(error)) NatsOption {
	return func(opts *NatsOptions) error {
		opts.logError = fn
		return nil
	}
}

// make sure that NatsProvider implements all the chu.Provider
var _ chu.Provider = &NatsProvider{}

type NatsProvider struct {
	conn             stan.Conn
	receiver         *NatsReceiver
	receiverInitOnce sync.Once
	senderInitOnce   sync.Once
	opts             *NatsOptions
}

// Sender accepts list of middleware and creates a brand new Sender object
// Sender object is unique per middleware. It is a best practice to call this method once
// if you are not planning to change the middleware and assign it to somewhere so you can access it later
// the return object is also thread-safe as long as middlewares are pure stateless fucntions.
func (p *NatsProvider) Sender(middlewares ...func(chu.Handler) chu.Handler) chu.Sender {
	var handler chu.Handler
	for _, middlewares := range middlewares {
		handler = middlewares(handler)
	}

	return &NatsSender{
		provier:     p,
		opts:        p.opts,
		middlewares: handler,
	}
}

// Receiver always return the root Reciver. It only creates the root receiver once.
// if you want to create a chain of Receiver, use Receiver `Group` and `Route` methods
func (p *NatsProvider) Receiver() chu.Receiver {
	p.receiverInitOnce.Do(func() {
		p.receiver = &NatsReceiver{
			provier:     p,
			opts:        p.opts,
			middlewares: make([]func(chu.Handler) chu.Handler, 0),
		}
	})
	return p.receiver
}

func (p *NatsProvider) Close() error {
	return p.conn.Close()
}

func NewProvider(opts ...NatsOption) (*NatsProvider, error) {
	options := &NatsOptions{}
	for _, opt := range opts {
		if err := opt(options); err != nil {
			return nil, err
		}
	}

	natsOpts := make([]gonats.Option, 0)
	if options.tlsConfig != nil {
		natsOpts = append(natsOpts, gonats.Secure(options.tlsConfig))
	}

	nc, err := gonats.Connect(options.addr, natsOpts...)
	if err != nil {
		return nil, err
	}

	var conn stan.Conn

	for {
		conn, err = stan.Connect(options.clusterID, options.clientID, stan.NatsConn(nc))
		if err != nil {
			if err == stan.ErrConnectReqTimeout {
				time.Sleep(1 * time.Second)
				continue
			}
			return nil, err
		}

		break
	}

	return &NatsProvider{
		conn: conn,
		opts: options,
	}, nil
}
