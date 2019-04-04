package binary

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"time"
)

type SimpleBinary struct {
	cap    int
	idx    int
	buffer []byte
}

func (s *SimpleBinary) Reset() {
	s.idx = 0
}

func (s *SimpleBinary) EncodeBytes(val []byte) error {
	l := len(val)
	if (s.idx + l) > s.cap {
		return fmt.Errorf("buffer is small")
	}

	err := s.EncodeUint64(uint64(l))
	if err != nil {
		return err
	}

	copy(s.buffer[s.idx:s.idx+l], val)
	s.idx += l

	return nil
}

func (s *SimpleBinary) EncodeUint64(val uint64) error {
	if (s.idx + 8) > s.cap {
		return fmt.Errorf("buffer is too small")
	}

	binary.PutUvarint(s.buffer[s.idx:s.idx+8], val)

	s.idx += 8
	return nil
}

func (s *SimpleBinary) EncodeString(val string) error {
	l := len(val)
	if (s.idx + l) > s.cap {
		return fmt.Errorf("buffer is small")
	}

	err := s.EncodeUint64(uint64(l))
	if err != nil {
		return err
	}

	copy(s.buffer[s.idx:s.idx+l], val)
	s.idx += l

	return nil
}

func (s *SimpleBinary) EncodeTime(val time.Time) error {
	encoded := val.Format(time.RFC3339Nano)
	return s.EncodeString(encoded)
}

func (s *SimpleBinary) DecodeBytes() ([]byte, error) {
	l, err := s.DecodeUint64()
	if err != nil {
		return nil, err
	}

	b := s.buffer[s.idx : s.idx+int(l)]
	s.idx += int(l)
	return b, nil
}

func (s *SimpleBinary) DecodeUint64() (uint64, error) {
	val, n := binary.Uvarint(s.buffer[s.idx : s.idx+8])
	if n <= 0 {
		return 0, fmt.Errorf("can't decode uint64. code: %d", n)
	}

	s.idx += 8
	return val, nil
}

func (s *SimpleBinary) DecodeString() (string, error) {
	l, err := s.DecodeUint64()
	if err != nil {
		return "", err
	}

	val := s.buffer[s.idx : s.idx+int(l)]
	s.idx += int(l)

	return string(val), nil
}

func (s *SimpleBinary) DecodeTime() (time.Time, error) {
	var t time.Time
	encoded, err := s.DecodeString()
	if err != nil {
		return t, err
	}

	return time.Parse(time.RFC3339Nano, encoded)
}

func (s *SimpleBinary) Bytes() []byte {
	return s.buffer[0:s.idx]
}

func NewEncoding(size int) *SimpleBinary {
	return &SimpleBinary{
		buffer: make([]byte, size),
		cap:    size,
		idx:    0,
	}
}

func NewDecoding(data []byte) *SimpleBinary {
	return &SimpleBinary{
		buffer: data,
		cap:    len(data),
		idx:    0,
	}
}

func DefaultEncode(ptr interface{}) ([]byte, error) {
	var buffer bytes.Buffer
	err := gob.NewEncoder(&buffer).Encode(ptr)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func DefaultDecode(d []byte, ptr interface{}) error {
	buffer := bytes.NewBuffer(d)
	return gob.NewDecoder(buffer).Decode(ptr)
}
