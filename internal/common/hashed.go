package common

import (
	"hash"
	"io"
	"crypto/sha256"
	"encoding/hex"
	"encoding/base64"
)

type ReaderHashed struct {
	io.Reader
	hash hash.Hash
}

func ReaderSHA256(r io.Reader) (rh *ReaderHashed) {
	h := sha256.New()

	return &ReaderHashed {
		Reader: io.TeeReader(r, h),
		hash: h,
	}
}

func (rh *ReaderHashed) Digest() []byte {
	return rh.hash.Sum(nil)
}

func (rh *ReaderHashed) HexDigest() string {
	return hex.EncodeToString(rh.Digest())
}

func (rh *ReaderHashed) B64Digest() string {
	return base64.StdEncoding.EncodeToString(rh.Digest())
}


type WriterHashed struct {
	io.Writer
	hash hash.Hash
}

func WriterSHA256(w io.Writer) (wh *WriterHashed) {
	h := sha256.New()

	return &WriterHashed {
		Writer: io.MultiWriter(w, h),
		hash: h,
	}
}

func (wh *WriterHashed) Digest() []byte {
	return wh.hash.Sum(nil)
}

func (wh *WriterHashed) HexDigest() string {
	return hex.EncodeToString(wh.Digest())
}

func (wh *WriterHashed) B64Digest() string {
	return base64.StdEncoding.EncodeToString(wh.Digest())
}
