#!/bin/sh

set -o errexit -o nounset

OUTPUT=$1

if [ -z "${2-}" ]; then
	cat <<EOF >"$OUTPUT"
// Code generated DO NOT EDIT.
package logging

import (
	"log/slog"
	"context"
	"fmt"
)

EOF
	exit 0
fi

LEVEL=$2
VALUE=$3

cat <<EOF >>"$OUTPUT"

const Level$LEVEL = Level($VALUE)

func (l *Logger) $LEVEL(msg string, args ...any) {
	l.log(nil, Level$LEVEL, msg, args...)
}

func (l *Logger) ${LEVEL}Context(ctx context.Context, msg string, args ...any) {
	l.log(ctx, Level$LEVEL, msg, args...)
}

func (l *Logger) ${LEVEL}f(format string, args ...any) {
	l.log(nil, Level$LEVEL, fmt.Sprintf(format, args...))
}

func (l *Logger) ${LEVEL}fContext(ctx context.Context, format string, args ...any) {
	l.log(ctx, Level$LEVEL, fmt.Sprintf(format, args...))
}
EOF
