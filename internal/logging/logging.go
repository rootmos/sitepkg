package logging

import (
	"context"
	"flag"
	"fmt"
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

	h := HumanHandler {
		w: w,
		level: level,
	}

	inner := slog.New(&h)

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

type HumanHandler struct {
	w io.Writer
	level Level
}

func (h *HumanHandler) Enabled(_ context.Context, lvl slog.Level) bool {
	return h.w != nil && lvl >= slog.Level(h.level)
}

const TimeLayout = "20060102T150405Z"

func (h *HumanHandler) Handle(_ context.Context, r slog.Record) (err error) {
	var sb strings.Builder
	if _, err = sb.WriteString(r.Time.UTC().Format(TimeLayout)); err != nil {
		return err
	}

	var pid int64 = -1
	var caller string
	var file string
	var line int64 = -1
	var as strings.Builder
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "pid" && a.Value.Kind() == slog.KindInt64 {
			pid = a.Value.Int64()
			return true
		}

		if a.Key == "caller" && a.Value.Kind() == slog.KindGroup {
			for _, b := range a.Value.Group() {
				if b.Key == "name" && b.Value.Kind() == slog.KindString {
					caller = b.Value.String()
				}

				if b.Key == "file" && b.Value.Kind() == slog.KindString {
					file = b.Value.String()
				}

				if b.Key == "line" && b.Value.Kind() == slog.KindInt64 {
					line = b.Value.Int64()
				}
			}

			return true
		}

		if _, err = fmt.Fprintf(&as, " (%s: %v)", a.Key, a.Value); err != nil {
			return false
		}

		return true
	})
	if err != nil {
		return err
	}

	if pid >= 0 {
		if _, err = fmt.Fprintf(&sb, ":%d", pid); err != nil {
			return err
		}
	}

	if caller != "" {
		if _, err = fmt.Fprintf(&sb, ":%s", caller); err != nil {
			return err
		}
	}

	if file != "" {
		if _, err = fmt.Fprintf(&sb, ":%s", file); err != nil {
			return err
		}
	}

	if line >= 0 {
		if _, err = fmt.Fprintf(&sb, ":%d", line); err != nil {
			return err
		}
	}

	if err = sb.WriteByte(' '); err != nil {
		return err
	}

	if _, err = sb.WriteString(r.Message); err != nil {
		return err
	}

	if _, err = sb.WriteString(as.String()); err != nil {
		return err
	}

	if err = sb.WriteByte('\n'); err != nil {
		return err
	}

	_, err = io.Copy(h.w, strings.NewReader(sb.String()))
	return err
}

func (h *HumanHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *HumanHandler) WithGroup(name string) slog.Handler {
	return h
}
