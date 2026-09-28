package workertrainingservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/services/workertrainingservice"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanAssign_OpensNothing(t *testing.T) {
	h := newHarness(t)
	course := h.course("ORIENT", true, nil)

	planned, err := h.svc.PlanAssign(t.Context(), &workertrainingservice.AssignRequest{
		TenantInfo: h.tenant, WorkerID: h.wrk.ID, CourseID: course.ID, UserID: h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, worker.TrainingStatusAssigned, planned.Status)
	assert.Equal(t, course.Name, planned.Course.Name)
	require.NotNil(t, planned.DueAt)
	assert.Empty(t, h.repo.records)

	gaps, err := h.svc.PlanAssignRequired(t.Context(), h.tenant, h.wrk.ID)
	require.NoError(t, err)
	require.Len(t, gaps, 1)
	assert.Equal(t, course.ID, gaps[0].ID)
	assert.Empty(t, h.repo.records)
}

func TestPlanBulkAssign_SortsEachPair(t *testing.T) {
	h := newHarness(t)
	open := h.course("OPEN", false, nil)
	fresh := h.course("FRESH", false, nil)
	_, err := h.svc.Assign(t.Context(), &workertrainingservice.AssignRequest{
		TenantInfo: h.tenant, WorkerID: h.wrk.ID, CourseID: open.ID, UserID: h.userID,
	})
	require.NoError(t, err)

	plan, err := h.svc.PlanBulkAssign(t.Context(), &workertrainingservice.BulkAssignRequest{
		TenantInfo: h.tenant,
		WorkerIDs:  []pulid.ID{h.wrk.ID},
		CourseIDs:  []pulid.ID{open.ID, fresh.ID},
		UserID:     h.userID,
	})
	require.NoError(t, err)
	require.Len(t, plan.Records, 1)
	assert.Equal(t, fresh.ID, plan.Records[0].CourseID)
	require.Len(t, plan.Skipped, 1)
	assert.Equal(t, open.ID, plan.Skipped[0].CourseID)
	assert.Len(t, h.repo.records, 1)

	_, err = h.svc.PlanBulkAssign(t.Context(), &workertrainingservice.BulkAssignRequest{
		TenantInfo: h.tenant,
	})
	require.Error(t, err)
}

func TestPlanCompleteWaiveCancelAttach(t *testing.T) {
	h := newHarness(t)
	scored := h.course("SCORED", true, func(c *worker.TrainingCourse) {
		c.PassingScore = decimal.NewNullDecimal(decimal.NewFromInt(80))
	})
	record, err := h.svc.Assign(t.Context(), &workertrainingservice.AssignRequest{
		TenantInfo: h.tenant, WorkerID: h.wrk.ID, CourseID: scored.ID, UserID: h.userID,
	})
	require.NoError(t, err)

	failed, err := h.svc.PlanComplete(t.Context(), &workertrainingservice.CompleteRequest{
		TenantInfo: h.tenant,
		ID:         record.ID,
		Score:      decimal.NewNullDecimal(decimal.NewFromInt(60)),
		UserID:     h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, worker.TrainingStatusFailed, failed.After.Status)
	require.NotNil(t, failed.Before)
	assert.Equal(t, worker.TrainingStatusAssigned, h.repo.records[0].Status)

	_, err = h.svc.PlanWaive(t.Context(), &workertrainingservice.StatusRequest{
		TenantInfo: h.tenant, ID: record.ID,
	})
	require.Error(t, err, "a waiver needs a reason")
	waived, err := h.svc.PlanWaive(t.Context(), &workertrainingservice.StatusRequest{
		TenantInfo: h.tenant, ID: record.ID, Reason: "Certificate from last employer",
	})
	require.NoError(t, err)
	assert.Equal(t, worker.TrainingStatusWaived, waived.After.Status)

	cancelled, err := h.svc.PlanCancel(t.Context(), &workertrainingservice.StatusRequest{
		TenantInfo: h.tenant, ID: record.ID, Reason: "Assigned twice",
	})
	require.NoError(t, err)
	assert.Equal(t, worker.TrainingStatusCancelled, cancelled.After.Status)
	assert.Equal(t, "Assigned twice", cancelled.After.Notes)
	assert.Equal(t, worker.TrainingStatusAssigned, h.repo.records[0].Status)
}
