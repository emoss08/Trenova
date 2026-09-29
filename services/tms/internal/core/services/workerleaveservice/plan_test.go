package workerleaveservice_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/services/workerleaveservice"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanOpenCase_FillsTheCaseWithoutFilingIt(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	planned, err := h.svc.PlanOpenCase(t.Context(), &worker.WorkerLeaveCase{
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		WorkerID:       h.workerID,
		StartsAt:       at(2026, time.March, 2),
		Reason:         "  Surgery  ",
	}, h.userID)
	require.NoError(t, err)
	assert.Equal(t, worker.LeaveCasePending, planned.Status)
	assert.Equal(t, worker.LeaveTypeFMLA, planned.LeaveType)
	assert.Equal(t, "Surgery", planned.Reason)
	assert.Equal(t, h.userID, planned.RecordedByID)
	assert.Empty(t, h.repo.cases)
}

func TestPlanCaseChanges_LeaveTheCaseAsItIs(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	open := h.openCase(t)

	notes := "Returned paperwork by fax"
	updated, err := h.svc.PlanUpdateCase(t.Context(), &workerleaveservice.UpdateCaseRequest{
		TenantInfo: h.tenant,
		CaseID:     open.ID,
		Notes:      &notes,
	})
	require.NoError(t, err)
	assert.Equal(t, notes, updated.After.Notes)
	assert.Empty(t, h.repo.cases[open.ID].Notes)

	_, err = h.svc.PlanCloseCase(t.Context(), h.tenant, open.ID)
	require.Error(t, err, "a pending case is decided before it closes")

	cert, err := h.svc.PlanRequestCertification(t.Context(),
		&workerleaveservice.RequestCertificationRequest{TenantInfo: h.tenant, CaseID: open.ID})
	require.NoError(t, err)
	assert.Equal(t, worker.CertificationRequested, cert.After.CertificationStatus)
	require.NotNil(t, cert.After.CertificationDueAt)
	assert.Equal(t, worker.CertificationNotRequired, h.repo.cases[open.ID].CertificationStatus)

	approved := h.approved(t, true)
	closing, err := h.svc.PlanCloseCase(t.Context(), h.tenant, approved.ID)
	require.NoError(t, err)
	assert.Equal(t, worker.LeaveCaseClosed, closing.After.Status)
	assert.NotNil(t, closing.After.ClosedAt)
	assert.Equal(t, worker.LeaveCaseApproved, h.repo.cases[approved.ID].Status)
}

func TestPlanDays_CountAsTheCaseCountsNow(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	approved := h.approved(t, true)

	day, err := h.svc.PlanRecordDay(t.Context(), &workerleaveservice.RecordDayRequest{
		TenantInfo: h.tenant,
		CaseID:     approved.ID,
		UsedOn:     at(2026, time.March, 3),
		Hours:      decimal.NewFromInt(8),
	})
	require.NoError(t, err)
	assert.True(t, day.CountsAgainstEntitlement)
	assert.Equal(t, approved.WorkerID, day.WorkerID)
	assert.Empty(t, h.repo.entries)

	created, err := h.svc.RecordDay(t.Context(), &workerleaveservice.RecordDayRequest{
		TenantInfo: h.tenant,
		CaseID:     approved.ID,
		UsedOn:     at(2026, time.March, 3),
		Hours:      decimal.NewFromInt(8),
	})
	require.NoError(t, err)

	counts := false
	change, err := h.svc.PlanUpdateDay(t.Context(), &workerleaveservice.UpdateDayRequest{
		TenantInfo: h.tenant,
		EntryID:    created.ID,
		Counts:     &counts,
	})
	require.NoError(t, err)
	assert.False(t, change.After.CountsAgainstEntitlement)
	assert.True(t, h.repo.entries[created.ID].CountsAgainstEntitlement)

	gone, err := h.svc.PlanDeleteDay(t.Context(), h.tenant, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, gone.ID)
	assert.Len(t, h.repo.entries, 1)
}
