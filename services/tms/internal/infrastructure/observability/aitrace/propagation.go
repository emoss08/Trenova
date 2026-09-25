package aitrace

import (
	"context"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const traceparentHeader = "traceparent"

func (a Anchor) Link() trace.Link {
	return trace.Link{SpanContext: a.SpanContext()}
}

func IDs(ctx context.Context) (traceID, spanID string) {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return "", ""
	}

	return sc.TraceID().String(), sc.SpanID().String()
}

func Traceparent(ctx context.Context) string {
	if !trace.SpanContextFromContext(ctx).IsValid() {
		return ""
	}

	carrier := propagation.MapCarrier{}
	propagation.TraceContext{}.Inject(ctx, carrier)

	return carrier.Get(traceparentHeader)
}

func LinkFromTraceparent(traceparent string) (trace.Link, bool) {
	if traceparent == "" {
		return trace.Link{}, false
	}

	ctx := propagation.TraceContext{}.Extract(
		context.Background(),
		propagation.MapCarrier{traceparentHeader: traceparent},
	)
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return trace.Link{}, false
	}

	return trace.Link{SpanContext: sc}, true
}

func LinkTo(traceID, spanID string) (trace.Link, bool) {
	tid, err := trace.TraceIDFromHex(traceID)
	if err != nil {
		return trace.Link{}, false
	}
	sid, err := trace.SpanIDFromHex(spanID)
	if err != nil {
		return trace.Link{}, false
	}

	return trace.Link{
		SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
			TraceID: tid,
			SpanID:  sid,
			Remote:  true,
		}),
	}, true
}
