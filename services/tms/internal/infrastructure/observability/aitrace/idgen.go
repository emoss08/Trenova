package aitrace

import (
	"context"
	"crypto/rand"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type forcedAnchorKey struct{}

var _ sdktrace.IDGenerator = IDGenerator{}

type IDGenerator struct{}

func NewIDGenerator() IDGenerator {
	return IDGenerator{}
}

func (IDGenerator) NewIDs(ctx context.Context) (trace.TraceID, trace.SpanID) {
	if forced, ok := forcedAnchor(ctx); ok {
		return forced.TraceID, forced.RootSpanID
	}

	return randomTraceID(), randomSpanID()
}

func (IDGenerator) NewSpanID(context.Context, trace.TraceID) trace.SpanID {
	return randomSpanID()
}

func withForcedAnchor(ctx context.Context, a Anchor) context.Context {
	return context.WithValue(ctx, forcedAnchorKey{}, a)
}

func forcedAnchor(ctx context.Context) (Anchor, bool) {
	if ctx == nil {
		return Anchor{}, false
	}

	a, ok := ctx.Value(forcedAnchorKey{}).(Anchor)
	if !ok || !a.IsValid() {
		return Anchor{}, false
	}

	return a, true
}

func randomTraceID() trace.TraceID {
	var id trace.TraceID
	for !id.IsValid() {
		_, _ = rand.Read(id[:])
	}

	return id
}

func randomSpanID() trace.SpanID {
	var id trace.SpanID
	for !id.IsValid() {
		_, _ = rand.Read(id[:])
	}

	return id
}
