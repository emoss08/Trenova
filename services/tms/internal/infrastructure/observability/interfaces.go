package observability

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Tracer interface {
	StartSpan(
		ctx context.Context,
		name string,
		opts ...trace.SpanStartOption,
	) (context.Context, trace.Span)
	AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue)
	SetAttributes(ctx context.Context, attrs ...attribute.KeyValue)
	RecordError(ctx context.Context, err error, opts ...trace.EventOption)
	IsEnabled() bool
}

var _ Tracer = (*TracerProvider)(nil)
