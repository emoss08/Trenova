package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/workersafetyservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSafety struct {
	safetyKeeper

	guard       *writeGuard
	event       *worker.WorkerSafetyEvent
	violation   *worker.WorkerSafetyViolation
	recognition *worker.WorkerRecognition

	created   *worker.WorkerSafetyEvent
	updated   *worker.WorkerSafetyEvent
	moved     []*workersafetyservice.EventStatusRequest
	deleted   []pulid.ID
	given     *worker.WorkerRecognition
	cited     *workersafetyservice.RecordViolationRequest
	corrected *workersafetyservice.UpdateViolationRequest
}

func newFakeSafety() *fakeSafety {
	workerID := pulid.MustNew("wrk_")
	eventID := pulid.MustNew("wsev_")
	return &fakeSafety{
		guard: &writeGuard{},
		event: &worker.WorkerSafetyEvent{
			ID:          eventID,
			WorkerID:    workerID,
			Kind:        worker.SafetyEventInspection,
			Severity:    worker.SafetySeverityModerate,
			Status:      worker.SafetyEventStatusOpen,
			OccurredAt:  1_790_000_000,
			Description: "Level 1 inspection at the Seymour scale",
			Points:      3,
			Version:     4,
		},
		violation: &worker.WorkerSafetyViolation{
			ID:             pulid.MustNew("wsvi_"),
			SafetyEventID:  eventID,
			WorkerID:       workerID,
			Basic:          worker.BasicVehicleMaintenance,
			Code:           "393.47(e)",
			Description:    "Brake out of adjustment",
			SeverityWeight: 4,
			Version:        2,
		},
		recognition: &worker.WorkerRecognition{
			ID:       pulid.MustNew("wrec_"),
			WorkerID: workerID,
			Kind:     worker.RecognitionKindTenure,
			Title:    "Five years",
			Version:  1,
		},
	}
}

func (f *fakeSafety) GetEvent(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*worker.WorkerSafetyEvent, error) {
	copied := *f.event
	return &copied, nil
}

func (f *fakeSafety) PlanCreateEvent(
	_ context.Context,
	entity *worker.WorkerSafetyEvent,
	userID pulid.ID,
) (*worker.WorkerSafetyEvent, error) {
	planned := *entity
	planned.RecordedByID = userID
	return &planned, nil
}

func (f *fakeSafety) CreateEvent(
	_ context.Context,
	entity *worker.WorkerSafetyEvent,
	_ pulid.ID,
) (*worker.WorkerSafetyEvent, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.created = entity
	created := *entity
	created.ID = pulid.MustNew("wsev_")
	return &created, nil
}

func (f *fakeSafety) PlanUpdateEvent(
	_ context.Context,
	entity *worker.WorkerSafetyEvent,
) (*workersafetyservice.EventChange, error) {
	return &workersafetyservice.EventChange{Before: f.event, After: entity}, nil
}

func (f *fakeSafety) UpdateEvent(
	_ context.Context,
	entity *worker.WorkerSafetyEvent,
	_ pulid.ID,
) (*worker.WorkerSafetyEvent, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.updated = entity
	return entity, nil
}

func (f *fakeSafety) planMove(
	req *workersafetyservice.EventStatusRequest,
	status worker.SafetyEventStatus,
) *workersafetyservice.EventChange {
	after := *f.event
	after.Status = status
	after.Resolution = req.Resolution
	return &workersafetyservice.EventChange{Before: f.event, After: &after}
}

func (f *fakeSafety) move(
	req *workersafetyservice.EventStatusRequest,
) (*worker.WorkerSafetyEvent, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.moved = append(f.moved, req)
	return f.event, nil
}

func (f *fakeSafety) PlanCloseEvent(
	_ context.Context,
	req *workersafetyservice.EventStatusRequest,
) (*workersafetyservice.EventChange, error) {
	return f.planMove(req, worker.SafetyEventStatusClosed), nil
}

func (f *fakeSafety) CloseEvent(
	_ context.Context,
	req *workersafetyservice.EventStatusRequest,
) (*worker.WorkerSafetyEvent, error) {
	return f.move(req)
}

func (f *fakeSafety) PlanReviewEvent(
	_ context.Context,
	req *workersafetyservice.EventStatusRequest,
) (*workersafetyservice.EventChange, error) {
	return f.planMove(req, worker.SafetyEventStatusUnderReview), nil
}

func (f *fakeSafety) ReviewEvent(
	_ context.Context,
	req *workersafetyservice.EventStatusRequest,
) (*worker.WorkerSafetyEvent, error) {
	return f.move(req)
}

func (f *fakeSafety) PlanDeleteEvent(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*worker.WorkerSafetyEvent, error) {
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
	if err := f.guard.write(); err != nil {
		return err
	}
	f.deleted = append(f.deleted, id)
	return nil
}

func (f *fakeSafety) PlanGiveRecognition(
	_ context.Context,
	entity *worker.WorkerRecognition,
	_ pulid.ID,
) (*worker.WorkerRecognition, error) {
	return entity, nil
}

func (f *fakeSafety) GiveRecognition(
	_ context.Context,
	entity *worker.WorkerRecognition,
	_ pulid.ID,
) (*worker.WorkerRecognition, error) {
	if err := f.guard.write(); err != nil {
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
) (*worker.WorkerSafetyViolation, error) {
	return &worker.WorkerSafetyViolation{
		SafetyEventID:  req.SafetyEventID,
		WorkerID:       f.event.WorkerID,
		Basic:          worker.BasicVehicleMaintenance,
		Description:    req.Description,
		SeverityWeight: max(req.SeverityWeight, 1),
	}, nil
}

func (f *fakeSafety) RecordViolation(
	_ context.Context,
	req *workersafetyservice.RecordViolationRequest,
) (*worker.WorkerSafetyViolation, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.cited = req
	return f.violation, nil
}

func (f *fakeSafety) GetViolation(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*worker.WorkerSafetyViolation, error) {
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
) (*worker.WorkerSafetyViolation, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.corrected = req
	return f.violation, nil
}

func TestOpenWorkerSafetyEvent_TakesTheHouseScaleForPoints(t *testing.T) {
	t.Parallel()

	events := newFakeSafety()
	tool := newOpenWorkerSafetyEventTool(events)
	params := executeParams(map[string]any{
		paramWorkerID:      events.event.WorkerID.String(),
		paramEventKind:     "Accident",
		paramEventSeverity: "Major",
		wfParamOccurred:    "2026-09-20T14:30:00-05:00",
		fieldDescription:   "Rear-ended at a light in Joliet",
		paramPreventable:   true,
		paramFineAmount:    "250.00",
	})

	preview := previewWithoutWrites(t, events.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, agent.PreviewOperationCreate, previewChange(t, preview, 0).Operation)
	assert.Contains(t, preview.Summary, "major accident")

	require.NoError(t, tool.(serviceports.ToolValidator).Validate(t.Context(), params))
	result, err := tool.(serviceports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	require.NotNil(t, events.created)
	assert.Equal(t, worker.DefaultSafetyPoints(worker.SafetyEventAccident,
		worker.SafetySeverityMajor, true, worker.InspectionResultNone), events.created.Points)
	assert.True(t, events.created.FineAmount.Valid)
	assert.Equal(t, params.OrganizationID, events.created.OrganizationID)
	assert.Contains(t, result.IDs, paramSafetyEventID)

	policy := tool.Policy()
	assert.Equal(t, permission.ResourceWorkerSafetyEvent, policy.Resource)
	assert.Equal(t, permission.OpCreate, policy.Operation)

	for name, raw := range map[string]map[string]any{
		"no description": {
			paramWorkerID: events.event.WorkerID.String(), paramEventKind: "Accident",
			paramEventSeverity: "Major", wfParamOccurred: "2026-09-20T14:30:00-05:00",
		},
		"unknown kind": {
			paramWorkerID: events.event.WorkerID.String(), paramEventKind: "Fire",
			paramEventSeverity: "Major", wfParamOccurred: "2026-09-20T14:30:00-05:00",
			fieldDescription: "x",
		},
		"epoch seconds": {
			paramWorkerID: events.event.WorkerID.String(), paramEventKind: "Accident",
			paramEventSeverity: "Major", wfParamOccurred: float64(1_790_000_000),
			fieldDescription: "x",
		},
	} {
		require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
			executeParams(raw)), name)
	}
}

func TestUpdateWorkerSafetyEvent_ChangesOnlyWhatIsNamed(t *testing.T) {
	t.Parallel()

	events := newFakeSafety()
	tool := newUpdateWorkerSafetyEventTool(events)
	params := executeParams(map[string]any{
		paramSafetyEventID: events.event.ID.String(),
		paramPoints:        float64(5),
	})

	preview := previewWithoutWrites(t, events.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.InDelta(t, 5, fieldByPath(t, previewChange(t, preview, 0), paramPoints).After, 0)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, events.updated)
	assert.Equal(t, int32(5), events.updated.Points)
	assert.Equal(t, events.event.Description, events.updated.Description)
	target, ok := tool.(serviceports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, permission.ResourceWorkerSafetyEvent, target.Resource)
}

func TestChangeWorkerSafetyEventStatus_ClosingNeedsAResolution(t *testing.T) {
	t.Parallel()

	events := newFakeSafety()
	tool := newChangeWorkerSafetyEventStatusTool(events)
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{
			paramSafetyEventID: events.event.ID.String(),
			paramEventMove:     "Close",
		})))

	params := executeParams(map[string]any{
		paramSafetyEventID:  events.event.ID.String(),
		paramEventMove:      "Close",
		paramResolutionText: "Brakes adjusted and re-inspected",
	})
	preview := previewWithoutWrites(t, events.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, string(worker.SafetyEventStatusClosed),
		fieldByPath(t, previewChange(t, preview, 0), fieldStatus).After)
	require.NoError(t, tool.Execute(t.Context(), params))
	require.Len(t, events.moved, 1)
	assert.Equal(t, permission.OpClose, tool.Policy().Operation)

	require.NoError(t, tool.Execute(t.Context(), executeParams(map[string]any{
		paramSafetyEventID: events.event.ID.String(),
		paramEventMove:     "Review",
	})))
	assert.Len(t, events.moved, 2)
}

func TestDeleteWorkerSafetyEvent_RefusesAClosedEvent(t *testing.T) {
	t.Parallel()

	events := newFakeSafety()
	tool := newDeleteWorkerSafetyEventTool(events)
	params := executeParams(map[string]any{paramSafetyEventID: events.event.ID.String()})

	preview := previewWithoutWrites(t, events.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, agent.PreviewOperationDelete, previewChange(t, preview, 0).Operation)
	assert.Equal(t, agent.TierPropose, tool.Policy().MaxTier)
	assert.False(t, tool.Policy().Reversible)

	events.event.Status = worker.SafetyEventStatusClosed
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(), params))
	assert.Empty(t, events.deleted)
}

func TestGiveWorkerRecognition_TellsTheDriver(t *testing.T) {
	t.Parallel()

	events := newFakeSafety()
	tool := newGiveWorkerRecognitionTool(events)
	params := executeParams(map[string]any{
		paramWorkerID:        events.event.WorkerID.String(),
		paramRecognitionKind: "SafetyMilestone",
		paramTitle:           "A million safe miles",
		fieldMessage:         "Thank you for every one of them.",
	})

	preview := previewWithoutWrites(t, events.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
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
	params := executeParams(map[string]any{
		paramSafetyEventID:  events.event.ID.String(),
		fieldDescription:    "Inoperative marker lamp",
		paramSeverityWeight: float64(2),
	})
	preview := previewWithoutWrites(t, events.guard, func() (*agent.ToolPreview, error) {
		return record.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "severity weight 2")
	require.NoError(t, record.Execute(t.Context(), params))
	require.NotNil(t, events.cited)
	assert.Equal(t, int16(2), events.cited.SeverityWeight)

	update := newUpdateSafetyViolationTool(events)
	require.NoError(t, update.Execute(t.Context(), executeParams(map[string]any{
		paramViolationID:  events.violation.ID.String(),
		paramOutOfService: true,
	})))
	require.NotNil(t, events.corrected)
	assert.True(t, events.corrected.OutOfService)
	assert.Equal(t, events.violation.Description, events.corrected.Description)
	assert.Equal(t, events.violation.Code, events.corrected.Code)

	require.Error(t, record.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{
			paramSafetyEventID:  events.event.ID.String(),
			fieldDescription:    "x",
			paramSeverityWeight: float64(11),
		})), "a severity weight runs 1 to 10")
}
