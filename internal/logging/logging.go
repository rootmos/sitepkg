package logging

import (
	"context"
	"flag"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"time"
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
	HumanWriter io.Writer
	humanLogLevelRaw *string
	HumanLevel Level

	JsonWriter io.Writer
	jsonLogLevelRaw *string
	JsonLevel Level

	Handlers []slog.Handler
}

func PrepareConfig(envPrefix string) Config {
	getenv := func(key string) string {
		return os.Getenv(envPrefix + key)
	}
	return Config {
		HumanWriter: os.Stderr,
		humanLogLevelRaw: flag.String("log-level", getenv("LOG_LEVEL"), "set logging level"),
		HumanLevel: LevelInfo,

		jsonLogLevelRaw: flag.String("json-log-level", getenv("JSON_LOG_LEVEL"), "set JSON logging level"),
		JsonLevel: LevelInfo,
	}
}

func (c *Config) SetupLogger() (l *Logger, err error) {
	hs := c.Handlers
	if c.HumanWriter != nil {
		level := c.HumanLevel
		if c.humanLogLevelRaw != nil && *c.humanLogLevelRaw != "" {
			if level, err = parseLogLevel(*c.humanLogLevelRaw); err != nil {
				return nil, err
			}
		}

		hs = append(hs, &HumanHandler {
			w: c.HumanWriter,
			level: level,
		})
	}

	if c.JsonWriter != nil {
		level := c.JsonLevel
		if c.jsonLogLevelRaw != nil && *c.jsonLogLevelRaw != "" {
			if level, err = parseLogLevel(*c.jsonLogLevelRaw); err != nil {
				return nil, err
			}
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

		hs = append(hs, slog.NewJSONHandler(c.JsonWriter, &opts))
	}

	var inner *slog.Logger
	switch len(hs) {
	case 0:
		inner = slog.Default() // TODO: or use a NullHandler?
	case 1:
		inner = slog.New(hs[0])
	default:
		mh := MultiHandler(hs)
		inner = slog.New(&mh)
	}

	return &Logger{ inner: inner }, nil
}

func (c *Config) SetupDefaultLogger() (*Logger, error) {
	logger, err := c.SetupLogger()
	if err != nil {
		return nil, err
	}

	slog.SetDefault(logger.inner)
	return logger, nil
}

func (l *Logger) log(ctx context.Context, lvl Level, msg string, args... any) {
	slvl := slog.Level(lvl)
	if !l.inner.Enabled(ctx, slvl) {
		return
	}

	var pcs [1]uintptr
	runtime.Callers(3, pcs[:])
	pc := pcs[0] - 1

	r := slog.NewRecord(time.Now(), slvl, msg, pc)
	r.Add(args...)

	caller := runtime.FuncForPC(pc)
	if caller != nil {
		file, line := caller.FileLine(pc)
		r.Add(slog.Group("caller",
			slog.String("name", caller.Name()),
			slog.String("file", file),
			slog.Int("line", line),
		))
	}

	r.Add(slog.Int("pid", os.Getpid()))

	_ = l.inner.Handler().Handle(ctx, r)
}
