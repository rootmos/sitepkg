package logging

import (
	"log/slog"
	"context"
)

type MultiHandler []slog.Handler

func (mh *MultiHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	for _, h := range *mh {
		if h.Enabled(ctx, lvl) {
			return true
		}
	}
	return false
}

func (mh *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, h := range *mh {
		if h.Enabled(ctx, r.Level) {
			if err := h.Handle(ctx, r); err != nil {
				return err
			}
		}
	}
	return nil
}

func (mh *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	nh := MultiHandler(make([]slog.Handler, len(*mh)))
	for i, h := range *mh {
		// TODO: "The Handler owns the slice: it may retain, modify or discard it."
		// does this imply a deep copy is needed?
		as := make([]slog.Attr, len(attrs))
		copy(as, attrs)

		nh[i] = h.WithAttrs(as)
	}
	return &nh
}

func (mh *MultiHandler) WithGroup(name string) slog.Handler {
	nh := MultiHandler(make([]slog.Handler, len(*mh)))
	for i, h := range *mh {
		nh[i] = h.WithGroup(name)
	}
	return &nh
}
