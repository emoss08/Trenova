package aitrace

import (
	"os"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

var exporter = tracetest.NewInMemoryExporter()

func TestMain(m *testing.M) {
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
		sdktrace.WithIDGenerator(NewIDGenerator()),
		sdktrace.WithSampler(NewSampler(1)),
	)
	otel.SetTracerProvider(provider)

	os.Exit(m.Run())
}

func spansIn(traceID trace.TraceID) tracetest.SpanStubs {
	var out tracetest.SpanStubs
	for _, span := range exporter.GetSpans() {
		if span.SpanContext.TraceID() == traceID {
			out = append(out, span)
		}
	}

	return out
}

func spanNamed(t *testing.T, traceID trace.TraceID, name string) tracetest.SpanStub {
	t.Helper()

	spans := spansIn(traceID)
	for i := len(spans) - 1; i >= 0; i-- {
		if spans[i].Name == name {
			return spans[i]
		}
	}
	t.Fatalf("no span %q in trace %s", name, traceID)

	return tracetest.SpanStub{}
}
