package complete_handler

import (
	"context"
	"log/slog"
)

var _ slog.Handler = (*TraceHandler)(nil)

type TraceHandler struct { // want "TraceHandler implements slog.Handler but does not implement WithAttrs method" "TraceHandler implements slog.Handler but does not implement WithGroup method"
	slog.Handler
}

func (h *TraceHandler) Handle(ctx context.Context, r slog.Record) error {
	traceID, ok := ctx.Value("traceID").(string)
	if ok && traceID != "" {
		r.AddAttrs(slog.String("traceID", traceID))
	}
	return h.Handler.Handle(ctx, r)
}
