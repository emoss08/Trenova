package workersafetyservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/workersafetyservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (f *fakeRepo) GetRecognitionByID(
	_ context.Context,
	req *repositories.GetWorkerRecognitionByIDRequest,
) (*worker.WorkerRecognition, error) {
	for _, r := range f.recognitions {
		if r.ID == req.ID {
			copied := *r
			return &copied, nil
		}
	}
	return nil, errortypes.NewNotFoundError("WorkerRecognition not found")
}

func (f *fakeRepo) GetViolationByID(
	_ context.Context,
	req *repositories.GetWorkerSafetyViolationByIDRequest,
) (*worker.WorkerSafetyViolation, error) {
	for _, v := range f.violations {
		if v.ID == req.ID {
			copied := *v
			return &copied, nil
		}
	}
	return nil, errortypes.NewNotFoundError("WorkerSafetyViolation not found")
}

func TestPlanCreateEvent_FillsTheEventWithoutRecordingIt(t *testing.T) {
	h := newHarness(t)

	planned, err := h.svc.PlanCreateEvent(
		t.Context(),
		h.event(worker.SafetyEventInspection, func(e *worker.WorkerSafetyEvent) {
			e.InspectionResult = worker.InspectionResultOutOfService
		}),
		h.userID,
	)
	require.NoError(t, err)
	assert.Equal(t, worker.SafetyEventStatusOpen, planned.Status)
	assert.Equal(t, h.userID, planned.RecordedByID)
	assert.True(t, planned.OutOfService)
	assert.Empty(t, h.repo.events, "a plan records nothing")

	_, err = h.svc.PlanCreateEvent(t.Context(), h.event(worker.SafetyEventInspection, nil),
		h.userID)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr, "the plan refuses what the create refuses")
}

func TestPlanMoveEvent_ShowsTheMoveAndLeavesTheEvent(t *testing.T) {
	h := newHarness(t)
	created, err := h.svc.CreateEvent(t.Context(), h.event(worker.SafetyEventIncident, nil),
		h.userID)
	require.NoError(t, err)

	req := &workersafetyservice.EventStatusRequest{
		ID:         created.ID,
		TenantInfo: h.tenant,
		Resolution: "  Coached on backing  ",
		UserID:     h.userID,
	}
	change, err := h.svc.PlanCloseEvent(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, worker.SafetyEventStatusOpen, change.Before.Status)
	assert.Equal(t, worker.SafetyEventStatusClosed, change.After.Status)
	assert.Equal(t, "Coached on backing", change.After.Resolution)
	assert.Equal(t, h.userID, change.After.ClosedByID)
	assert.Equal(t, worker.SafetyEventStatusOpen, h.repo.events[0].Status)

	review, err := h.svc.PlanReviewEvent(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, worker.SafetyEventStatusUnderReview, review.After.Status)

	same, err := h.svc.PlanReopenEvent(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, same.Before.Status, same.After.Status, "reopening an open event changes nothing")

	_, err = h.svc.CloseEvent(t.Context(), req)
	require.NoError(t, err)
	_, err = h.svc.PlanDeleteEvent(t.Context(), h.tenant, created.ID)
	require.Error(t, err, "a closed event is history and is not deleted")
}

func TestPlanUpdateEvent_KeepsWhatTheUpdateKeeps(t *testing.T) {
	h := newHarness(t)
	created, err := h.svc.CreateEvent(t.Context(), h.event(worker.SafetyEventIncident, nil),
		h.userID)
	require.NoError(t, err)

	edit := *created
	edit.Description = "Clipped a mirror in the yard"
	edit.Status = worker.SafetyEventStatusClosed
	change, err := h.svc.PlanUpdateEvent(t.Context(), &edit)
	require.NoError(t, err)
	assert.Equal(t, "Clipped a mirror in the yard", change.After.Description)
	assert.Equal(t, worker.SafetyEventStatusOpen, change.After.Status,
		"an update never moves the status")
	assert.Equal(t, "Backed into a dock door at the consignee", h.repo.events[0].Description)
}

func TestPlanRecognitionAndViolations(t *testing.T) {
	h := newHarness(t)

	recognition, err := h.svc.PlanGiveRecognition(t.Context(), &worker.WorkerRecognition{
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		WorkerID:       h.wrk.ID,
		Kind:           worker.RecognitionKindSafetyMilestone,
		Title:          "  A million safe miles  ",
	}, h.userID)
	require.NoError(t, err)
	assert.Equal(t, "A million safe miles", recognition.Title)
	assert.Equal(t, h.userID, recognition.AwardedByID)
	assert.Positive(t, recognition.OccurredAt)
	assert.Empty(t, h.repo.recognitions)

	event, err := h.svc.CreateEvent(t.Context(), h.event(worker.SafetyEventInspection,
		func(e *worker.WorkerSafetyEvent) { e.InspectionResult = worker.InspectionResultFail }),
		h.userID)
	require.NoError(t, err)
	violation, err := h.svc.PlanRecordViolation(t.Context(),
		&workersafetyservice.RecordViolationRequest{
			TenantInfo:    h.tenant,
			SafetyEventID: event.ID,
			Description:   " Brake out of adjustment ",
		})
	require.NoError(t, err)
	assert.Equal(t, h.wrk.ID, violation.WorkerID, "the worker comes from the event")
	assert.Equal(t, int16(1), violation.SeverityWeight)
	assert.NotEmpty(t, violation.Basic)
	assert.Equal(t, "Brake out of adjustment", violation.Description)

	stored := *violation
	stored.ID = pulid.MustNew("wsvi_")
	stored.Version = 3
	h.repo.violations = append(h.repo.violations, &stored)
	change, err := h.svc.PlanUpdateViolation(t.Context(),
		&workersafetyservice.UpdateViolationRequest{
			TenantInfo:   h.tenant,
			ID:           stored.ID,
			Description:  "Brake chamber leaking",
			OutOfService: true,
		})
	require.NoError(t, err)
	assert.True(t, change.After.OutOfService)
	assert.False(t, change.Before.OutOfService)

	_, err = h.svc.PlanUpdateViolation(t.Context(), &workersafetyservice.UpdateViolationRequest{
		TenantInfo:  h.tenant,
		ID:          stored.ID,
		Description: "Brake chamber leaking",
		Version:     2,
	})
	require.Error(t, err, "a stale version is refused")

	gone, err := h.svc.PlanDeleteViolation(t.Context(), h.tenant, stored.ID)
	require.NoError(t, err)
	assert.Equal(t, stored.ID, gone.ID)
}
