package aitrace

import (
	"context"

	"github.com/emoss08/trenova/shared/pulid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type DelegateOpenSpec struct {
	Anchor            Anchor
	OwnerID           pulid.ID
	DelegateCallID    string
	DelegateAgentID   pulid.ID
	DelegateAgentName string
}

func StartDelegateOpen(ctx context.Context, spec *DelegateOpenSpec) (context.Context, trace.Span) {
	attrs := make([]attribute.KeyValue, 0, 3)
	attrs = appendString(attrs, AIDelegateCallID, spec.DelegateCallID)
	attrs = appendString(attrs, AIDelegateAgentID, spec.DelegateAgentID.String())
	attrs = appendString(attrs, AIDelegateAgentName, spec.DelegateAgentName)

	var links []trace.Link
	if delegate := ForDelegate(spec.OwnerID, spec.DelegateCallID); delegate.IsValid() {
		links = []trace.Link{delegate.Link()}
	}

	return start(ctx, spec.Anchor, SpanDelegateOpen, trace.SpanKindInternal, attrs, links)
}
