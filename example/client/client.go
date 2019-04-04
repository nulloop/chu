package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/rs/xid"

	"github.com/nulloop/chu"
	"github.com/nulloop/chu/broker"
)

type Security struct {
	cp   *x509.CertPool
	cert *tls.Certificate
}

func (s *Security) Certificate(crt, key string) error {
	cert, err := tls.LoadX509KeyPair(crt, key)
	if err != nil {
		return err
	}

	s.cert = &cert
	return nil
}

func (s *Security) CertificateAuthority(filename string) error {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	s.cp = x509.NewCertPool()
	if !s.cp.AppendCertsFromPEM(b) {
		return fmt.Errorf("credentials: failed to append certificates")
	}

	return nil
}

func (s *Security) ClientTLS(serverName string) *tls.Config {
	tlsConfig := tls.Config{RootCAs: s.cp}
	if serverName != "" {
		tlsConfig.ServerName = serverName
	}

	if s.cert != nil {
		tlsConfig.Certificates = []tls.Certificate{*s.cert}
	}
	return &tlsConfig
}

type Sub struct{}

func (s *Sub) Topic() string {
	return "a.b.c"
}

func (s *Sub) Durable() bool {
	return true
}

func (s *Sub) Group() string {
	return "123"
}

func (s *Sub) HandleEvent(event chu.ReceivedEvent) bool {
	fmt.Printf("got Event: %s, %s\n", event.ID(), event.AggregateID())

	return true
}

func main() {
	chu.GenID = func() string {
		return xid.New().String()
	}

	security := &Security{}

	err := security.Certificate("./etc/service.crt", "./etc/service.key")
	if err != nil {
		panic(err)
	}

	err = security.CertificateAuthority("./etc/ca.crt")
	if err != nil {
		panic(err)
	}

	broker, err := broker.NewNats(&broker.NatsOptions{
		Addr:      "nats://127.0.0.1:4222",
		ClientID:  "client5",
		ClusterID: "sample",
		TLS:       security.ClientTLS(""),
	})
	if err != nil {
		panic(err)
	}

	defer broker.Close()

	// subscription, err := broker.Subscribe(&Sub{})
	// if err != nil {
	// 	panic(err)
	// }

	// defer subscription.Close()

	for i := 0; i < 100; i++ {
		event, err := broker.CreateEvent(chu.EventOptions{
			Topic: "a.b.c",
		})
		if err != nil {
			panic(err)
		}

		err = broker.Publish(event)
		if err != nil {
			panic(err)
		}
	}

	time.Sleep(1 * time.Second)
}
