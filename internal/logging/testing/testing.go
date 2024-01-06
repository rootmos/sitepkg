package testing

import (
	"testing"
	"context"
	"io"
	"bufio"

	"rootmos.io/sitepkg/internal/logging"
)

func SetupTestLogger(ctx context.Context, t *testing.T) context.Context {
	r, w := io.Pipe()
	ch := make(chan struct{})

	go func() {
		s := bufio.NewScanner(r)
		for s.Scan() {
			t.Log(s.Text())
		}
		if err := s.Err(); err != nil {
			t.Fatalf("unable to read pipe: %v", err)
		}
		ch <- struct{}{}
	}()

	t.Cleanup(func() {
		w.Close()
		<- ch
	})

	logConfig := logging.Config{
		HumanWriter: w,
	}
	logger, err := logConfig.SetupLogger()
	if err != nil {
		t.Fatalf("unable to setup logger: %v", err)
	}
	ctx = logging.Set(ctx, logger)
	_, ctx = logging.WithAttrs(ctx, "test", t.Name())
	return ctx
}
