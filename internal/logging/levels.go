// Code generated DO NOT EDIT.
package logging

import (
	"log/slog"
	"context"
)

const LevelTrace = Level(slog.Level(-8))

func (l *Logger) Trace(msg string, args ...any) {
	l.inner.Log(nil, slog.Level(LevelTrace), msg, args...)
}

func (l *Logger) TraceContext(ctx context.Context, msg string, args ...any) {
	l.inner.Log(ctx, slog.Level(LevelTrace), msg, args...)
}

const LevelDebug = Level(slog.LevelDebug)

func (l *Logger) Debug(msg string, args ...any) {
	l.inner.Log(nil, slog.Level(LevelDebug), msg, args...)
}

func (l *Logger) DebugContext(ctx context.Context, msg string, args ...any) {
	l.inner.Log(ctx, slog.Level(LevelDebug), msg, args...)
}

const LevelInfo = Level(slog.LevelInfo)

func (l *Logger) Info(msg string, args ...any) {
	l.inner.Log(nil, slog.Level(LevelInfo), msg, args...)
}

func (l *Logger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.inner.Log(ctx, slog.Level(LevelInfo), msg, args...)
}

const LevelWarn = Level(slog.LevelWarn)

func (l *Logger) Warn(msg string, args ...any) {
	l.inner.Log(nil, slog.Level(LevelWarn), msg, args...)
}

func (l *Logger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.inner.Log(ctx, slog.Level(LevelWarn), msg, args...)
}

const LevelError = Level(slog.LevelError)

func (l *Logger) Error(msg string, args ...any) {
	l.inner.Log(nil, slog.Level(LevelError), msg, args...)
}

func (l *Logger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.inner.Log(ctx, slog.Level(LevelError), msg, args...)
}
