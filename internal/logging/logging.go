package logging

import (
	"log/slog"
	"flag"
	"os"
	"context"
	"io"

	"rootmos.io/sitepkg/internal/common"
)

var (
	LogLevelFlag = flag.String("log-level", common.Getenv("LOG_LEVEL"), "set logging level")
	Level = new(slog.LevelVar)
	Key = "logging"
)

func ParseLogLevelFlag() (err error) {
	if LogLevelFlag != nil && *LogLevelFlag != "" {
		var lvl slog.Level
		err = lvl.UnmarshalText([]byte(*LogLevelFlag))
		if err == nil {
			Level.Set(lvl)
		}
	}
	return
}

func SetupLogger(w io.Writer) (*slog.Logger, error) {
	if err := ParseLogLevelFlag(); err != nil {
		return nil, err
	}
	if w == nil {
		w = os.Stderr
	}
	opts := slog.HandlerOptions{ Level: Level }
	logger := slog.New(slog.NewTextHandler(w, &opts))
	return logger, nil
}

func SetupDefaultLogger() (*slog.Logger, error) {
	logger, err := SetupLogger(nil)
	if err == nil {
		slog.SetDefault(logger)
	}
	return logger, err
}

func Set(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, Key, logger)
}

func Get(ctx context.Context) *slog.Logger {
	return ctx.Value(Key).(*slog.Logger)
}

type F func(*slog.Logger) *slog.Logger

func With(ctx context.Context, fs ...func(*slog.Logger) *slog.Logger) (*slog.Logger, context.Context) {
	logger := Get(ctx)
	for _, f := range fs {
		logger = f(logger)
	}
	return logger, Set(ctx, logger)
}

func WithAttrs(ctx context.Context, args ...any) (*slog.Logger, context.Context) {
	return With(ctx, func(l *slog.Logger) *slog.Logger {
		return l.With(args...)
	})
}
