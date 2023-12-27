package sealedbox

import (
	"bytes"
	"encoding/json"
	"log"
	"math/rand"
	"testing"
	"time"
)

var seed = time.Now().UnixNano()
var prng = rand.New(rand.NewSource(seed))

func Must0(err error) {
	if err != nil {
		log.Fatalf("a must failed: %v", err)
	}
}

func Must[T any](obj T, err error) T {
	if err != nil {
		log.Fatalf("a must failed: %v", err)
	}
	return obj
}

func FreshBytes() []byte {
	n := prng.Intn(4096)
	bs := make([]byte, n)
	_ = Must(prng.Read(bs))
	return bs
}

func FiddleWithBytes(xs []byte) []byte {
	ys := bytes.Clone(xs)

	for bytes.Equal(xs, ys) {
		N := prng.Intn(10)
		for n := 0; n < N; n++ {
			b := make([]byte, 1)
			Must(prng.Read(b))
			i := prng.Intn(len(ys))
			ys[i] = b[0]
		}
	}

	return ys
}

func TestRoundtrip(t *testing.T) {
	key := Must(NewKey())
	defer key.Close()

	pt0 := FreshBytes()

	box, err := Seal(key, pt0)
	if err != nil {
		t.Errorf("unable to seal plaintext")
	}

	pt1, err := box.Open(key)
	if err != nil {
		t.Errorf("unable to open box")
	}

	if !bytes.Equal(pt0, pt1) {
		t.Errorf("incorrect plaintext")
	}
}

func TestRoundtripJSON(t *testing.T) {
	key := Must(NewKey())
	defer key.Close()

	pt0 := FreshBytes()

	b0, err := Seal(key, pt0)
	if err != nil {
		t.Errorf("unable to seal plaintext")
	}

	bs, err := json.Marshal(b0)
	if err != nil {
		t.Errorf("unable to marshal box to JSON")
	}

	var b1 Box
	if err := json.Unmarshal(bs, &b1); err != nil {
		t.Errorf("unable to unmarshal box from JSON")
	}

	pt1, err := b1.Open(key)
	if err != nil {
		t.Errorf("unable to open box")
	}

	if !bytes.Equal(pt0, pt1) {
		t.Errorf("incorrect plaintext")
	}
}

func TestRoundtripBinary(t *testing.T) {
	key := Must(NewKey())
	defer key.Close()

	pt0 := FreshBytes()

	b0, err := Seal(key, pt0)
	if err != nil {
		t.Errorf("unable to seal plaintext")
	}

	bs, err := b0.MarshalBinary()
	if err != nil {
		t.Errorf("unable to marshal box to binary")
	}

	var b1 Box
	if err := b1.UnmarshalBinary(bs); err != nil {
		t.Errorf("unable to unmarshal box from binary")
	}

	pt1, err := b1.Open(key)
	if err != nil {
		t.Errorf("unable to open box")
	}

	if !bytes.Equal(pt0, pt1) {
		t.Errorf("incorrect plaintext")
	}
}

func TestIncorrectKey(t *testing.T) {
	key := Must(NewKey())
	defer key.Close()

	pt := FreshBytes()
	box, err := Seal(key, pt)
	if err != nil {
		t.Errorf("unable to seal plaintext")
	}

	key.bs = [KeySize]byte(FiddleWithBytes(key.bs[:]))

	_, err = box.Open(key)
	if err == nil {
		t.Errorf("unexpected success")
	}
}

func TestModifiedCipherText(t *testing.T) {
	key := Must(NewKey())
	defer key.Close()

	pt := FreshBytes()
	box, err := Seal(key, pt)
	if err != nil {
		t.Errorf("unable to seal plaintext")
	}

	box.CipherText = FiddleWithBytes(box.CipherText)

	_, err = box.Open(key)
	if err == nil {
		t.Errorf("unexpected success")
	}
}

func TestModifiedNonce(t *testing.T) {
	key := Must(NewKey())
	defer key.Close()

	pt := FreshBytes()
	box, err := Seal(key, pt)
	if err != nil {
		t.Errorf("unable to seal plaintext")
	}

	box.Nonce = FiddleWithBytes(box.Nonce)

	_, err = box.Open(key)
	if err == nil {
		t.Errorf("unexpected success")
	}
}
