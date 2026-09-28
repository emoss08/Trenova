package workerdqfservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/services/workerdqfservice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanVerifications_FileNothing(t *testing.T) {
	t.Parallel()
	h := newHarness(t, employedProfile())

	planned, err := h.svc.PlanRecordVerification(t.Context(), &worker.WorkerEmploymentVerification{
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		WorkerID:       h.workerID,
		EmployerName:   "  Prior Carrier  ",
	}, h.userID)
	require.NoError(t, err)
	assert.Equal(t, h.userID, planned.RequestedByID)
	assert.Empty(t, h.repo.verifications)

	created, err := h.svc.RecordVerification(t.Context(), &worker.WorkerEmploymentVerification{
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		WorkerID:       h.workerID,
		EmployerName:   "Prior Carrier",
	}, h.userID)
	require.NoError(t, err)

	_, err = h.svc.PlanRecordFollowUp(t.Context(), h.tenant, created.ID)
	require.Error(t, err, "only a request that went out can be chased")

	requested, err := h.svc.PlanMarkRequested(t.Context(), h.tenant, created.ID)
	require.NoError(t, err)
	assert.Equal(t, worker.VerificationRequested, requested.After.Status)
	assert.NotEqual(t, worker.VerificationRequested, h.repo.verifications[created.ID].Status)

	_, err = h.svc.MarkRequested(t.Context(), h.tenant, created.ID, h.userID)
	require.NoError(t, err)
	chase, err := h.svc.PlanRecordFollowUp(t.Context(), h.tenant, created.ID)
	require.NoError(t, err)
	assert.Equal(t, chase.Before.FollowUpCount+1, chase.After.FollowUpCount)

	findings := "Two preventable accidents"
	update, err := h.svc.PlanUpdateVerification(t.Context(),
		&workerdqfservice.UpdateVerificationRequest{
			TenantInfo:     h.tenant,
			VerificationID: created.ID,
			Findings:       &findings,
		})
	require.NoError(t, err)
	assert.Equal(t, findings, update.After.Findings)

	gone, err := h.svc.PlanDeleteVerification(t.Context(), h.tenant, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, gone.ID)
	assert.Len(t, h.repo.verifications, 1)
}
