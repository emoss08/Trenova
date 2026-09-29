package workerdrugalcoholservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/services/workerdrugalcoholservice"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanRecordTest_SchedulesWithoutFiling(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	test := h.newTest(worker.DOTTestPostAccident, worker.DOTSubstanceAlcohol)
	test.Status = ""
	test.Result = ""
	test.MROName = "Dr. Somebody"
	test.Reason = "Jackknife on I-80"
	planned, err := h.svc.PlanRecordTest(t.Context(), test, h.userID)
	require.NoError(t, err)
	assert.Equal(t, worker.DOTTestStatusScheduled, planned.Status)
	assert.Equal(t, worker.DOTResultPending, planned.Result)
	assert.Equal(t, h.userID, planned.OrderedByID)
	assert.Empty(t, planned.MROName, "an alcohol test has no medical review officer")
	assert.Empty(t, h.repo.tests, "a plan files nothing")
	assert.Empty(t, test.RecordedByID, "the caller's test is left as it was")
}

func TestPlanCancelTest_NeedsAReasonAndAnOpenTest(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	created, err := h.svc.RecordTest(t.Context(),
		h.newTest(worker.DOTTestRandom, worker.DOTSubstanceDrug), h.userID)
	require.NoError(t, err)

	_, err = h.svc.PlanCancelTest(t.Context(), h.tenant, created.ID, "  ")
	require.Error(t, err)

	change, err := h.svc.PlanCancelTest(t.Context(), h.tenant, created.ID, "Driver on leave")
	require.NoError(t, err)
	assert.Equal(t, worker.DOTTestStatusCancelled, change.After.Status)
	assert.Equal(t, worker.DOTResultCancelled, change.After.Result)
	assert.Contains(t, change.After.Notes, "Driver on leave")
	assert.Equal(t, worker.DOTTestStatusCollected, h.repo.tests[created.ID].Status)
}

func TestPlanRunDraw_NamesNobody(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	pool := newPool(h.tenant)
	h.repo.pools[pool.ID] = pool
	h.repo.defaults = pool
	for range 40 {
		h.repo.candidates = append(h.repo.candidates, pulid.MustNew("wrk_"))
	}

	draw, err := h.svc.PlanRunDraw(t.Context(), &workerdrugalcoholservice.RunDrawRequest{
		TenantInfo: h.tenant,
		UserID:     h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, int32(40), draw.PoolSize)
	assert.Equal(t, int32(5), draw.DrugTarget)
	assert.Equal(t, int32(5), draw.DrugSelected)
	assert.Empty(t, draw.Seed, "the seed is made when the round is drawn")
	assert.Empty(t, draw.Entries)
	assert.Equal(t, pool.Code, draw.Pool.Code)
	assert.Empty(t, h.repo.draws)

	_, err = h.svc.RunDraw(t.Context(), &workerdrugalcoholservice.RunDrawRequest{
		TenantInfo: h.tenant,
		UserID:     h.userID,
	})
	require.NoError(t, err)
	_, err = h.svc.PlanRunDraw(t.Context(), &workerdrugalcoholservice.RunDrawRequest{
		TenantInfo: h.tenant,
		UserID:     h.userID,
	})
	require.Error(t, err, "the plan refuses a second round as the draw does")
}

func TestPlanFinalizeAndCancelDraw(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	draw := &worker.DOTRandomDraw{
		ID:             pulid.MustNew("drdraw_"),
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		Status:         worker.RandomDrawStatusDraft,
		PeriodKey:      "2026-Q3",
	}
	h.repo.draws[draw.ID] = draw

	final, err := h.svc.PlanFinalizeDraw(t.Context(), h.tenant, draw.ID)
	require.NoError(t, err)
	assert.Equal(t, worker.RandomDrawStatusFinal, final.After.Status)
	assert.NotNil(t, final.After.FinalizedAt)
	assert.Equal(t, worker.RandomDrawStatusDraft, h.repo.draws[draw.ID].Status)

	_, err = h.svc.PlanCancelDraw(t.Context(), h.tenant, draw.ID, "")
	require.Error(t, err)
	cancelled, err := h.svc.PlanCancelDraw(t.Context(), h.tenant, draw.ID, "Wrong roster")
	require.NoError(t, err)
	assert.Equal(t, worker.RandomDrawStatusCancelled, cancelled.After.Status)

	draw.Status = worker.RandomDrawStatusFinal
	_, err = h.svc.PlanFinalizeDraw(t.Context(), h.tenant, draw.ID)
	require.Error(t, err)
}

func TestPlanUpdateDrawEntry_StampsTheNotice(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	entry := &worker.DOTRandomDrawEntry{
		ID:             pulid.MustNew("drde_"),
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		DrawID:         pulid.MustNew("drdraw_"),
		WorkerID:       h.workerID,
		Substance:      worker.DOTSubstanceDrug,
		Status:         worker.RandomEntrySelected,
	}
	h.repo.entries[entry.ID] = entry

	change, err := h.svc.PlanUpdateDrawEntry(t.Context(),
		&workerdrugalcoholservice.UpdateEntryRequest{
			TenantInfo: h.tenant,
			EntryID:    entry.ID,
			Status:     worker.RandomEntryNotified,
		})
	require.NoError(t, err)
	assert.Equal(t, worker.RandomEntryNotified, change.After.Status)
	assert.NotNil(t, change.After.NotifiedAt)
	assert.Nil(t, h.repo.entries[entry.ID].NotifiedAt)

	_, err = h.svc.PlanUpdateDrawEntry(t.Context(), &workerdrugalcoholservice.UpdateEntryRequest{
		TenantInfo: h.tenant,
		EntryID:    entry.ID,
		Status:     worker.RandomEntryCompleted,
	})
	require.Error(t, err)
}
