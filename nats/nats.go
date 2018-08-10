package nats

import (
	"crypto/tls"
	"sync"
	"time"

	gonats "github.com/nats-io/go-nats"
	stan "github.com/nats-io/go-nats-streaming"
	"github.com/nulloop/choo"
)

type NatsOptions struct {
	tlsConfig      *tls.Config
	addr           string
	clusterID      string
	clientID       string
	genDurableName func(string) string
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

func OptGenDurableName(fn func(string) string) NatsOption {
	return func(opts *NatsOptions) error {
		opts.genDurableName = fn
		return nil
	}
}

func OptUpdateSequence(fn func(string, uint64)) NatsOption {
	return func(opts *NatsOptions) error {
		opts.updateSequence = fn
		return nil
	}
}

func OptGetSequence(fn func(string) uint64) NatsOption {
	return func(opts *NatsOptions) error {
		opts.getSequence = fn
		return nil
	}
}

func OptLogError(fn func(error)) NatsOption {
	return func(opts *NatsOptions) error {
		opts.logError = fn
		return nil
	}
}

// make sure that NatsProvider implements all the choo.Provider
var _ choo.Provider = &NatsProvider{}

type NatsProvider struct {
	conn             stan.Conn
	receiver         *NatsReceiver
	sender           *NatsSender
	receiverInitOnce sync.Once
	senderInitOnce   sync.Once
	opts             *NatsOptions
}

func (p *NatsProvider) Sender() choo.Sender {
	p.senderInitOnce.Do(func() {
		p.sender = &NatsSender{
			provier: p,
			opts:    p.opts,
		}
	})
	return p.sender
}

func (p *NatsProvider) Receiver() choo.Receiver {
	p.receiverInitOnce.Do(func() {
		p.receiver = &NatsReceiver{
			provier:     p,
			opts:        p.opts,
			middlewares: make([]func(choo.Handler) choo.Handler, 0),
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
