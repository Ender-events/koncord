// Package log provides context-scoped structured logging built on top of
// the standard [log/slog] package, following the same pattern used by
// github.com/containerd/log (which carriers a *logrus.Entry in context).
//
// Usage:
//
//	// At a request / goroutine boundary, enrich the context:
//	ctx = log.With(ctx, slog.String("component", "docker"))
//
//	// Anywhere in the call chain, retrieve and use the logger:
//	log.G(ctx).Info("starting container", "id", id)
package log

import (
	"context"
	"log/slog"
)

// key is the unexported context key for the logger to avoid collisions.
type key struct{}

// WithLogger returns a new context carrying the provided logger.
// Use this at top-level boundaries (main, request handlers, goroutines).
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, key{}, logger)
}

// With returns a new context with a logger derived from the current one,
// pre-enriched with the given attributes. If ctx carries no logger yet,
// slog.Default() is used as the base.
//
//	ctx = log.With(ctx, slog.String("component", "docker"))
func With(ctx context.Context, args ...any) context.Context {
	return WithLogger(ctx, G(ctx).With(args...))
}

// G retrieves the logger stored in ctx ("G" stands for "Get", following
// the containerd/log convention). Falls back to slog.Default() if none
// has been set, so it is always safe to call.
func G(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(key{}).(*slog.Logger); ok && logger != nil {
		return logger
	}
	return slog.Default()
}
