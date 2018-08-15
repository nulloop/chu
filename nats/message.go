package nats

import (
	"bytes"
	"context"
	"encoding/gob"

	"github.com/nulloop/chu"
)

type meta struct {
	ID   string
	Body []byte
}

func (i *meta) encode() ([]byte, error) {
	buffer := &bytes.Buffer{}
	err := gob.NewEncoder(buffer).Encode(i)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (i *meta) decode(data []byte) error {
	buffer := bytes.NewBuffer(data)
	return gob.NewDecoder(buffer).Decode(i)
}

type NatsMessage struct {
	id        string
	subject   string
	body      []byte
	sequence  uint64
	timestamp int64
	ctx       context.Context
}

var _ chu.Message = &NatsMessage{}

func (m *NatsMessage) ID() string {
	return m.id
}

func (m *NatsMessage) Subject() string {
	return m.subject
}

func (m *NatsMessage) Body() []byte {
	return m.body
}

func (m *NatsMessage) Sequence() uint64 {
	return m.sequence
}

func (m *NatsMessage) Timestamp() int64 {
	return m.timestamp
}

func (m *NatsMessage) Context() context.Context {
	return m.ctx
}

func (m *NatsMessage) WithContext(ctx context.Context) chu.Message {
	return &NatsMessage{
		id:        m.id,
		subject:   m.subject,
		body:      m.body,
		sequence:  m.sequence,
		timestamp: m.timestamp,
		ctx:       ctx,
	}
}

func (m *NatsMessage) encode() ([]byte, error) {
	meta := &meta{
		ID:   m.id,
		Body: m.body,
	}
	return meta.encode()
}

func (m *NatsMessage) decode(data []byte) error {
	meta := &meta{}
	err := meta.decode(data)
	if err != nil {
		return err
	}

	m.id = meta.ID
	m.body = meta.Body

	return nil
}

func NewMessage(ctx context.Context, id string, subject string, body []byte) *NatsMessage {
	return &NatsMessage{
		id:      id,
		subject: subject,
		body:    body,
		ctx:     ctx,
	}
}
