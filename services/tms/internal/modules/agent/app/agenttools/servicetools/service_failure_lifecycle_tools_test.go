package servicetools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/detentionpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/editools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/receivabletools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/servicefailuretools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionreview"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeFailureLifecycle struct {
	guard    *agenttoolstest.WriteGuard
	refusal  error
	current  *servicefailure.ServiceFailure
	updated  *servicefailureports.UpdateServiceFailureRequest
	reviewed *servicefailureports.ServiceFailureLifecycleRequest
	voided   *servicefailureports.ServiceFailureLifecycleRequest
}

func newFakeFailureLifecycle() *fakeFailureLifecycle {
	return &fakeFailureLifecycle{
		guard: &agenttoolstest.WriteGuard{},
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
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.updated = req

	return f.apply(req), nil
}

func (f *fakeFailureLifecycle) PreviewUpdate(
	_ context.Context,
	req *servicefailureports.UpdateServiceFailureRequest,
) (*servicefailureports.ServiceFailureLifecyclePreview, error) {
	return &servicefailureports.ServiceFailureLifecyclePreview{
		Before: f.current,
		After:  f.apply(req),
	}, nil
}

func (f *fakeFailureLifecycle) Review(
	_ context.Context,
	req *servicefailureports.ServiceFailureLifecycleRequest,
	_ *permissionports.RequestActor,
) (*servicefailure.ServiceFailure, error) {
	if err := f.guard.Write(); err != nil {
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

	return &servicefailureports.ServiceFailureLifecyclePreview{
		Before: f.current,
		After:  &after,
	}, nil
}

func (f *fakeFailureLifecycle) Void(
	_ context.Context,
	req *servicefailureports.ServiceFailureLifecycleRequest,
	_ *permissionports.RequestActor,
) (*servicefailure.ServiceFailure, error) {
	if err := f.guard.Write(); err != nil {
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
			SkippedReason: servicefailuretools.EdiReadyForGeneration,
		},
	}, nil
}

func TestUpdateServiceFailure_ChangesOnlyWhatIsSent(t *testing.T) {
	t.Parallel()

	failures := newFakeFailureLifecycle()
	tool := NewUpdateServiceFailureTool(failures)
	params := agenttoolstest.ExecuteParams(map[string]any{
		editools.ParamServiceFailureID: failures.current.ID.String(),
		paramX12Status:                 "ax",
		toolparams.ParamNote:           "Consignee closed early",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		failures.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, "AX", agenttoolstest.FieldByPath(t, change, paramX12Status).After)
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
	target, ok := tool.(agenttoolports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, permission.ResourceServiceFailure, target.Resource)

	for name, raw := range map[string]map[string]any{
		"nothing to change": {editools.ParamServiceFailureID: failures.current.ID.String()},
		"both reason and clear": {
			editools.ParamServiceFailureID:  failures.current.ID.String(),
			changepreview.FieldReasonCodeID: pulid.MustNew("sfrc_").String(),
			paramClearReasonCode:            true,
		},
		"a long code": {
			editools.ParamServiceFailureID: failures.current.ID.String(),
			paramX12Reason:                 "ABCD",
		},
	} {
		require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(),
			agenttoolstest.ExecuteParams(raw)), name)
	}
}

func TestReviewServiceFailure_IsAPersonsAndAWouldFailWhenRefused(t *testing.T) {
	t.Parallel()

	failures := newFakeFailureLifecycle()
	tool := NewReviewServiceFailureTool(failures)
	reason := pulid.MustNew("sfrc_")
	params := agenttoolstest.ExecuteParams(map[string]any{
		editools.ParamServiceFailureID:  failures.current.ID.String(),
		changepreview.FieldReasonCodeID: reason.String(),
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		failures.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Equal(
		t,
		string(servicefailure.StatusReviewed),
		agenttoolstest.FieldByPath(
			t,
			agenttoolstest.PreviewChange(t, preview, 0),
			changepreview.FieldStatus,
		).After,
	)

	require.ErrorIs(t, tool.Execute(t.Context(), params), toolspec.ErrNeedsAPersonsApproval)
	approved := agenttoolstest.ApprovedParams(params.Params)
	require.NoError(t, tool.Execute(t.Context(), approved))
	assert.Equal(t, reason, failures.reviewed.ReasonCodeID)
	assert.Equal(t, permission.OpApprove, tool.Policy().Operation)
	assert.Equal(t, agent.TierPropose, tool.Policy().MaxTier)

	failures.refusal = errortypes.NewBusinessError("a reason code is required")
	refused, err := tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	agenttoolstest.RequireWarning(t, refused, agentpreview.PreviewWarningWouldFail)
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), params))
}

func TestVoidServiceFailure_NeedsAReasonAndShowsThe214(t *testing.T) {
	t.Parallel()

	failures := newFakeFailureLifecycle()
	tool := NewVoidServiceFailureTool(failures)
	params := agenttoolstest.ApprovedParams(map[string]any{
		editools.ParamServiceFailureID: failures.current.ID.String(),
		toolparams.ParamNote:           "The appointment window was keyed wrong",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		failures.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	require.Len(t, preview.Changes, 2)
	message := agenttoolstest.PreviewChange(t, preview, 1).Message
	require.NotNil(t, message)
	assert.Contains(t, message.Body, "Voided: The appointment window was keyed wrong")

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, "The appointment window was keyed wrong", failures.voided.Notes)
	assert.Equal(t, permission.OpArchive, tool.Policy().Operation)

	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(
		t.Context(),
		agenttoolstest.ExecuteParams(
			map[string]any{editools.ParamServiceFailureID: failures.current.ID.String()},
		),
	))
}

func TestEvaluateServiceFailures_OneStopOrSeveralShipments(t *testing.T) {
	t.Parallel()

	failures := &agenttoolstest.FakeFailureDecider{}
	tool := servicefailuretools.NewEvaluateServiceFailuresTool(failures)
	shipmentID, stopID := pulid.MustNew("shp_"), pulid.MustNew("stp_")

	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"shipmentId": shipmentID.String(),
		"stopId":     stopID.String(),
	})))
	require.NotNil(t, failures.EvaluatedStop)
	assert.Equal(t, stopID, failures.EvaluatedStop.StopID)
	assert.Equal(t, shipmentID, failures.EvaluatedStop.ShipmentID)

	many := []any{pulid.MustNew("shp_").String(), pulid.MustNew("shp_").String()}
	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"shipmentIds": many,
		"force":       true,
	})))
	require.NotNil(t, failures.Bulk)
	assert.Len(t, failures.Bulk.ShipmentIDs, 2)
	assert.True(t, failures.Bulk.Force)

	for name, raw := range map[string]map[string]any{
		"neither":            {},
		"both":               {"shipmentId": shipmentID.String(), "shipmentIds": many},
		"a stop without one": {"shipmentIds": many, "stopId": stopID.String()},
		"a bad stop":         {"shipmentId": shipmentID.String(), "stopId": "stop-1"},
		"too many shipments": {"shipmentIds": agenttoolstest.ManyIDs(servicefailuretools.MaxBulkEvaluationShipments + 1)},
	} {
		require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(),
			agenttoolstest.ExecuteParams(raw)), name)
	}
}

func TestDisputeDetention_HoldsTheChargeFromBilling(t *testing.T) {
	t.Parallel()

	detentions := &agenttoolstest.FakeDetention{Guard: &agenttoolstest.WriteGuard{}}
	tool := receivabletools.NewDisputeDetentionTool(detentions)
	occurrenceID := pulid.MustNew("dto_")
	params := agenttoolstest.ExecuteParams(map[string]any{
		receivabletools.ParamOccurrenceID: occurrenceID.String(),
		toolparams.ParamNote:              "Customer says the truck arrived after the appointment",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		detentions.Guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "disputed")
	assert.Contains(t, preview.Summary, "PRO-3001")

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, detentions.Disputed)
	assert.Equal(t, occurrenceID, detentions.Disputed.OccurrenceID)
	assert.Equal(t, params.Actor.UserID, detentions.Disputed.UserID)

	policy := tool.Policy()
	assert.Equal(t, permission.ResourceDetention, policy.Resource)
	assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, policy.Egress)

	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(
		t.Context(),
		agenttoolstest.ExecuteParams(
			map[string]any{receivabletools.ParamOccurrenceID: occurrenceID.String()},
		),
	))
	var _ detentionpreview.DetentionDisputer = (*detentionreview.Service)(nil)
}
