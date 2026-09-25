package aitracetest

import (
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

var (
	installOnce sync.Once
	exporter    *tracetest.InMemoryExporter
)

func Install() *tracetest.InMemoryExporter {
	installOnce.Do(func() {
		exporter = tracetest.NewInMemoryExporter()
		otel.SetTracerProvider(sdktrace.NewTracerProvider(
			sdktrace.WithSyncer(exporter),
			sdktrace.WithIDGenerator(aitrace.NewIDGenerator()),
			sdktrace.WithSampler(aitrace.NewSampler(1)),
		))
	})

	return exporter
}

func SpansIn(traceID trace.TraceID) tracetest.SpanStubs {
	var out tracetest.SpanStubs
	for _, span := range Install().GetSpans() {
		if span.SpanContext.TraceID() == traceID {
			out = append(out, span)
		}
	}

	return out
}

func Named(traceID trace.TraceID, name string) tracetest.SpanStubs {
	var out tracetest.SpanStubs
	for _, span := range SpansIn(traceID) {
		if span.Name == name {
			out = append(out, span)
		}
	}

	return out
}

func One(t *testing.T, traceID trace.TraceID, name string) tracetest.SpanStub {
	t.Helper()

	spans := Named(traceID, name)
	if len(spans) != 1 {
		t.Fatalf("want one span %q in trace %s, found %d", name, traceID, len(spans))
	}

	return spans[0]
}

func Attr(span tracetest.SpanStub, key attribute.Key) (attribute.Value, bool) {
	for _, kv := range span.Attributes {
		if kv.Key == key {
			return kv.Value, true
		}
	}

	return attribute.Value{}, false
}
