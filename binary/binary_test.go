package binary_test

import (
	"testing"
	"time"

	"github.com/nulloop/chu/v2/binary"
)

func TestEncodingDecoding(t *testing.T) {

	enc := binary.NewEncoding(100)

	err := enc.EncodeUint64(100)
	if err != nil {
		t.Fatal(err)
	}

	err = enc.EncodeString("Hello World")
	if err != nil {
		t.Fatal(err)
	}

	currentTime := time.Now()

	err = enc.EncodeTime(currentTime)
	if err != nil {
		t.Fatal(err)
	}

	err = enc.EncodeBytes([]byte("BYTESSSSSSSSSSSSS"))
	if err != nil {
		t.Fatal(err)
	}

	encodedData := enc.Bytes()

	dec := binary.NewDecoding(encodedData)

	num, err := dec.DecodeUint64()
	if err != nil {
		t.Fatal(err)
	}

	if num != 100 {
		t.Fatalf("expected num to be 100 but got %d", num)
	}

	str, err := dec.DecodeString()
	if err != nil {
		t.Fatal(err)
	}

	if str != "Hello World" {
		t.Fatalf("expected str to be 'Hello World' but got %s", str)
	}

	decodedTime, err := dec.DecodeTime()
	if err != nil {
		t.Fatal(err)
	}

	if !decodedTime.Equal(currentTime) {
		t.Fatalf("expected to decode the time currently")
	}

	decodedBytes, err := dec.DecodeBytes()
	if err != nil {
		t.Fatal(err)
	}

	if "BYTESSSSSSSSSSSSS" != string(decodedBytes) {
		t.Fatalf("expected to decode bytes currently")
	}
}
