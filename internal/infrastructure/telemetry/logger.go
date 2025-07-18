package telemetry

import (
	"context"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
)

type TraceHook struct{}

func (h TraceHook) Run(e *zerolog.Event, _ zerolog.Level, msg string) {
	ctx := e.GetCtx()
	if ctx == nil {
		return
	}

	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return
	}

	e.Str("trace_id", span.SpanContext().TraceID().String())
	e.Str("span_id", span.SpanContext().SpanID().String())
}

func LoggerWithContext(ctx context.Context, logger *zerolog.Logger) *zerolog.Logger {
	l := logger.With().Logger()

	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		l = l.With().
			Str("trace_id", span.SpanContext().TraceID().String()).
			Str("span_id", span.SpanContext().SpanID().String()).
			Logger()
	}

	return &l
}

func ExtractTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}
