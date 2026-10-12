package shipmentrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyShipmentOptionFilters_ActivityWindowAddsOverlapPredicate(t *testing.T) {
	t.Parallel()

	repo, _ := newCancelTestRepository(t)
	dba := repo.db.DB()

	q := dba.NewSelect().Model((*shipment.Shipment)(nil))
	q = applyShipmentOptionFilters(q, dba, shipmentports.ShipmentOptions{
		ActivityWindowStart: 1_700_000_000,
		ActivityWindowEnd:   1_700_086_400,
	})

	sql := q.String()
	assert.Contains(t, sql, "EXISTS (SELECT 1")
	assert.Contains(t, sql, `"shipment_moves" AS "sm_aw"`)
	assert.Contains(t, sql, `"stops" AS "stp_aw"`)
	assert.Contains(t, sql, "sm_aw.shipment_id = sp.id")
	assert.Contains(t, sql, "stp_aw.scheduled_window_start > 0")
	assert.Contains(t, sql, "stp_aw.scheduled_window_start <= 1700086400")
	assert.Contains(
		t,
		sql,
		"COALESCE(stp_aw.scheduled_window_end, stp_aw.scheduled_window_start) >= 1700000000",
	)
	assert.Contains(t, sql, "sm_aw.status != 'Canceled'")
}

func TestApplyShipmentOptionFilters_NoWindowLeavesQueryUntouched(t *testing.T) {
	t.Parallel()

	repo, _ := newCancelTestRepository(t)
	dba := repo.db.DB()

	base := dba.NewSelect().Model((*shipment.Shipment)(nil))
	filtered := applyShipmentOptionFilters(
		dba.NewSelect().Model((*shipment.Shipment)(nil)),
		dba,
		shipmentports.ShipmentOptions{},
	)

	assert.Equal(t, base.String(), filtered.String())
}

func TestApplyShipmentOptionFilters_HalfOpenWindowIgnored(t *testing.T) {
	t.Parallel()

	require.False(t, (&shipmentports.ShipmentOptions{ActivityWindowStart: 100}).HasActivityWindow())
	require.False(t, (&shipmentports.ShipmentOptions{ActivityWindowEnd: 100}).HasActivityWindow())
	require.True(t, (&shipmentports.ShipmentOptions{
		ActivityWindowStart: 100,
		ActivityWindowEnd:   200,
	}).HasActivityWindow())
}

func TestApplyShipmentOptionFilters_TractorIDsMatchLiveAssignments(t *testing.T) {
	t.Parallel()

	repo, _ := newCancelTestRepository(t)
	dba := repo.db.DB()
	tractorA := pulid.MustNew("tr_")
	tractorB := pulid.MustNew("tr_")

	q := dba.NewSelect().Model((*shipment.Shipment)(nil))
	q = applyShipmentOptionFilters(q, dba, shipmentports.ShipmentOptions{
		TractorIDs: []pulid.ID{tractorA, tractorB},
	})

	sql := q.String()
	assert.Contains(t, sql, "EXISTS (SELECT 1")
	assert.Contains(t, sql, `"shipment_moves" AS "sm_wrk"`)
	assert.Contains(t, sql, `JOIN "assignments" AS "a_wrk"`)
	assert.Contains(t, sql, "a_wrk.archived_at IS NULL")
	assert.Contains(t, sql, "a_wrk.status != 'Canceled'")
	assert.Contains(t, sql, "sm_wrk.status != 'Canceled'")
	assert.Contains(t, sql, "sm_wrk.shipment_id = sp.id")
	assert.Contains(
		t,
		sql,
		"a_wrk.tractor_id IN ('"+tractorA.String()+"', '"+tractorB.String()+"')",
	)
	assert.NotContains(t, sql, "primary_worker_id")
}

func TestApplyShipmentOptionFilters_WorkerIDsStillMatchEitherSeat(t *testing.T) {
	t.Parallel()

	repo, _ := newCancelTestRepository(t)
	dba := repo.db.DB()
	workerID := pulid.MustNew("wrk_")

	q := dba.NewSelect().Model((*shipment.Shipment)(nil))
	q = applyShipmentOptionFilters(q, dba, shipmentports.ShipmentOptions{
		WorkerIDs: []pulid.ID{workerID},
	})

	sql := q.String()
	assert.Contains(t, sql, "a_wrk.archived_at IS NULL")
	assert.Contains(
		t,
		sql,
		"((a_wrk.primary_worker_id IN ('"+workerID.String()+"')) OR "+
			"(a_wrk.secondary_worker_id IN ('"+workerID.String()+"')))",
	)
	assert.NotContains(t, sql, "tractor_id")
}

func TestApplyShipmentOptionFilters_TrailerIDsMatchLiveAssignments(t *testing.T) {
	t.Parallel()

	repo, _ := newCancelTestRepository(t)
	dba := repo.db.DB()
	trailerID := pulid.MustNew("trl_")

	q := dba.NewSelect().Model((*shipment.Shipment)(nil))
	q = applyShipmentOptionFilters(q, dba, shipmentports.ShipmentOptions{
		TrailerIDs: []pulid.ID{trailerID},
	})

	sql := q.String()
	assert.Contains(t, sql, `JOIN "assignments" AS "a_wrk"`)
	assert.Contains(t, sql, "a_wrk.archived_at IS NULL")
	assert.Contains(t, sql, "a_wrk.trailer_id IN ('"+trailerID.String()+"')")
	assert.NotContains(t, sql, "tractor_id")
}
