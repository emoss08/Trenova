package shipmentrepository

import (
	"testing"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func quickFilterTestBasis(t *testing.T) *shipmentports.ShipmentQuickFilterBasis {
	t.Helper()

	loc, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	return &shipmentports.ShipmentQuickFilterBasis{
		Now:      time.Date(2026, 10, 6, 15, 30, 0, 0, loc),
		Location: loc,
		Margin: &shipmentports.ShipmentMarginBasis{
			CostPerMile:          decimal.RequireFromString("1.85"),
			IncludeDeadheadMiles: true,
			TargetMarginPercent:  decimal.NewFromInt(12),
		},
		Detention: &shipmentports.ShipmentDetentionBasis{ThresholdMinutes: 30},
	}
}

func renderShipments(t *testing.T, preds ...keel.Pred) (string, []any) {
	t.Helper()

	sp := shipmentdb.Shipments
	return keeltest.Render(t, sp.Pick(sp.ID).Where(preds...).Limit(1))
}

func renderQuickFilter(
	t *testing.T,
	basis *shipmentports.ShipmentQuickFilterBasis,
	spec shipment.QuickFilterSpec,
) (string, []any) {
	t.Helper()

	cond, err := QuickFilterCondition(basis, spec)
	require.NoError(t, err)

	return renderShipments(t, cond)
}

func TestQuickFilterCondition_StageFiltersUseStageRank(t *testing.T) {
	t.Parallel()

	basis := quickFilterTestBasis(t)
	cases := map[shipment.QuickFilter]shipment.Stage{
		shipment.QuickFilterLate:      shipment.StageLate,
		shipment.QuickFilterUncovered: shipment.StageNeedsCoverage,
		shipment.QuickFilterMoving:    shipment.StageMoving,
	}
	for filter, stage := range cases {
		sql, args := renderQuickFilter(t, basis, shipment.Quick(filter))
		assert.Contains(t, sql, `s."stage_rank" = $3`, filter)
		assert.Equal(t, stage.Rank(), args[2], filter)
	}
}

func TestQuickFilterCondition_DeliveringTodayUsesLocalDay(t *testing.T) {
	t.Parallel()

	basis := quickFilterTestBasis(t)
	sql, args := renderQuickFilter(t, basis, shipment.Quick(shipment.QuickFilterDeliveringToday))

	day := LocalDayOf(basis.Now, basis.Location)
	assert.Equal(t, int64(24*60*60), day.End-day.Start)
	assert.Contains(t, args, int64(1791262800))
	assert.Contains(t, args, int64(1791349200))
	assert.Contains(t, sql, "int8range(")
	assert.Contains(t, sql, `coalesce(qf_stp."actual_arrival", qf_stp."scheduled_window_start")`)
	assert.Contains(t, sql, `ORDER BY qf_sm."sequence" DESC, qf_stp."sequence" DESC LIMIT 1`)
	assert.Contains(t, sql, `qf_sm."shipment_id" = s."id"`)
	assert.Contains(t, args, []string{"Delivery", "SplitDelivery"})
}

func TestQuickFilterCondition_DeliveryHourIsOneLocalHour(t *testing.T) {
	t.Parallel()

	basis := quickFilterTestBasis(t)
	_, args := renderQuickFilter(t, basis, shipment.DeliveryHourFilter(14))

	start := LocalHourStart(basis.Now, basis.Location, 14)
	assert.Equal(t, time.Date(2026, 10, 6, 14, 0, 0, 0, basis.Location).Unix(), start)
	assert.Contains(t, args, int64(1791313200))
	assert.Contains(t, args, int64(1791316800))
}

func TestQuickFilterCondition_PickupWindowBounds(t *testing.T) {
	t.Parallel()

	basis := quickFilterTestBasis(t)
	now := basis.Now.Unix()

	end := 120
	soon, soonArgs := renderQuickFilter(t, basis, shipment.PickupWindowFilter(0, &end))
	assert.Contains(t, soon, "::bigint, ")
	assert.Contains(t, soon, `ORDER BY qf_sm."sequence" ASC, qf_stp."sequence" ASC LIMIT 1`)
	assert.Contains(t, soonArgs, []string{"Pickup", "SplitPickup"})
	upper := now + 120*60
	assert.Contains(t, soonArgs, &upper)

	_, laterArgs := renderQuickFilter(t, basis, shipment.PickupWindowFilter(360, nil))
	lower := now + 360*60
	assert.Contains(t, laterArgs, &lower)
}

func TestQuickFilterCondition_LowMarginMatchesEstimator(t *testing.T) {
	t.Parallel()

	basis := quickFilterTestBasis(t)
	sql, args := renderQuickFilter(t, basis, shipment.Quick(shipment.QuickFilterLowMargin))
	assert.Contains(t, sql, `HAVING (coalesce(sum(qf_mi."distance"), $4) > $5)`)
	assert.Contains(t, sql, `(s."total_charge_amount" <= $6)`)
	assert.Contains(t, sql, `round(($7::numeric * CAST(coalesce(sum(qf_mi."distance"), $8) AS numeric)), $9)`)
	assert.Contains(t, sql, `* $10) / s."total_charge_amount"), $11) < $12)`)
	assert.Equal(t, []any{0, 0, 0}, args[3:6])
	assert.Equal(t, []any{0, 2, 100, 2}, args[7:11])
	assert.Contains(t, args, decimal.RequireFromString("1.85"))
	assert.Contains(t, args, decimal.NewFromInt(12))

	basis.Margin.IncludeDeadheadMiles = false
	loaded, _ := renderQuickFilter(t, basis, shipment.Quick(shipment.QuickFilterLowMargin))
	assert.Contains(t, loaded, `coalesce(sum(qf_mi."distance") FILTER (WHERE qf_mi."loaded"), $8)`)
}

func TestQuickFilterCondition_DetentionFollowsTheEngine(t *testing.T) {
	t.Parallel()

	basis := quickFilterTestBasis(t)
	dwell, dwellArgs := renderQuickFilter(t, basis, shipment.Quick(shipment.QuickFilterDetention))
	assert.Contains(t, dwell, `(coalesce(qf_dw."actual_arrival", $5) > $6)`)
	assert.Contains(t, dwell, `(coalesce(qf_dw."actual_departure", $7) = $8)`)
	assert.Contains(t, dwell, `coalesce(qf_dw."count_detention_override", $9)`)
	assert.Equal(t, []any{0, 0, 0, 0, true}, dwellArgs[4:9])
	assert.Contains(t, dwellArgs, basis.Now.Unix()-30*60)

	basis.Detention.UsePolicyEngine = true
	engine, engineArgs := renderQuickFilter(t, basis, shipment.Quick(shipment.QuickFilterDetention))
	assert.Contains(t, engine, `"detention_occurrences" qf_do`)
	assert.Contains(t, engine, `qf_do."shipment_id" = s."id"`)
	assert.Contains(t, engineArgs, basis.Now.Unix())
}

func TestQuickFilterCondition_ReeferAndReadyToBill(t *testing.T) {
	t.Parallel()

	basis := quickFilterTestBasis(t)
	reefer, _ := renderQuickFilter(t, basis, shipment.Quick(shipment.QuickFilterReefer))
	assert.Contains(t, reefer, `s."temperature_min" IS NOT NULL`)
	assert.Contains(t, reefer, `qf_mpf."equipment_type_ids" @> jsonb_build_array(s."trailer_type_id")`)

	bill, _ := renderQuickFilter(t, basis, shipment.Quick(shipment.QuickFilterReadyToBill))
	assert.Contains(t, bill, `s."billing_transfer_status" IS NULL`)
	assert.Contains(t, bill, `s."status" = ANY(`)
}

func TestQuickFilterCondition_RequiresBasis(t *testing.T) {
	t.Parallel()

	_, err := QuickFilterCondition(nil, shipment.Quick(shipment.QuickFilterLate))
	require.ErrorIs(t, err, ErrQuickFilterBasisMissing)

	basis := quickFilterTestBasis(t)
	basis.Margin = nil
	_, err = QuickFilterCondition(basis, shipment.Quick(shipment.QuickFilterLowMargin))
	require.ErrorIs(t, err, ErrQuickFilterMarginMissing)

	_, err = QuickFilterCondition(basis, shipment.QuickFilterSpec{
		Filter: shipment.QuickFilterDeliveryHour,
	})
	require.ErrorIs(t, err, ErrQuickFilterParameterMissing)
}

func TestAggregateScope_AndsEveryQuickFilter(t *testing.T) {
	t.Parallel()

	q, err := AggregateScope(
		&pagination.QueryOptions{},
		&shipmentports.ShipmentOptions{
			IncludeCustomer: true,
			QuickFilters: []shipment.QuickFilterSpec{
				shipment.Quick(shipment.QuickFilterLate),
				shipment.Quick(shipment.QuickFilterReefer),
			},
			QuickFilterBasis: quickFilterTestBasis(t),
		},
	)
	require.NoError(t, err)

	sql, _ := keeltest.Render(t, q.Limit(1))
	assert.Contains(t, sql, `s."stage_rank" = `)
	assert.Contains(t, sql, `s."temperature_min" IS NOT NULL`)
	assert.NotContains(t, sql, `"customers"`)
	assert.Contains(t, sql, `s."organization_id" = $`)

	_, err = AggregateScope(nil, &shipmentports.ShipmentOptions{})
	require.ErrorIs(t, err, ErrShipmentScopeFilterMissing)
}
