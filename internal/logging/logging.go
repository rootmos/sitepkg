package logging

import (
	"log/slog"
	"flag"
	"os"
	"context"

	"rootmos.io/sitepkg/internal/common"
)

var (
	LogLevelFlag = flag.String("log-level", common.Getenv("LOG_LEVEL"), "set logging level")
	Key = "logging"
)

func SetupDefaultLogger() (*slog.Logger, error) {
	level := slog.LevelInfo
	if *LogLevelFlag != "" {
		err := level.UnmarshalText([]byte(*LogLevelFlag))
		if err != nil {
			return nil, err
		}
	}
	opts := slog.HandlerOptions{ Level: level }
	logger := slog.New(slog.NewTextHandler(os.Stderr, &opts))
	slog.SetDefault(logger)
	return logger, nil
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
