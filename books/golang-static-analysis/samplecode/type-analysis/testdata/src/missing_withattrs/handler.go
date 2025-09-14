package missing_withattrs

import (
	"context"
	"log/slog"
)

var _ slog.Handler = (*TraceHandler)(nil)

type TraceHandler struct { // want "TraceHandler implements slog.Handler but does not implement WithAttrs method"
	slog.Handler
}

func (h *TraceHandler) Handle(ctx context.Context, r slog.Record) error {
	traceID, ok := ctx.Value("traceID").(string)
	if ok && traceID != "" {
		r.AddAttrs(slog.String("traceID", traceID))
	}
	return h.Handler.Handle(ctx, r)
}

func (h *TraceHandler) WithGroup(name string) slog.Handler {
	return &TraceHandler{Handler: h.Handler.WithGroup(name)}
}
