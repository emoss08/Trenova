package aitrace

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const ScopeName = "github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"

type RootSpec struct {
	Name              string
	Start             time.Time
	End               time.Time
	Attrs             []attribute.KeyValue
	Links             []trace.Link
	Status            codes.Code
	StatusDescription string
}

func tracer() trace.Tracer {
	return otel.Tracer(ScopeName)
}

func EmitRoot(ctx context.Context, a Anchor, spec *RootSpec) {
	if spec == nil {
		return
	}
	if !a.IsValid() {
		return
	}

	end := spec.End
	if end.IsZero() {
		end = time.Now()
	}
	start := spec.Start
	if start.IsZero() || start.After(end) {
		start = end
	}

	attrs := make([]attribute.KeyValue, 0, len(spec.Attrs)+1)
	attrs = append(attrs, spec.Attrs...)
	attrs = append(attrs, AIAnchor.Bool(true))

	_, span := tracer().Start(
		withForcedAnchor(ctx, a),
		spec.Name,
		trace.WithNewRoot(),
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithTimestamp(start),
		trace.WithAttributes(attrs...),
		trace.WithLinks(spec.Links...),
	)
	if spec.Status != codes.Unset {
		span.SetStatus(spec.Status, spec.StatusDescription)
	}
	span.End(trace.WithTimestamp(end))
}
