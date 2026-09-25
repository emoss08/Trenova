package aitrace

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
)

func TestStartPreview_RecordsWhatWasFoundAndNoValue(t *testing.T) {
	t.Parallel()

	ctx, request := tracer().Start(t.Context(), "POST /graphql")
	_, span := StartPreview(ctx, &PreviewSpec{
		ToolName:       "cancel_shipment",
		ProposalID:     pulid.ID("ap_preview"),
		OrganizationID: pulid.ID("org_1"),
		BusinessUnitID: pulid.ID("bu_1"),
		Purpose:        PreviewPurposeDecide,
	})
	FinishPreview(span, &PreviewResult{
		Coverage: "Partial",
		Stale:    true,
		Records:  2,
		Withheld: 1,
	})
	RecordDecidedPreview(span, "abc", true)
	span.End()
	request.End()

	preview := spanNamed(t, request.SpanContext().TraceID(), "trenova.ai.preview cancel_shipment")
	assert.Equal(t, request.SpanContext().SpanID(), preview.Parent.SpanID())
	for _, want := range []attribute.KeyValue{
		GenAIToolName.String("cancel_shipment"),
		AIProposalID.String("ap_preview"),
		TenantOrganizationID.String("org_1"),
		AIPreviewPurpose.String(PreviewPurposeDecide),
		AIPreviewCoverage.String("Partial"),
		AIPreviewStale.Bool(true),
		AIPreviewRecords.Int(2),
		AIPreviewWithheld.Int(1),
		AIPreviewRecorded.Bool(false),
		AIPreviewDigest.String("abc"),
		AIPreviewReviewed.Bool(true),
	} {
		assert.Contains(t, preview.Attributes, want)
	}
}
