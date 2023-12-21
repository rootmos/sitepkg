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
	"encoding/hex"
	"crypto/sha256"

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

	dgst := sha256.Sum256(bs)
	t.Logf("populated file: %s (len=%d SHA256=%s)",
		path, n,
		hex.EncodeToString(dgst[:]),
	)

	return
}

func CheckFile(t *testing.T, path string, bs0 []byte) {
	f := Must(os.Open(path))
	defer f.Close()

	bs1 := Must(io.ReadAll(f))
	if !bytes.Equal(bs0, bs1) {
		dgst0 := sha256.Sum256(bs0)
		dgst1 := sha256.Sum256(bs1)
		t.Errorf("content mismatch: %s (actual: len=%d SHA256=%s) (expected: len=%d SHA256=%s)",
			path,
			len(bs0),
			hex.EncodeToString(dgst1[:]),
			len(bs1),
			hex.EncodeToString(dgst0[:]),
		)
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

func TestTarballRoundtripEmptyDirectory(t *testing.T) {
	ctx := testinglogging.SetupLogger(context.TODO(), t)

	tmp := t.TempDir()
	a := filepath.Join(tmp, "a")
	dir := filepath.Join(a, "dir")
	Must0(os.MkdirAll(dir, 0755))

	m0 := &manifest.Manifest {
		Root: a,
		Paths: []string{
			"dir",
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
			"dir",
		},
	}

	if err := m1.Extract(ctx, &buf); err != nil {
		t.Fatalf("unable to extract tarball: %v", err)
	}

	p := filepath.Join(b, "dir")
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatalf("unable to stat; %s: %v", p, err)
	}

	if !fi.IsDir() {
		t.Fatalf("not a directory: %s", p)
	}
}

func TestTarballRoundtripDirectoryNoRecurse(t *testing.T) {
	ctx := testinglogging.SetupLogger(context.TODO(), t)

	tmp := t.TempDir()
	a := filepath.Join(tmp, "a")
	dir := filepath.Join(a, "dir")
	bs := PopulateFile(t, filepath.Join(dir, "foo"))

	m0 := &manifest.Manifest {
		Root: a,
		Paths: []string{
			"dir",
			filepath.Join("dir", "foo"),
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
			"dir",
			filepath.Join("dir", "foo"),
		},
	}

	if err := m1.Extract(ctx, &buf); err != nil {
		t.Fatalf("unable to extract tarball: %v", err)
	}

	CheckFile(t, filepath.Join(b, "dir", "foo"), bs)
}

func TestTarballRoundtripNonEmptyDirectory(t *testing.T) {
	ctx := testinglogging.SetupLogger(context.TODO(), t)

	tmp := t.TempDir()
	a := filepath.Join(tmp, "a")
	dir := filepath.Join(a, "dir")
	_ = PopulateFile(t, filepath.Join(dir, "foo"))

	m0 := &manifest.Manifest {
		Root: a,
		Paths: []string{
			"dir",
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
			"dir",
		},
	}

	if err := m1.Extract(ctx, &buf); err != nil {
		t.Fatalf("unable to extract tarball: %v", err)
	}

	p := filepath.Join(b, "dir")
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatalf("unable to stat; %s: %v", p, err)
	}

	if !fi.IsDir() {
		t.Fatalf("not a directory: %s", p)
	}
}

func TestTarballFailForMissingFilesWhenCreating(t *testing.T) {
	ctx := testinglogging.SetupLogger(context.TODO(), t)

	tmp := t.TempDir()
	a := filepath.Join(tmp, "a")

	m0 := &manifest.Manifest {
		Root: a,
		IgnoreMissing: false,
		Paths: []string{
			"foo",
		},
	}

	var buf bytes.Buffer
	if err := m0.Create(ctx, &buf); !os.IsNotExist(err) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTarballIgnoreMissingFilesWhenCreating(t *testing.T) {
	ctx := testinglogging.SetupLogger(context.TODO(), t)

	tmp := t.TempDir()
	a := filepath.Join(tmp, "a")
	_ = PopulateFile(t, filepath.Join(a, "foo"))

	m0 := &manifest.Manifest {
		Root: a,
		IgnoreMissing: true,
		Paths: []string{
			"foo",
			"bar",
		},
	}

	var buf bytes.Buffer
	if err := m0.Create(ctx, &buf); err != nil {
		t.Fatalf("unable to create tarball: %v", err)
	}
}

func TestTarballFailForMissingFilesWhenExtracting(t *testing.T) {
	ctx := testinglogging.SetupLogger(context.TODO(), t)

	tmp := t.TempDir()
	a := filepath.Join(tmp, "a")

	m0 := &manifest.Manifest {
		Root: a,
	}

	var buf bytes.Buffer
	if err := m0.Create(ctx, &buf); err != nil {
		t.Fatalf("unable to create tarball: %v", err)
	}

	b := filepath.Join(tmp, "b")
	Must0(os.Mkdir(b, 0755))
	m1 := &manifest.Manifest {
		Root: b,
		IgnoreMissing: false,
		Paths: []string{
			"foo",
		},
	}

	if err := m1.Extract(ctx, &buf); err == nil {
		t.Fatalf("unexpected success")
	}
}

func TestTarballIgnoreMissingFilesWhenExtracting(t *testing.T) {
	ctx := testinglogging.SetupLogger(context.TODO(), t)

	tmp := t.TempDir()
	a := filepath.Join(tmp, "a")

	m0 := &manifest.Manifest {
		Root: a,
	}

	var buf bytes.Buffer
	if err := m0.Create(ctx, &buf); err != nil {
		t.Fatalf("unable to create tarball: %v", err)
	}

	b := filepath.Join(tmp, "b")
	Must0(os.Mkdir(b, 0755))
	m1 := &manifest.Manifest {
		Root: b,
		IgnoreMissing: true,
		Paths: []string{
			"foo",
		},
	}

	if err := m1.Extract(ctx, &buf); err != nil {
		t.Fatalf("unexpected failure: %v", err)
	}
}

func TestTarballOverwriteFile(t *testing.T) {
	ctx := testinglogging.SetupLogger(context.TODO(), t)

	tmp := t.TempDir()
	a := filepath.Join(tmp, "a")
	foo := filepath.Join(a, "foo")
	bs0 := PopulateFile(t, foo)

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
	foo = filepath.Join(b, "foo")
	_ = PopulateFile(t, foo)
	m1 := &manifest.Manifest {
		Root: b,
		Paths: []string{
			"foo",
		},
	}

	if err := m1.Extract(ctx, &buf); err != nil {
		t.Fatalf("unable to extract tarball: %v", err)
	}

	CheckFile(t, foo, bs0)
}

func TestTarballExistingDir(t *testing.T) {
	ctx := testinglogging.SetupLogger(context.TODO(), t)

	tmp := t.TempDir()
	a := filepath.Join(tmp, "a")
	dir := filepath.Join(a, "dir")
	Must0(os.MkdirAll(dir, 0755))

	m0 := &manifest.Manifest {
		Root: a,
		Paths: []string{
			"dir",
		},
	}

	var buf bytes.Buffer
	if err := m0.Create(ctx, &buf); err != nil {
		t.Fatalf("unable to create tarball: %v", err)
	}

	b := filepath.Join(tmp, "b")
	dir = filepath.Join(b, "dir")
	Must0(os.MkdirAll(dir, 0755))
	m1 := &manifest.Manifest {
		Root: b,
		Paths: []string{
			"dir",
		},
	}

	if err := m1.Extract(ctx, &buf); err != nil {
		t.Fatalf("unable to extract tarball: %v", err)
	}
}
