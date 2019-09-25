package codec

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"io"

	"github.com/alinz/conceal"

	"github.com/nulloop/chu/v2"
)

var _ chu.Codec = &FieldsEncryption{}
var _ conceal.Cipher = &FieldsEncryption{}

// KeyFinder is an interface which abstract how key can be access
// for encrypt and decrypt
type KeyFinder interface {
	// LookupKey is searching for key and return the key. Usually it depends on Storage
	// to retrive the key correspond with given id.
	LookupKey(id string) (string, error)
}

// FieldsEncryption is base codec that responsible
// for detecting and encrypting and decrypting fields before and after
// being publish and consume in message bus
type FieldsEncryption struct {
	finder KeyFinder
}

// Encode is a method that satisfy the chu's Codec interface
// and it's responsible is to call internal conceal's package to encrypt all
// targeted fields
func (fe *FieldsEncryption) Encode(ptrMsg interface{}) error {
	return conceal.Encrypt(ptrMsg, fe)
}

// Decode is a method that satisfy the chu's Codec interface
// and it's responsible is to call internal conceal's package to decrypt all
// targeted fields
func (fe *FieldsEncryption) Decode(ptrMsg interface{}) error {
	return conceal.Decrypt(ptrMsg, fe)
}

func (fe *FieldsEncryption) hash(key string) []byte {
	data := sha256.Sum256([]byte(key))
	return data[0:]
}

// Encrypt is method that satisfy the conceal's Cipher interface
// It gets the id and call LookupKey method to find a key for given id
// and it encrypt the value
func (fe *FieldsEncryption) Encrypt(value []byte, id string) ([]byte, error) {
	key, err := fe.finder.LookupKey(id)
	if err != nil {
		return nil, err
	}

	block, _ := aes.NewCipher(fe.hash(key))
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, value, nil)
	return ciphertext, nil
}

// Decrypt is method that satisfy the conceal's Cipher interface
// It gets the id and call LookupKey method to find a key for given id
// and it decrypt the value
func (fe *FieldsEncryption) Decrypt(value []byte, id string) ([]byte, error) {
	key, err := fe.finder.LookupKey(id)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(fe.hash(key))
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	nonce, ciphertext := value[:nonceSize], value[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// NewFieldsEncryption creates a FieldsEncryption that uses conceal package
// to encrypt/decrypt personal fields
func NewFieldsEncryption(finder KeyFinder) *FieldsEncryption {
	return &FieldsEncryption{
		finder: finder,
	}
}
