package osext

import (
	"testing"
	"context"
	"path/filepath"

	logging "rootmos.io/sitepkg/internal/logging/testing"
)

func TestTarballNotExist(t *testing.T) {
	ctx := logging.SetupTestLogger(context.TODO(), t)
	tmp := t.TempDir()
	noent := filepath.Join(tmp, "noent")

	_, err := Open(ctx, noent)
	if !IsNotExist(err) {
		t.Errorf("unexpected error: %v", err)
	}
}

