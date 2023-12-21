package main

import (
	"testing"
	"path/filepath"
	"context"

	testinglogging "rootmos.io/sitepkg/internal/logging/testing"
)

func TestTarballNotExist(t *testing.T) {
	ctx := testinglogging.SetupLogger(context.TODO(), t)
	tmp := t.TempDir()
	noent := filepath.Join(tmp, "noent")

	_, err := Open(ctx, noent)
	if !IsNotExist(err) {
		t.Fatalf("unexpected error: %v", err)
	}
}
