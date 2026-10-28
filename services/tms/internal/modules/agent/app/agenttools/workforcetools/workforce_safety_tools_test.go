package workforcetools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workersafetyservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workersafety"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSafety struct {
	SafetyKeeper

	guard       *agenttoolstest.WriteGuard
	event       *workersafety.WorkerSafetyEvent
	violation   *workersafety.WorkerSafetyViolation
	recognition *workersafety.WorkerRecognition

	created   *workersafety.WorkerSafetyEvent
	updated   *workersafety.WorkerSafetyEvent
	moved     []*workersafetyservice.EventStatusRequest
	deleted   []pulid.ID
	given     *workersafety.WorkerRecognition
	cited     *workersafetyservice.RecordViolationRequest
	corrected *workersafetyservice.UpdateViolationRequest
}

func newFakeSafety() *fakeSafety {
	workerID := pulid.MustNew("wrk_")
	eventID := pulid.MustNew("wsev_")
	return &fakeSafety{
		guard: &agenttoolstest.WriteGuard{},
		event: &workersafety.WorkerSafetyEvent{
			ID:          eventID,
			WorkerID:    workerID,
			Kind:        workersafety.SafetyEventInspection,
			Severity:    workersafety.SafetySeverityModerate,
			Status:      workersafety.SafetyEventStatusOpen,
			OccurredAt:  1_790_000_000,
			Description: "Level 1 inspection at the Seymour scale",
			Points:      3,
			Version:     4,
		},
		violation: &workersafety.WorkerSafetyViolation{
			ID:             pulid.MustNew("wsvi_"),
			SafetyEventID:  eventID,
			WorkerID:       workerID,
			Basic:          workersafety.BasicVehicleMaintenance,
			Code:           "393.47(e)",
			Description:    "Brake out of adjustment",
			SeverityWeight: 4,
			Version:        2,
		},
		recognition: &workersafety.WorkerRecognition{
			ID:       pulid.MustNew("wrec_"),
			WorkerID: workerID,
			Kind:     workersafety.RecognitionKindTenure,
			Title:    "Five years",
			Version:  1,
		},
	}
}

func (f *fakeSafety) GetEvent(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*workersafety.WorkerSafetyEvent, error) {
	copied := *f.event
	return &copied, nil
}

func (f *fakeSafety) PlanCreateEvent(
	_ context.Context,
	entity *workersafety.WorkerSafetyEvent,
	userID pulid.ID,
) (*workersafety.WorkerSafetyEvent, error) {
	planned := *entity
	planned.RecordedByID = userID
	return &planned, nil
}

func (f *fakeSafety) CreateEvent(
	_ context.Context,
	entity *workersafety.WorkerSafetyEvent,
	_ pulid.ID,
) (*workersafety.WorkerSafetyEvent, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.created = entity
	created := *entity
	created.ID = pulid.MustNew("wsev_")
	return &created, nil
}

func (f *fakeSafety) PlanUpdateEvent(
	_ context.Context,
	entity *workersafety.WorkerSafetyEvent,
) (*workersafetyservice.EventChange, error) {
	return &workersafetyservice.EventChange{Before: f.event, After: entity}, nil
}

func (f *fakeSafety) UpdateEvent(
	_ context.Context,
	entity *workersafety.WorkerSafetyEvent,
	_ pulid.ID,
) (*workersafety.WorkerSafetyEvent, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.updated = entity
	return entity, nil
}

func (f *fakeSafety) planMove(
	req *workersafetyservice.EventStatusRequest,
	status workersafety.SafetyEventStatus,
) *workersafetyservice.EventChange {
	after := *f.event
	after.Status = status
	after.Resolution = req.Resolution
	return &workersafetyservice.EventChange{Before: f.event, After: &after}
}

func (f *fakeSafety) move(
	req *workersafetyservice.EventStatusRequest,
) (*workersafety.WorkerSafetyEvent, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.moved = append(f.moved, req)
	return f.event, nil
}

func (f *fakeSafety) PlanCloseEvent(
	_ context.Context,
	req *workersafetyservice.EventStatusRequest,
) (*workersafetyservice.EventChange, error) {
	return f.planMove(req, workersafety.SafetyEventStatusClosed), nil
}

func (f *fakeSafety) CloseEvent(
	_ context.Context,
	req *workersafetyservice.EventStatusRequest,
) (*workersafety.WorkerSafetyEvent, error) {
	return f.move(req)
}

func (f *fakeSafety) PlanReviewEvent(
	_ context.Context,
	req *workersafetyservice.EventStatusRequest,
) (*workersafetyservice.EventChange, error) {
	return f.planMove(req, workersafety.SafetyEventStatusUnderReview), nil
}

func (f *fakeSafety) ReviewEvent(
	_ context.Context,
	req *workersafetyservice.EventStatusRequest,
) (*workersafety.WorkerSafetyEvent, error) {
	return f.move(req)
}

func (f *fakeSafety) PlanDeleteEvent(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*workersafety.WorkerSafetyEvent, error) {
	if f.event.IsClosed() {
		return nil, errortypes.NewValidationError("status", errortypes.ErrInvalidOperation,
			"Closed events are part of the record and cannot be deleted. Reopen it first")
	}
	return f.event, nil
}

func (f *fakeSafety) DeleteEvent(
	_ context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
	_ pulid.ID,
) error {
	if err := f.guard.Write(); err != nil {
		return err
	}
	f.deleted = append(f.deleted, id)
	return nil
}

func (f *fakeSafety) PlanGiveRecognition(
	_ context.Context,
	entity *workersafety.WorkerRecognition,
	_ pulid.ID,
) (*workersafety.WorkerRecognition, error) {
	return entity, nil
}

func (f *fakeSafety) GiveRecognition(
	_ context.Context,
	entity *workersafety.WorkerRecognition,
	_ pulid.ID,
) (*workersafety.WorkerRecognition, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.given = entity
	created := *entity
	created.ID = pulid.MustNew("wrec_")
	return &created, nil
}

func (f *fakeSafety) PlanRecordViolation(
	_ context.Context,
	req *workersafetyservice.RecordViolationRequest,
) (*workersafety.WorkerSafetyViolation, error) {
	return &workersafety.WorkerSafetyViolation{
		SafetyEventID:  req.SafetyEventID,
		WorkerID:       f.event.WorkerID,
		Basic:          workersafety.BasicVehicleMaintenance,
		Description:    req.Description,
		SeverityWeight: max(req.SeverityWeight, 1),
	}, nil
}

func (f *fakeSafety) RecordViolation(
	_ context.Context,
	req *workersafetyservice.RecordViolationRequest,
) (*workersafety.WorkerSafetyViolation, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.cited = req
	return f.violation, nil
}

func (f *fakeSafety) GetViolation(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*workersafety.WorkerSafetyViolation, error) {
	copied := *f.violation
	return &copied, nil
}

func (f *fakeSafety) PlanUpdateViolation(
	_ context.Context,
	req *workersafetyservice.UpdateViolationRequest,
) (*workersafetyservice.ViolationChange, error) {
	after := *f.violation
	after.Description = req.Description
	after.OutOfService = req.OutOfService
	return &workersafetyservice.ViolationChange{Before: f.violation, After: &after}, nil
}

func (f *fakeSafety) UpdateViolation(
	_ context.Context,
	req *workersafetyservice.UpdateViolationRequest,
) (*workersafety.WorkerSafetyViolation, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.corrected = req
	return f.violation, nil
}

func TestOpenWorkerSafetyEvent_TakesTheHouseScaleForPoints(t *testing.T) {
	t.Parallel()

	events := newFakeSafety()
	tool := newOpenWorkerSafetyEventTool(events)
	params := agenttoolstest.ExecuteParams(map[string]any{
		common.ParamWorkerID:    events.event.WorkerID.String(),
		paramEventKind:          "Accident",
		paramEventSeverity:      "Major",
		wfParamOccurred:         "2026-09-20T14:30:00-05:00",
		common.FieldDescription: "Rear-ended at a light in Joliet",
		paramPreventable:        true,
		paramFineAmount:         "250.00",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		events.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Equal(
		t,
		agent.PreviewOperationCreate,
		agenttoolstest.PreviewChange(t, preview, 0).Operation,
	)
	assert.Contains(t, preview.Summary, "major accident")

	require.NoError(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), params))
	result, err := tool.(agenttoolports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	require.NotNil(t, events.created)
	assert.Equal(t, workersafety.DefaultSafetyPoints(
		workersafety.SafetyEventAccident,
		workersafety.SafetySeverityMajor,
		true,
		workersafety.InspectionResultNone,
	), events.created.Points)
	assert.True(t, events.created.FineAmount.Valid)
	assert.Equal(t, params.OrganizationID, events.created.OrganizationID)
	assert.Contains(t, result.IDs, paramSafetyEventID)

	policy := tool.Policy()
	assert.Equal(t, permission.ResourceWorkerSafetyEvent, policy.Resource)
	assert.Equal(t, permission.OpCreate, policy.Operation)

	for name, raw := range map[string]map[string]any{
		"no description": {
			common.ParamWorkerID: events.event.WorkerID.String(), paramEventKind: "Accident",
			paramEventSeverity: "Major", wfParamOccurred: "2026-09-20T14:30:00-05:00",
		},
		"unknown kind": {
			common.ParamWorkerID: events.event.WorkerID.String(), paramEventKind: "Fire",
			paramEventSeverity: "Major", wfParamOccurred: "2026-09-20T14:30:00-05:00",
			common.FieldDescription: "x",
		},
		"epoch seconds": {
			common.ParamWorkerID: events.event.WorkerID.String(), paramEventKind: "Accident",
			paramEventSeverity: "Major", wfParamOccurred: float64(1_790_000_000),
			common.FieldDescription: "x",
		},
	} {
		require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(),
			agenttoolstest.ExecuteParams(raw)), name)
	}
}

func TestUpdateWorkerSafetyEvent_ChangesOnlyWhatIsNamed(t *testing.T) {
	t.Parallel()

	events := newFakeSafety()
	tool := newUpdateWorkerSafetyEventTool(events)
	params := agenttoolstest.ExecuteParams(map[string]any{
		paramSafetyEventID: events.event.ID.String(),
		paramPoints:        float64(5),
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		events.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.InDelta(
		t,
		5,
		agenttoolstest.FieldByPath(
			t,
			agenttoolstest.PreviewChange(t, preview, 0),
			paramPoints,
		).After,
		0,
	)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, events.updated)
	assert.Equal(t, int32(5), events.updated.Points)
	assert.Equal(t, events.event.Description, events.updated.Description)
	target, ok := tool.(agenttoolports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, permission.ResourceWorkerSafetyEvent, target.Resource)
}

func TestChangeWorkerSafetyEventStatus_ClosingNeedsAResolution(t *testing.T) {
	t.Parallel()

	events := newFakeSafety()
	tool := newChangeWorkerSafetyEventStatusTool(events)
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(),
		agenttoolstest.ExecuteParams(map[string]any{
			paramSafetyEventID: events.event.ID.String(),
			paramEventMove:     "Close",
		})))

	params := agenttoolstest.ExecuteParams(map[string]any{
		paramSafetyEventID:  events.event.ID.String(),
		paramEventMove:      "Close",
		paramResolutionText: "Brakes adjusted and re-inspected",
	})
	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		events.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Equal(
		t,
		string(workersafety.SafetyEventStatusClosed),
		agenttoolstest.FieldByPath(
			t,
			agenttoolstest.PreviewChange(t, preview, 0),
			common.FieldStatus,
		).After,
	)
	require.NoError(t, tool.Execute(t.Context(), params))
	require.Len(t, events.moved, 1)
	assert.Equal(t, permission.OpClose, tool.Policy().Operation)

	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		paramSafetyEventID: events.event.ID.String(),
		paramEventMove:     "Review",
	})))
	assert.Len(t, events.moved, 2)
}

func TestDeleteWorkerSafetyEvent_RefusesAClosedEvent(t *testing.T) {
	t.Parallel()

	events := newFakeSafety()
	tool := newDeleteWorkerSafetyEventTool(events)
	params := agenttoolstest.ExecuteParams(
		map[string]any{paramSafetyEventID: events.event.ID.String()},
	)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		events.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Equal(
		t,
		agent.PreviewOperationDelete,
		agenttoolstest.PreviewChange(t, preview, 0).Operation,
	)
	assert.Equal(t, agent.TierPropose, tool.Policy().MaxTier)
	assert.False(t, tool.Policy().Reversible)

	events.event.Status = workersafety.SafetyEventStatusClosed
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), params))
	assert.Empty(t, events.deleted)
}

func TestGiveWorkerRecognition_TellsTheDriver(t *testing.T) {
	t.Parallel()

	events := newFakeSafety()
	tool := newGiveWorkerRecognitionTool(events)
	params := agenttoolstest.ExecuteParams(map[string]any{
		common.ParamWorkerID: events.event.WorkerID.String(),
		paramRecognitionKind: "SafetyMilestone",
		paramTitle:           "A million safe miles",
		common.FieldMessage:  "Thank you for every one of them.",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		events.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "told in Dash")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, events.given)
	assert.True(t, events.given.VisibleToWorker)
	assert.Equal(t, []agent.EgressClass{agent.EgressDriverVisible}, tool.Policy().Egress)
	assert.Equal(t, permission.ResourceWorkerRecognition, tool.Policy().Resource)
}

func TestSafetyViolationTools_KeepWhatIsNotNamed(t *testing.T) {
	t.Parallel()

	events := newFakeSafety()
	record := newRecordSafetyViolationTool(events)
	params := agenttoolstest.ExecuteParams(map[string]any{
		paramSafetyEventID:      events.event.ID.String(),
		common.FieldDescription: "Inoperative marker lamp",
		paramSeverityWeight:     float64(2),
	})
	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		events.guard,
		func() (*agent.ToolPreview, error) {
			return record.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "severity weight 2")
	require.NoError(t, record.Execute(t.Context(), params))
	require.NotNil(t, events.cited)
	assert.Equal(t, int16(2), events.cited.SeverityWeight)

	update := newUpdateSafetyViolationTool(events)
	require.NoError(t, update.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		paramViolationID:  events.violation.ID.String(),
		paramOutOfService: true,
	})))
	require.NotNil(t, events.corrected)
	assert.True(t, events.corrected.OutOfService)
	assert.Equal(t, events.violation.Description, events.corrected.Description)
	assert.Equal(t, events.violation.Code, events.corrected.Code)

	require.Error(t, record.(agenttoolports.ToolValidator).Validate(t.Context(),
		agenttoolstest.ExecuteParams(map[string]any{
			paramSafetyEventID:      events.event.ID.String(),
			common.FieldDescription: "x",
			paramSeverityWeight:     float64(11),
		})), "a severity weight runs 1 to 10")
}
