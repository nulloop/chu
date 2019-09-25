package codec_test

import (
	"testing"

	"github.com/nulloop/chu/v2/binary"
	"github.com/nulloop/chu/v2/codec"
)

var _ codec.KeyFinder = &MemoryKeyFinder{}

type MemoryKeyFinder struct {
	keys map[string]string
}

func (m *MemoryKeyFinder) LookupKey(id string) (string, error) {
	key, ok := m.keys[id]
	if !ok {
		key = "1234"
		m.keys[id] = key
	}

	return key, nil
}

func NewMemoryKeyFinder() *MemoryKeyFinder {
	return &MemoryKeyFinder{
		keys: make(map[string]string),
	}
}

type User struct {
	ID        string `conceal:"id"`
	FirstName string
	Email     string `conceal:"data"`
}

func (u *User) Encode() ([]byte, error) {
	return binary.DefaultEncode(u)
}

func (u *User) Decode(data []byte) error {
	return binary.DefaultDecode(data, u)
}

func TestBasicEncryption(t *testing.T) {
	keyFinder := NewMemoryKeyFinder()

	fieldEncryption := codec.NewFieldsEncryption(keyFinder)

	user := User{
		ID:        "1",
		FirstName: "Ali",
		Email:     "ali@nulloop.com",
	}

	err := fieldEncryption.Encode(&user)
	if err != nil {
		t.Fatal(err)
	}

	if user.ID != "1" {
		t.Fatalf("user.id must be 1 but got %s", user.ID)
	}

	if user.FirstName != "Ali" {
		t.Fatalf("user.FirstName must be Ali but got %s", user.FirstName)
	}

	if user.Email == "ali@nulloop.com" {
		t.Fatalf("user.Email must be encrypted but got %s", user.Email)
	}

	err = fieldEncryption.Decode(&user)
	if err != nil {
		t.Fatal(err)
	}

	if user.ID != "1" {
		t.Fatalf("user.id must be 1 but got %s", user.ID)
	}

	if user.FirstName != "Ali" {
		t.Fatalf("user.FirstName must be Ali but got %s", user.FirstName)
	}

	if user.Email != "ali@nulloop.com" {
		t.Fatalf("user.Email must be decrypted but got %s", user.Email)
	}
}

func TestKeyRemovedEncryption(t *testing.T) {
	keyFinder := NewMemoryKeyFinder()

	fieldEncryption := codec.NewFieldsEncryption(keyFinder)

	user := User{
		ID:        "1",
		FirstName: "Ali",
		Email:     "ali@nulloop.com",
	}

	err := fieldEncryption.Encode(&user)
	if err != nil {
		t.Fatal(err)
	}

	if user.ID != "1" {
		t.Fatalf("user.id must be 1 but got %s", user.ID)
	}

	if user.FirstName != "Ali" {
		t.Fatalf("user.FirstName must be Ali but got %s", user.FirstName)
	}

	if user.Email == "ali@nulloop.com" {
		t.Fatalf("user.Email must be encrypted but got %s", user.Email)
	}

	// remove key for id 1
	keyFinder.keys["1"] = ""

	err = fieldEncryption.Decode(&user)
	if err != nil {
		t.Fatal(err)
	}

	if user.ID != "1" {
		t.Fatalf("user.id must be 1 but got %s", user.ID)
	}

	if user.FirstName != "Ali" {
		t.Fatalf("user.FirstName must be Ali but got %s", user.FirstName)
	}

	if user.Email != "" {
		t.Fatalf("user.Email must be empty string but got %s", user.Email)
	}
}

func TestMultiLevelEncryption(t *testing.T) {

	type Class struct {
		Name string `conceal:"data"`
	}

	type User struct {
		ID      string   `conceal:"id"`
		Name    string   `conceal:"data"`
		Classes []*Class `conceal:"data"`
	}

	user := &User{
		ID:   "1",
		Name: "John",
		Classes: []*Class{
			{Name: "Class A"},
			{Name: "Class B"},
		},
	}

	keyFinder := NewMemoryKeyFinder()

	fieldEncryption := codec.NewFieldsEncryption(keyFinder)

	err := fieldEncryption.Encode(user)
	if err != nil {
		t.Fatal(err)
	}

	if user.Name == "John" {
		t.Fatal("expect Name field to be encrypted")
	}

	err = fieldEncryption.Decode(user)
	if err != nil {
		t.Fatal(err)
	}
}
