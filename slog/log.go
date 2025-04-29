package slog

import (
	"context"

	"log/slog"

	"github.com/gojek/courier-go"
)

// New returns a new courier.Logger that wraps the slog.Handler.
func New(h slog.Handler) courier.Logger {
	return &slogWrapper{log: slog.New(h)}
}

var _ courier.Logger = (*slogWrapper)(nil)

type slogWrapper struct {
	log *slog.Logger
}

func (sw *slogWrapper) Info(ctx context.Context, msg string, attrs map[string]any) {
	sw.log.LogAttrs(ctx, slog.LevelInfo, msg, sw.mapAttrs(attrs)...)
}

func (sw *slogWrapper) Error(ctx context.Context, err error, attrs map[string]any) {
	sw.log.LogAttrs(ctx, slog.LevelError, err.Error(), sw.mapAttrs(attrs)...)
}

func (sw *slogWrapper) Warn(ctx context.Context, msg string, attrs map[string]any) {
	sw.log.LogAttrs(ctx, slog.LevelWarn, msg, sw.mapAttrs(attrs)...)
}

func (sw *slogWrapper) Debug(ctx context.Context, msg string, attrs map[string]any) {
	sw.log.LogAttrs(ctx, slog.LevelDebug, msg, sw.mapAttrs(attrs)...)
}

func (sw *slogWrapper) mapAttrs(attrs map[string]any) []slog.Attr {
	logAttrs := make([]slog.Attr, 0, len(attrs))

	for k, v := range attrs {
		logAttrs = append(logAttrs, slog.Attr{Key: k, Value: slog.AnyValue(v)})
	}

	return logAttrs
}
