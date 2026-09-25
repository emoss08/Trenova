package aitrace

import (
	"context"

	"github.com/emoss08/trenova/shared/pulid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const AIJobWorkflowID = attribute.Key("trenova.ai.job.workflow_id")

type JobSpec struct {
	Feature        string
	WorkflowID     string
	OrganizationID pulid.ID
}

func StartJob(ctx context.Context, spec *JobSpec) (context.Context, trace.Span) {
	attrs := make([]attribute.KeyValue, 0, 3)
	attrs = appendString(attrs, AIJobFeature, spec.Feature)
	attrs = appendString(attrs, AIJobWorkflowID, spec.WorkflowID)
	attrs = appendString(attrs, TenantOrganizationID, spec.OrganizationID.String())

	opts := []trace.SpanStartOption{
		trace.WithNewRoot(),
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(attrs...),
	}
	if caller := trace.SpanContextFromContext(ctx); caller.IsValid() {
		opts = append(opts, trace.WithLinks(trace.Link{SpanContext: caller}))
	}

	name := spanName(SpanJob, spec.Feature)

	return tracer().Start(ctx, name, opts...) //nolint:spancheck // the caller ends it
}
