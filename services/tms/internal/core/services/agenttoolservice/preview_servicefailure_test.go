package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// PreviewResolve answers from the request, so the fake's record of the
// write stays empty unless the tool resolves.
func (f *fakeFailureDecider) PreviewResolve(
	_ context.Context,
	req *serviceports.ServiceFailureLifecycleRequest,
	actor *serviceports.RequestActor,
) (*serviceports.ServiceFailureLifecyclePreview, error) {
	after := *f.existing
	after.Status = servicefailure.StatusResolved
	after.InternalNotes = req.Notes
	after.ResolvedByID = &actor.UserID
	if req.ReasonCodeID.IsNotNil() {
		reasonID := req.ReasonCodeID
		after.ReasonCodeID = &reasonID
		after.ReasonCode = &servicefailure.ReasonCode{
			Code:        "LATE_SHIPPER",
			Description: "Shipper closed early",
		}
	}

	return &serviceports.ServiceFailureLifecyclePreview{
		Before: f.existing,
		After:  &after,
		EDI: &serviceports.ServiceFailure214LifecycleResult{
			Action:        serviceports.ServiceFailureEDIActionSkipped,
			SkippedReason: ediReadyForGeneration,
			EDIPartnerID:  pulid.MustNew("edip_"),
		},
	}, nil
}

func TestResolveServiceFailurePreview_ShowsTheResolutionAndTheCustomer214(t *testing.T) {
	t.Parallel()

	existing := openFailure()
	failures := &fakeFailureDecider{existing: existing}
	tool := newResolveServiceFailureTool(failures)
	params := executeParams(map[string]any{
		"serviceFailureId": existing.ID.String(),
		"reasonCodeId":     pulid.MustNew("sfrc_").String(),
		"notes":            "Shipper closed early; driver waited until opening.",
	})

	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	require.Nil(t, failures.resolved, "a preview must not resolve")
	assert.Contains(t, preview.Summary, "SF-7")
	assert.Contains(t, preview.Summary, "LATE_SHIPPER (Shipper closed early)")

	change := findChange(t, preview, agent.PreviewOperationUpdate)
	assert.Equal(t, permission.ResourceServiceFailure, change.Resource)
	assert.Equal(t, "Resolved", findField(t, change, "status").After)
	reason := findField(t, change, "reasonCodeId")
	require.NotNil(t, reason.AfterRef)
	assert.Equal(t, permission.ResourceServiceFailureReasonCode, reason.AfterRef.Resource)

	send := findChange(t, preview, agent.PreviewOperationSend)
	assert.Equal(t, agent.MessageChannelEDI, send.Message.Channel)
	assert.Contains(t, send.Message.Body, "Shipper closed early; driver waited")

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, failures.resolved)
	assert.Equal(t, failures.resolved.Notes, findField(t, change, "internalNotes").After)
}
