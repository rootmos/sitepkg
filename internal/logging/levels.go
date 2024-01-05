// Code generated DO NOT EDIT.
package logging

import (
	"log/slog"
	"context"
	"fmt"
)


const LevelTrace = Level(slog.Level(-8))

func (l *Logger) Trace(msg string, args ...any) {
	l.log(nil, LevelTrace, msg, args...)
}

func (l *Logger) TraceContext(ctx context.Context, msg string, args ...any) {
	l.log(ctx, LevelTrace, msg, args...)
}

func (l *Logger) Tracef(format string, args ...any) {
	l.log(nil, LevelTrace, fmt.Sprintf(format, args...))
}

func (l *Logger) TracefContext(ctx context.Context, format string, args ...any) {
	l.log(ctx, LevelTrace, fmt.Sprintf(format, args...))
}

const LevelDebug = Level(slog.LevelDebug)

func (l *Logger) Debug(msg string, args ...any) {
	l.log(nil, LevelDebug, msg, args...)
}

func (l *Logger) DebugContext(ctx context.Context, msg string, args ...any) {
	l.log(ctx, LevelDebug, msg, args...)
}

func (l *Logger) Debugf(format string, args ...any) {
	l.log(nil, LevelDebug, fmt.Sprintf(format, args...))
}

func (l *Logger) DebugfContext(ctx context.Context, format string, args ...any) {
	l.log(ctx, LevelDebug, fmt.Sprintf(format, args...))
}

const LevelInfo = Level(slog.LevelInfo)

func (l *Logger) Info(msg string, args ...any) {
	l.log(nil, LevelInfo, msg, args...)
}

func (l *Logger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.log(ctx, LevelInfo, msg, args...)
}

func (l *Logger) Infof(format string, args ...any) {
	l.log(nil, LevelInfo, fmt.Sprintf(format, args...))
}

func (l *Logger) InfofContext(ctx context.Context, format string, args ...any) {
	l.log(ctx, LevelInfo, fmt.Sprintf(format, args...))
}

const LevelWarn = Level(slog.LevelWarn)

func (l *Logger) Warn(msg string, args ...any) {
	l.log(nil, LevelWarn, msg, args...)
}

func (l *Logger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.log(ctx, LevelWarn, msg, args...)
}

func (l *Logger) Warnf(format string, args ...any) {
	l.log(nil, LevelWarn, fmt.Sprintf(format, args...))
}

func (l *Logger) WarnfContext(ctx context.Context, format string, args ...any) {
	l.log(ctx, LevelWarn, fmt.Sprintf(format, args...))
}

const LevelError = Level(slog.LevelError)

func (l *Logger) Error(msg string, args ...any) {
	l.log(nil, LevelError, msg, args...)
}

func (l *Logger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.log(ctx, LevelError, msg, args...)
}

func (l *Logger) Errorf(format string, args ...any) {
	l.log(nil, LevelError, fmt.Sprintf(format, args...))
}

func (l *Logger) ErrorfContext(ctx context.Context, format string, args ...any) {
	l.log(ctx, LevelError, fmt.Sprintf(format, args...))
}
