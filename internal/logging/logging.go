package logging

import (
	"log/slog"
	"flag"
	"os"
	"context"
	"io"
	"strings"
)

const Key = "logger"

type Logger struct {
	inner *slog.Logger
}

func (l *Logger) With(args ...any) *Logger {
	return &Logger { inner: l.inner.With(args...) }
}

func Set(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, Key, logger)
}

func Get(ctx context.Context) *Logger {
	switch v := ctx.Value(Key).(type) {
	case *Logger:
		return v
	case *slog.Logger:
		return &Logger { inner: v }
	default:
		return &Logger { inner: slog.Default() }

	}
}

func WithAttrs(ctx context.Context, args ...any) (*Logger, context.Context) {
	logger := Get(ctx).With(args...)
	return logger, Set(ctx, logger)
}

type Level slog.Level

//go:generate ./generate_level.sh levels.go
//go:generate ./generate_level.sh levels.go Trace slog.Level(-8)
//go:generate ./generate_level.sh levels.go Debug slog.LevelDebug
//go:generate ./generate_level.sh levels.go Info slog.LevelInfo
//go:generate ./generate_level.sh levels.go Warn slog.LevelWarn
//go:generate ./generate_level.sh levels.go Error slog.LevelError

func parseLogLevel(s string) (Level, error) {
	switch strings.ToUpper(s) {
	case "TRACE":
		return LevelTrace, nil
	}

	var l slog.Level
	if err := l.UnmarshalText([]byte(s)); err != nil {
		return 0, err
	}

	return Level(l), nil
}

type Config struct {
	logLevelRaw *string
	DefaultLevel Level
}

func PrepareConfig(envPrefix string) Config {
	getenv := func(key string) string {
		return os.Getenv(envPrefix + key)
	}
	return Config {
		logLevelRaw: flag.String("log-level", getenv("LOG_LEVEL"), "set logging level"),
	}
}

func (c *Config) SetupLogger(w io.Writer) (l *Logger, err error) {
	level := c.DefaultLevel
	if c.logLevelRaw != nil && *c.logLevelRaw != "" {
		if level, err = parseLogLevel(*c.logLevelRaw); err != nil {
			return nil, err
		}
	}

	if w == nil {
		w = os.Stderr
	}

	opts := slog.HandlerOptions {
		Level: slog.Level(level),
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				level := a.Value.Any().(slog.Level)
				if level == slog.Level(LevelTrace) {
					a.Value = slog.StringValue("TRACE")
				}
			}
			return a
		},
	}

	inner := slog.New(slog.NewTextHandler(w, &opts))

	return &Logger{ inner: inner }, nil
}

func (c *Config) SetupDefaultLogger() (*Logger, error) {
	logger, err := c.SetupLogger(nil)
	if err != nil {
		return nil, err
	}

	slog.SetDefault(logger.inner)
	return logger, nil
}
