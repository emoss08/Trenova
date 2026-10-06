package shipmentrepository

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func quickFilterTestBasis(t *testing.T) *repositories.ShipmentQuickFilterBasis {
	t.Helper()

	loc, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	return &repositories.ShipmentQuickFilterBasis{
		Now:      time.Date(2026, 10, 6, 15, 30, 0, 0, loc),
		Location: loc,
		Margin: &repositories.ShipmentMarginBasis{
			CostPerMile:          decimal.RequireFromString("1.85"),
			IncludeDeadheadMiles: true,
			TargetMarginPercent:  decimal.NewFromInt(12),
		},
		Detention: &repositories.ShipmentDetentionBasis{ThresholdMinutes: 30},
	}
}

func renderQuickFilter(
	t *testing.T,
	dba bun.IDB,
	basis *repositories.ShipmentQuickFilterBasis,
	spec shipment.QuickFilterSpec,
) string {
	t.Helper()

	cond, err := QuickFilterCondition(dba, basis, spec)
	require.NoError(t, err)

	return dba.NewSelect().
		Model((*shipment.Shipment)(nil)).
		ColumnExpr("1").
		Where("?", cond).
		String()
}

func TestQuickFilterCondition_StageFiltersUseStageRank(t *testing.T) {
	t.Parallel()

	repo, _ := newCancelTestRepository(t)
	dba := repo.db.DB()
	basis := quickFilterTestBasis(t)

	cases := map[shipment.QuickFilter]shipment.Stage{
		shipment.QuickFilterLate:      shipment.StageLate,
		shipment.QuickFilterUncovered: shipment.StageNeedsCoverage,
		shipment.QuickFilterMoving:    shipment.StageMoving,
	}
	for filter, stage := range cases {
		sql := renderQuickFilter(t, dba, basis, shipment.Quick(filter))
		assert.Contains(t, sql, "sp.stage_rank = ", filter)
		assert.Contains(t, sql, "sp.stage_rank = "+string(rune('0'+stage.Rank())), filter)
	}
}

func TestQuickFilterCondition_DeliveringTodayUsesLocalDay(t *testing.T) {
	t.Parallel()

	repo, _ := newCancelTestRepository(t)
	dba := repo.db.DB()
	basis := quickFilterTestBasis(t)

	sql := renderQuickFilter(t, dba, basis, shipment.Quick(shipment.QuickFilterDeliveringToday))

	day := LocalDayOf(basis.Now, basis.Location)
	assert.Equal(t, int64(24*60*60), day.End-day.Start)
	assert.Contains(t, sql, "int8range(1791262800, 1791349200)")
	assert.Contains(t, sql, "COALESCE(stp.actual_arrival, stp.scheduled_window_start)")
	assert.Contains(t, sql, "stp.type IN ('Delivery', 'SplitDelivery')")
	assert.Contains(t, sql, `ORDER BY "sm"."sequence" DESC, "stp"."sequence" DESC LIMIT 1`)
	assert.Contains(t, sql, "sp.status != 'Canceled'")
}

func TestQuickFilterCondition_DeliveryHourIsOneLocalHour(t *testing.T) {
	t.Parallel()

	repo, _ := newCancelTestRepository(t)
	dba := repo.db.DB()
	basis := quickFilterTestBasis(t)

	sql := renderQuickFilter(t, dba, basis, shipment.DeliveryHourFilter(14))

	start := LocalHourStart(basis.Now, basis.Location, 14)
	assert.Equal(t, time.Date(2026, 10, 6, 14, 0, 0, 0, basis.Location).Unix(), start)
	assert.Contains(t, sql, "int8range(1791313200, 1791316800)")
}

func TestQuickFilterCondition_PickupWindowBounds(t *testing.T) {
	t.Parallel()

	repo, _ := newCancelTestRepository(t)
	dba := repo.db.DB()
	basis := quickFilterTestBasis(t)
	now := basis.Now.Unix()

	end := 120
	soon := renderQuickFilter(t, dba, basis, shipment.PickupWindowFilter(0, &end))
	assert.Contains(t, soon, "int8range(NULL::bigint, ")
	assert.Contains(t, soon, "sp.stage_rank = 2")
	assert.Contains(t, soon, "stp.type IN ('Pickup', 'SplitPickup')")
	assert.Contains(t, soon, `ORDER BY "sm"."sequence" ASC, "stp"."sequence" ASC LIMIT 1`)

	later := renderQuickFilter(t, dba, basis, shipment.PickupWindowFilter(360, nil))
	assert.Contains(t, later, "::bigint, NULL::bigint)")
	assert.Contains(t, later, "int8range("+itoa(now+360*60)+"::bigint")
}

func TestQuickFilterCondition_LowMarginMatchesEstimator(t *testing.T) {
	t.Parallel()

	repo, _ := newCancelTestRepository(t)
	dba := repo.db.DB()
	basis := quickFilterTestBasis(t)

	sql := renderQuickFilter(t, dba, basis, shipment.Quick(shipment.QuickFilterLowMargin))
	assert.Contains(t, sql, "HAVING (COALESCE(SUM(sm.distance), 0) > 0)")
	assert.Contains(t, sql, "sp.total_charge_amount <= 0")
	assert.Contains(t, sql, "ROUND('1.85'::numeric * (COALESCE(SUM(sm.distance), 0))::numeric, 2)")
	assert.Contains(t, sql, "< '12'")

	basis.Margin.IncludeDeadheadMiles = false
	loaded := renderQuickFilter(t, dba, basis, shipment.Quick(shipment.QuickFilterLowMargin))
	assert.Contains(t, loaded, "COALESCE(SUM(sm.distance) FILTER (WHERE sm.loaded), 0)")
}

func TestQuickFilterCondition_DetentionFollowsTheEngine(t *testing.T) {
	t.Parallel()

	repo, _ := newCancelTestRepository(t)
	dba := repo.db.DB()
	basis := quickFilterTestBasis(t)

	dwell := renderQuickFilter(t, dba, basis, shipment.Quick(shipment.QuickFilterDetention))
	assert.Contains(t, dwell, "COALESCE(stp.actual_departure, 0) = 0")
	assert.Contains(t, dwell, "stp.actual_arrival <= "+itoa(basis.Now.Unix()-30*60))

	basis.Detention.UsePolicyEngine = true
	engine := renderQuickFilter(t, dba, basis, shipment.Quick(shipment.QuickFilterDetention))
	assert.Contains(t, engine, `"detention_occurrences" AS "dto"`)
	assert.Contains(t, engine, "dto.status = 'Accruing'")
}

func TestQuickFilterCondition_ReeferAndReadyToBill(t *testing.T) {
	t.Parallel()

	repo, _ := newCancelTestRepository(t)
	dba := repo.db.DB()
	basis := quickFilterTestBasis(t)

	reefer := renderQuickFilter(t, dba, basis, shipment.Quick(shipment.QuickFilterReefer))
	assert.Contains(t, reefer, "sp.temperature_min IS NOT NULL")
	assert.Contains(t, reefer, "mpf.equipment_class = 'Refrigerated'")
	assert.Contains(t, reefer, "mpf.equipment_type_ids @> jsonb_build_array(sp.trailer_type_id)")

	bill := renderQuickFilter(t, dba, basis, shipment.Quick(shipment.QuickFilterReadyToBill))
	assert.Contains(t, bill, "COALESCE(sp.billing_transfer_status, '') IN ('', 'SentBackToOps')")
}

func TestQuickFilterCondition_RequiresBasis(t *testing.T) {
	t.Parallel()

	repo, _ := newCancelTestRepository(t)
	dba := repo.db.DB()

	_, err := QuickFilterCondition(dba, nil, shipment.Quick(shipment.QuickFilterLate))
	require.ErrorIs(t, err, ErrQuickFilterBasisMissing)

	basis := quickFilterTestBasis(t)
	basis.Margin = nil
	_, err = QuickFilterCondition(dba, basis, shipment.Quick(shipment.QuickFilterLowMargin))
	require.ErrorIs(t, err, ErrQuickFilterMarginMissing)

	_, err = QuickFilterCondition(dba, basis, shipment.QuickFilterSpec{
		Filter: shipment.QuickFilterDeliveryHour,
	})
	require.ErrorIs(t, err, ErrQuickFilterParameterMissing)
}

func TestApplyAggregateScope_AndsEveryQuickFilter(t *testing.T) {
	t.Parallel()

	repo, _ := newCancelTestRepository(t)
	dba := repo.db.DB()

	q, err := ApplyAggregateScope(
		dba.NewSelect().Model((*shipment.Shipment)(nil)),
		dba,
		&pagination.QueryOptions{
			TenantInfo: pagination.TenantInfo{
				OrgID: pulid.MustNew("org_"),
				BuID:  pulid.MustNew("bu_"),
			},
		},
		repositories.ShipmentOptions{
			IncludeCustomer: true,
			QuickFilters: []shipment.QuickFilterSpec{
				shipment.Quick(shipment.QuickFilterLate),
				shipment.Quick(shipment.QuickFilterReefer),
			},
			QuickFilterBasis: quickFilterTestBasis(t),
		},
	)
	require.NoError(t, err)

	sql := q.String()
	assert.Contains(t, sql, "sp.stage_rank = 1")
	assert.Contains(t, sql, "sp.temperature_min IS NOT NULL")
	assert.NotContains(t, sql, `"customers"`)
	assert.Contains(t, sql, "sp.organization_id = ")

	_, err = ApplyAggregateScope(dba.NewSelect(), dba, nil, repositories.ShipmentOptions{})
	require.ErrorIs(t, err, ErrShipmentScopeFilterMissing)
}

func itoa(v int64) string {
	return decimal.NewFromInt(v).String()
}
