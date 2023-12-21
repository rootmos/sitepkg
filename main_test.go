package main

import (
	"log"
	"testing"
	"path/filepath"
	"context"
	"os"
	"math/rand"
	"time"
	"bytes"
	"io"

	"rootmos.io/sitepkg/internal/manifest"
	testinglogging "rootmos.io/sitepkg/internal/logging/testing"
)

func Must0(err error) {
	if err != nil {
		log.Fatalf("must fail: %v", err)
	}
}

func Must[T any](obj T, err error) T {
	if err != nil {
		log.Fatalf("must fail: %v", err)
	}
	return obj
}

func TestTarballNotExist(t *testing.T) {
	ctx := testinglogging.SetupLogger(context.TODO(), t)
	tmp := t.TempDir()
	noent := filepath.Join(tmp, "noent")

	_, err := Open(ctx, noent)
	if !IsNotExist(err) {
		t.Errorf("unexpected error: %v", err)
	}
}

func PopulateFile(t *testing.T, path string) (bs []byte) {
	Must0(os.MkdirAll(filepath.Dir(path), 0755))
	f := Must(os.Create(path))
	defer f.Close()

	seed := time.Now().UnixNano()
	r := rand.New(rand.NewSource(seed))

	n := r.Intn(4096)

	bs = make([]byte, n)

	_ = Must(r.Read(bs))

	_ = Must(io.Copy(f, bytes.NewReader(bs)))

	return
}

func CheckFile(t *testing.T, path string, bs []byte) {
	f := Must(os.Open(path))
	defer f.Close()

	bs0 := Must(io.ReadAll(f))
	if !bytes.Equal(bs, bs0) {
		t.Errorf("content mismatch: %s", path)
	}
}

func TestTarballRoundtripOneFileAtRoot(t *testing.T) {
	ctx := testinglogging.SetupLogger(context.TODO(), t)

	tmp := t.TempDir()
	a := filepath.Join(tmp, "a")
	foo := filepath.Join(a, "foo")
	bs := PopulateFile(t, foo)

	m0 := &manifest.Manifest {
		Root: a,
		Paths: []string{
			"foo",
		},
	}

	var buf bytes.Buffer
	if err := m0.Create(ctx, &buf); err != nil {
		t.Fatalf("unable to create tarball: %v", err)
	}

	b := filepath.Join(tmp, "b")
	Must0(os.Mkdir(b, 0755))
	m1 := &manifest.Manifest {
		Root: b,
		Paths: []string{
			"foo",
		},
	}

	if err := m1.Extract(ctx, &buf); err != nil {
		t.Fatalf("unable to extract tarball: %v", err)
	}

	CheckFile(t, filepath.Join(b, "foo"), bs)
}
