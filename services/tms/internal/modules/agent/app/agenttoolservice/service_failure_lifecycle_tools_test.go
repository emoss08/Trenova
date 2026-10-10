package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeFailureLifecycle struct {
	guard    *writeGuard
	refusal  error
	current  *servicefailure.ServiceFailure
	updated  *servicefailureports.UpdateServiceFailureRequest
	reviewed *servicefailureports.ServiceFailureLifecycleRequest
	voided   *servicefailureports.ServiceFailureLifecycleRequest
}

func newFakeFailureLifecycle() *fakeFailureLifecycle {
	return &fakeFailureLifecycle{
		guard: &writeGuard{},
		current: &servicefailure.ServiceFailure{
			ID:            pulid.MustNew("sf_"),
			ShipmentID:    pulid.MustNew("shp_"),
			Number:        "SF-0042",
			Status:        servicefailure.StatusOpen,
			Notes:         "Delivered two hours late",
			InternalNotes: "Driver stuck at the scale",
			Version:       5,
		},
	}
}

func (f *fakeFailureLifecycle) GetByID(
	context.Context,
	*servicefailureports.GetServiceFailureByIDRequest,
) (*servicefailure.ServiceFailure, error) {
	copied := *f.current

	return &copied, nil
}

func (f *fakeFailureLifecycle) apply(
	req *servicefailureports.UpdateServiceFailureRequest,
) *servicefailure.ServiceFailure {
	after := *f.current
	after.Notes = req.Notes
	after.InternalNotes = req.InternalNotes
	after.X12StatusCodeOverride = req.X12StatusCodeOverride
	if req.ReasonCodeID.IsNotNil() {
		reason := req.ReasonCodeID
		after.ReasonCodeID = &reason
	}

	return &after
}

func (f *fakeFailureLifecycle) Update(
	_ context.Context,
	req *servicefailureports.UpdateServiceFailureRequest,
	_ *permissionports.RequestActor,
) (*servicefailure.ServiceFailure, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.updated = req

	return f.apply(req), nil
}

func (f *fakeFailureLifecycle) PreviewUpdate(
	_ context.Context,
	req *servicefailureports.UpdateServiceFailureRequest,
) (*servicefailureports.ServiceFailureLifecyclePreview, error) {
	return &servicefailureports.ServiceFailureLifecyclePreview{Before: f.current, After: f.apply(req)}, nil
}

func (f *fakeFailureLifecycle) Review(
	_ context.Context,
	req *servicefailureports.ServiceFailureLifecycleRequest,
	_ *permissionports.RequestActor,
) (*servicefailure.ServiceFailure, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.reviewed = req

	return f.current, nil
}

func (f *fakeFailureLifecycle) PreviewReview(
	_ context.Context,
	req *servicefailureports.ServiceFailureLifecycleRequest,
	_ *permissionports.RequestActor,
) (*servicefailureports.ServiceFailureLifecyclePreview, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}
	after := *f.current
	after.Status = servicefailure.StatusReviewed
	reason := req.ReasonCodeID
	after.ReasonCodeID = &reason

	return &servicefailureports.ServiceFailureLifecyclePreview{Before: f.current, After: &after}, nil
}

func (f *fakeFailureLifecycle) Void(
	_ context.Context,
	req *servicefailureports.ServiceFailureLifecycleRequest,
	_ *permissionports.RequestActor,
) (*servicefailure.ServiceFailure, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.voided = req

	return f.current, nil
}

func (f *fakeFailureLifecycle) PreviewVoid(
	_ context.Context,
	req *servicefailureports.ServiceFailureLifecycleRequest,
	_ *permissionports.RequestActor,
) (*servicefailureports.ServiceFailureLifecyclePreview, error) {
	after := *f.current
	after.Status = servicefailure.StatusVoided
	after.VoidReason = req.Notes

	return &servicefailureports.ServiceFailureLifecyclePreview{
		Before: f.current,
		After:  &after,
		EDI: &servicefailureports.ServiceFailure214LifecycleResult{
			Action:        ediports.ServiceFailureEDIActionSkipped,
			SkippedReason: ediReadyForGeneration,
		},
	}, nil
}

func TestUpdateServiceFailure_ChangesOnlyWhatIsSent(t *testing.T) {
	t.Parallel()

	failures := newFakeFailureLifecycle()
	tool := newUpdateServiceFailureTool(failures)
	params := executeParams(map[string]any{
		paramServiceFailureID: failures.current.ID.String(),
		paramX12Status:        "ax",
		fieldNotes:            "Consignee closed early",
	})

	preview := previewWithoutWrites(t, failures.guard, func() (*agent.ToolPreview, error) {
		return tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
	})
	change := previewChange(t, preview, 0)
	assert.Equal(t, "AX", fieldByPath(t, change, paramX12Status).After)
	assert.Contains(t, preview.Summary, "SF-0042")

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, failures.updated)
	assert.Equal(t, "Consignee closed early", failures.updated.Notes)
	assert.Equal(t, "Driver stuck at the scale", failures.updated.InternalNotes,
		"an unsent note keeps its value")
	assert.Equal(t, int64(5), failures.updated.Version)

	policy := tool.Policy()
	assert.Equal(t, permission.ResourceServiceFailure, policy.Resource)
	assert.Equal(t, permission.OpUpdate, policy.Operation)
	target, ok := tool.(agentports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, permission.ResourceServiceFailure, target.Resource)

	for name, raw := range map[string]map[string]any{
		"nothing to change": {paramServiceFailureID: failures.current.ID.String()},
		"both reason and clear": {
			paramServiceFailureID: failures.current.ID.String(),
			fieldReasonCodeID:     pulid.MustNew("sfrc_").String(),
			paramClearReasonCode:  true,
		},
		"a long code": {
			paramServiceFailureID: failures.current.ID.String(),
			paramX12Reason:        "ABCD",
		},
	} {
		require.Error(t, tool.(agentports.ToolValidator).Validate(t.Context(),
			executeParams(raw)), name)
	}
}

func TestReviewServiceFailure_IsAPersonsAndAWouldFailWhenRefused(t *testing.T) {
	t.Parallel()

	failures := newFakeFailureLifecycle()
	tool := newReviewServiceFailureTool(failures)
	reason := pulid.MustNew("sfrc_")
	params := executeParams(map[string]any{
		paramServiceFailureID: failures.current.ID.String(),
		fieldReasonCodeID:     reason.String(),
	})

	preview := previewWithoutWrites(t, failures.guard, func() (*agent.ToolPreview, error) {
		return tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, string(servicefailure.StatusReviewed),
		fieldByPath(t, previewChange(t, preview, 0), fieldStatus).After)

	require.ErrorIs(t, tool.Execute(t.Context(), params), ErrNeedsAPersonsApproval)
	approved := approvedParams(params.Params)
	require.NoError(t, tool.Execute(t.Context(), approved))
	assert.Equal(t, reason, failures.reviewed.ReasonCodeID)
	assert.Equal(t, permission.OpApprove, tool.Policy().Operation)
	assert.Equal(t, agent.TierPropose, tool.Policy().MaxTier)

	failures.refusal = errortypes.NewBusinessError("a reason code is required")
	refused, err := tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	requireWarning(t, refused, agent.PreviewWarningWouldFail)
	require.Error(t, tool.(agentports.ToolValidator).Validate(t.Context(), params))
}

func TestVoidServiceFailure_NeedsAReasonAndShowsThe214(t *testing.T) {
	t.Parallel()

	failures := newFakeFailureLifecycle()
	tool := newVoidServiceFailureTool(failures)
	params := approvedParams(map[string]any{
		paramServiceFailureID: failures.current.ID.String(),
		fieldNotes:            "The appointment window was keyed wrong",
	})

	preview := previewWithoutWrites(t, failures.guard, func() (*agent.ToolPreview, error) {
		return tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
	})
	require.Len(t, preview.Changes, 2)
	message := previewChange(t, preview, 1).Message
	require.NotNil(t, message)
	assert.Contains(t, message.Body, "Voided: The appointment window was keyed wrong")

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, "The appointment window was keyed wrong", failures.voided.Notes)
	assert.Equal(t, permission.OpArchive, tool.Policy().Operation)

	require.Error(t, tool.(agentports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{paramServiceFailureID: failures.current.ID.String()})))
}

func TestEvaluateServiceFailures_OneStopOrSeveralShipments(t *testing.T) {
	t.Parallel()

	failures := &fakeFailureDecider{}
	tool := newEvaluateServiceFailuresTool(failures)
	shipmentID, stopID := pulid.MustNew("shp_"), pulid.MustNew("stp_")

	require.NoError(t, tool.Execute(t.Context(), executeParams(map[string]any{
		"shipmentId": shipmentID.String(),
		"stopId":     stopID.String(),
	})))
	require.NotNil(t, failures.evaluatedStop)
	assert.Equal(t, stopID, failures.evaluatedStop.StopID)
	assert.Equal(t, shipmentID, failures.evaluatedStop.ShipmentID)

	many := []any{pulid.MustNew("shp_").String(), pulid.MustNew("shp_").String()}
	require.NoError(t, tool.Execute(t.Context(), executeParams(map[string]any{
		"shipmentIds": many,
		"force":       true,
	})))
	require.NotNil(t, failures.bulk)
	assert.Len(t, failures.bulk.ShipmentIDs, 2)
	assert.True(t, failures.bulk.Force)

	for name, raw := range map[string]map[string]any{
		"neither":            {},
		"both":               {"shipmentId": shipmentID.String(), "shipmentIds": many},
		"a stop without one": {"shipmentIds": many, "stopId": stopID.String()},
		"a bad stop":         {"shipmentId": shipmentID.String(), "stopId": "stop-1"},
		"too many shipments": {"shipmentIds": manyIDs(maxBulkEvaluationShipments + 1)},
	} {
		require.Error(t, tool.(agentports.ToolValidator).Validate(t.Context(),
			executeParams(raw)), name)
	}
}

func manyIDs(n int) []any {
	ids := make([]any, 0, n)
	for range n {
		ids = append(ids, pulid.MustNew("shp_").String())
	}

	return ids
}

func TestDisputeDetention_HoldsTheChargeFromBilling(t *testing.T) {
	t.Parallel()

	detentions := &fakeDetention{guard: &writeGuard{}}
	tool := newDisputeDetentionTool(detentions)
	occurrenceID := pulid.MustNew("dto_")
	params := executeParams(map[string]any{
		paramOccurrenceID: occurrenceID.String(),
		paramNote:         "Customer says the truck arrived after the appointment",
	})

	preview := previewWithoutWrites(t, detentions.guard, func() (*agent.ToolPreview, error) {
		return tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "disputed")
	assert.Contains(t, preview.Summary, "PRO-3001")

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, detentions.disputed)
	assert.Equal(t, occurrenceID, detentions.disputed.OccurrenceID)
	assert.Equal(t, params.Actor.UserID, detentions.disputed.UserID)

	policy := tool.Policy()
	assert.Equal(t, permission.ResourceDetentionPolicy, policy.Resource)
	assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, policy.Egress)

	require.Error(t, tool.(agentports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{paramOccurrenceID: occurrenceID.String()})))
	var _ detentionDisputer = (*detentionservice.Service)(nil)
}
