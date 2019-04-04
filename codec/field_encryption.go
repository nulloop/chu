package codec

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"reflect"

	"github.com/nulloop/chu"
)

const (
	tagFieldTarget = "personal"
)

var (
	ErrNotPointer          = errors.New("not pass as pointer")
	ErrMultiplePersonalIDs = errors.New("multiple personal ids detected")
	ErrPersonalIDNotString = errors.New("personal id must be string")
	ErrPersonalIDEmpty     = errors.New("personal id is empty")
)

var _ chu.Codec = &FieldsEncryption{}

// KeyFinder is an interface which abstract how key can be access
// for encrypt and decrypt
type KeyFinder interface {
	// LookupKey is searching for key and return the key. Usually it depends on Storage
	// to retrive the key correspond with given id.
	LookupKey(id string) (string, error)
}

type FieldsEncryption struct {
	finder KeyFinder
}

func (fe *FieldsEncryption) Encode(ptrMsg interface{}) error {
	return fe.encrypt(ptrMsg, "")
}

func (fe *FieldsEncryption) Decode(ptrMsg interface{}) error {
	return fe.decrypt(ptrMsg, "")
}

func (fe *FieldsEncryption) encrypt(ptr interface{}, givenKey string) error {
	val := reflect.ValueOf(ptr)
	if val.IsNil() {
		return nil
	}

	if val.Kind() != reflect.Ptr {
		return ErrNotPointer
	}

	elm := val.Elem()

	id, indexes, err := extract(elm)
	if err != nil {
		return err
	}

	if givenKey == "" {
		givenKey, err = fe.finder.LookupKey(id)
		if err != nil {
			return err
		}
	}

	for _, idx := range indexes {
		field := elm.Field(idx)

		switch field.Kind() {
		case reflect.String:
			plainAsBytes := []byte(field.Interface().(string))
			encryptedAsBytes, err := encrypt(plainAsBytes, givenKey)
			if err != nil {
				return err
			}
			field.SetString(base64.StdEncoding.EncodeToString(encryptedAsBytes))
		case reflect.Slice:
			for i := 0; i < field.Len(); i++ {
				itemPtr := field.Index(i).Interface()
				err := fe.encrypt(itemPtr, givenKey)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (fe *FieldsEncryption) decrypt(ptr interface{}, givenKey string) error {
	val := reflect.ValueOf(ptr)
	if val.IsNil() {
		return nil
	}

	if val.Kind() != reflect.Ptr {
		return ErrNotPointer
	}

	elm := val.Elem()

	id, indexes, err := extract(elm)
	if err != nil {
		return err
	}

	if givenKey == "" {
		givenKey, err = fe.finder.LookupKey(id)
		if err != nil {
			return err
		}
	}

	for _, idx := range indexes {
		field := elm.Field(idx)
		switch field.Kind() {
		case reflect.String:
			inputAsBase64 := field.Interface().(string)
			encryptedAsBytes, err := base64.StdEncoding.DecodeString(inputAsBase64)
			if err != nil {
				return err
			}

			decryptedAsBytes, err := decrypt(encryptedAsBytes, givenKey)
			if err != nil {
				err = nil
				elm.Field(idx).SetString("")
				continue
			}

			elm.Field(idx).SetString(string(decryptedAsBytes))
		case reflect.Slice:
			for i := 0; i < field.Len(); i++ {
				itemPtr := field.Index(i).Interface()
				err := fe.decrypt(itemPtr, givenKey)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func NewFieldsEncryption(finder KeyFinder) *FieldsEncryption {
	return &FieldsEncryption{
		finder: finder,
	}
}

func extract(elm reflect.Value) (id string, indexes []int, err error) {
	ok := false
	indexes = make([]int, 0)

	for i := 0; i < elm.NumField(); i++ {
		tag := elm.Type().Field(i).Tag.Get(tagFieldTarget)
		if tag == "" {
			continue
		}

		if tag == "data" {
			indexes = append(indexes, i)
		} else if tag == "id" {
			if id != "" {
				return "", nil, ErrMultiplePersonalIDs
			}

			id, ok = elm.Field(i).Interface().(string)
			if !ok {
				return "", nil, ErrPersonalIDNotString
			}
			if id == "" {
				return "", nil, ErrPersonalIDEmpty
			}
		}
	}

	return
}

// Hash converts given string into sha256 hash bytes
func hash(key string) []byte {
	data := sha256.Sum256([]byte(key))
	return data[0:]
}

// encrypt converts plain data  encrypt it uses AES and passphrase
func encrypt(data []byte, passphrase string) ([]byte, error) {
	block, _ := aes.NewCipher(hash(passphrase))
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// decrypt converted encrypted data to plain text using passphrase and AES protocol
func decrypt(data []byte, passphrase string) ([]byte, error) {
	key := hash(passphrase)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
