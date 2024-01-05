package sealedbox

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"runtime"
)

const (
	KeySize = 32
	NonceSize = 12
)

var Magic = [...]byte { 0xce, 0x3a }

type Box struct {
	Alg uint16 `json:"alg"`
	Nonce []byte `json:"nonce"`
	CipherText []byte `json:"ciphertext"`
}

type Key struct {
	bs [KeySize]byte
}

func (k *Key) Bytes() []byte {
	return k.bs[:]
}

func (k *Key) Close() {
	clear(k.bs[:])
}

func (k *Key) Fingerprint() string {
	fpr := sha256.Sum256(k.bs[:])
	return hex.EncodeToString(fpr[:7])
}

func KeyFromBytes(data []byte) (k *Key, err error) {
	if len(data) != KeySize {
		return nil, fmt.Errorf("unable to unmarshal key from binary; unexpected length: %d != %d", len(data), KeySize)
	}
	k = mkkey()
	k.bs = [KeySize]byte(bytes.Clone(data))
	return k, nil
}

func mkkey() *Key {
	key := &Key{}

	runtime.SetFinalizer(key, func(k *Key) {
		k.Close()
	})

	return key
}

func NewKey() (*Key, error) {
	key := mkkey()

	_, err := rand.Read(key.bs[:])
	if err != nil {
		return nil, err
	}

	return key, nil
}

func NewKeyfile(path string, truncate bool) (*Key, error) {
	key, err := NewKey()
	if err != nil {
		return nil, err
	}

	flags := os.O_WRONLY|os.O_CREATE
	if !truncate {
		flags |= os.O_EXCL
	}
	f, err := os.OpenFile(path, flags, 0600)
	if err != nil {
		key.Close()
		return nil, err
	}
	defer f.Close()

	if _, err = f.Write(key.bs[:]); err != nil {
		key.Close()
		return nil, err
	}

	return key, nil
}

func LoadKeyfile(path string) (*Key, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	bs, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	if len(bs) != KeySize {
		return nil, fmt.Errorf("unusable keyfile (invalid size): %s", path)
	}

	key := mkkey()
	key.bs = [KeySize]byte(bs)

	return key, nil
}

func FreshNonce() (nonce []byte, err error) {
	nonce = make([]byte, NonceSize)
	_, err = rand.Read(nonce)
	return
}

func (box *Box) MarshalBinary() (data []byte, err error) {
	data = make([]byte, len(Magic) + 2 + len(box.Nonce) + len(box.CipherText))
	o := 0

	copy(data[o:], Magic[:])
	o += len(Magic)

	binary.BigEndian.PutUint16(data[o:], box.Alg)
	o += 2

	copy(data[o:], box.Nonce[:])
	o += len(box.Nonce)

	copy(data[o:], box.CipherText[:])
	o += len(box.CipherText)

	if o != len(data) {
		err = fmt.Errorf("incorrect encoded length (implementation error)")
	}

	return
}

func (box *Box) UnmarshalBinary(data []byte) (err error) {
	o := 0

	magic := make([]byte, len(Magic))
	copy(magic, data[o:])
	if !bytes.Equal(magic, Magic[:]) {
		return fmt.Errorf("unexpected magic bytes: %v != %v", magic, Magic)
	}
	o += len(Magic)

	box.Alg = binary.BigEndian.Uint16(data[o:])
	if box.Alg != 1 {
		return fmt.Errorf("unsupported version: %d", box.Alg)
	}
	o += 2

	NonceSize := 12
	box.Nonce = make([]byte, NonceSize)
	copy(box.Nonce, data[o:o+NonceSize])
	o += NonceSize

	box.CipherText = make([]byte, len(data) - o)
	copy(box.CipherText, data [o:] )

	return nil
}

func Seal(key *Key, plaintext []byte) (box *Box, err error) {
	block, err := aes.NewCipher(key.bs[:])
	if err != nil {
		return
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return
	}

	nonce, err := FreshNonce()
	if err != nil {
		return
	}

	box = &Box {
		Alg: 1,
		Nonce: nonce,
		CipherText: aesgcm.Seal(nil, nonce, plaintext, nil),
	}
	return
}

func (box *Box) Open(key *Key) (plaintext []byte, err error) {
	if box.Alg != 1 {
		return nil, fmt.Errorf("unsupported version: %d", box.Alg)
	}

	block, err := aes.NewCipher(key.bs[:])
	if err != nil {
		return
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return
	}

	plaintext, err = aesgcm.Open(nil, box.Nonce, box.CipherText, nil)
	if err != nil {
		return
	}

	return
}
