package shipmentrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyShipmentOptionFilters_ActivityWindowAddsOverlapPredicate(t *testing.T) {
	t.Parallel()

	repo, _ := newCancelTestRepository(t)
	dba := repo.db.DB()

	q := dba.NewSelect().Model((*shipment.Shipment)(nil))
	q = applyShipmentOptionFilters(q, dba, repositories.ShipmentOptions{
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
		repositories.ShipmentOptions{},
	)

	assert.Equal(t, base.String(), filtered.String())
}

func TestApplyShipmentOptionFilters_HalfOpenWindowIgnored(t *testing.T) {
	t.Parallel()

	require.False(t, repositories.ShipmentOptions{ActivityWindowStart: 100}.HasActivityWindow())
	require.False(t, repositories.ShipmentOptions{ActivityWindowEnd: 100}.HasActivityWindow())
	require.True(t, repositories.ShipmentOptions{
		ActivityWindowStart: 100,
		ActivityWindowEnd:   200,
	}.HasActivityWindow())
}
