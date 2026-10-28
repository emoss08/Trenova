package shipmentrepository

import (
	"testing"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/quickfilter"
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
	spec quickfilter.QuickFilterSpec,
) (string, []any) {
	t.Helper()

	cond, err := QuickFilterCondition(basis, spec)
	require.NoError(t, err)

	return renderShipments(t, cond)
}

func TestQuickFilterCondition_StageFiltersUseStageRank(t *testing.T) {
	t.Parallel()

	basis := quickFilterTestBasis(t)
	cases := map[quickfilter.QuickFilter]shipment.Stage{
		quickfilter.QuickFilterLate:      shipment.StageLate,
		quickfilter.QuickFilterUncovered: shipment.StageNeedsCoverage,
		quickfilter.QuickFilterMoving:    shipment.StageMoving,
	}
	for filter, stage := range cases {
		sql, args := renderQuickFilter(t, basis, quickfilter.Quick(filter))
		assert.Contains(t, sql, `s."stage_rank" = $3`, filter)
		assert.Equal(t, stage.Rank(), args[2], filter)
	}
}

func TestQuickFilterCondition_DeliveringTodayUsesLocalDay(t *testing.T) {
	t.Parallel()

	basis := quickFilterTestBasis(t)
	sql, args := renderQuickFilter(
		t,
		basis,
		quickfilter.Quick(quickfilter.QuickFilterDeliveringToday),
	)

	day := LocalDayOf(basis.Now, basis.Location)
	assert.Equal(t, int64(24*60*60), day.End-day.Start)
	assert.Contains(t, args, int64(1791262800))
	assert.Contains(t, args, int64(1791349200))
	assert.Contains(t, sql, "int8range(")
	assert.Contains(t, sql, `coalesce(qf_stp."actual_arrival", qf_stp."scheduled_window_start")`)
	assert.Contains(t, sql, `ORDER BY qf_sm."sequence" DESC, qf_stp."sequence" DESC LIMIT 1`)
	assert.Contains(t, sql, `qf_sm."shipment_id" = s."id"`)
	assert.Contains(t, sql, `qf_stp."type" IN ('Delivery', 'SplitDelivery')`)
}

func TestQuickFilterCondition_DeliveryHourIsOneLocalHour(t *testing.T) {
	t.Parallel()

	basis := quickFilterTestBasis(t)
	_, args := renderQuickFilter(t, basis, quickfilter.DeliveryHourFilter(14))

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
	soon, soonArgs := renderQuickFilter(t, basis, quickfilter.PickupWindowFilter(0, &end))
	assert.Contains(t, soon, "::bigint, ")
	assert.Contains(t, soon, `ORDER BY qf_sm."sequence" ASC, qf_stp."sequence" ASC LIMIT 1`)
	assert.Contains(t, soon, `qf_stp."type" IN ('Pickup', 'SplitPickup')`)
	upper := now + 120*60
	assert.Contains(t, soonArgs, upper)

	_, laterArgs := renderQuickFilter(t, basis, quickfilter.PickupWindowFilter(360, nil))
	lower := now + 360*60
	assert.Contains(t, laterArgs, lower)
}

func TestQuickFilterCondition_LowMarginMatchesEstimator(t *testing.T) {
	t.Parallel()

	basis := quickFilterTestBasis(t)
	sql, args := renderQuickFilter(t, basis, quickfilter.Quick(quickfilter.QuickFilterLowMargin))
	assert.Contains(t, sql, `HAVING coalesce(sum(qf_mi."distance"), $3) > $4 AND `)
	assert.Contains(t, sql, `(s."total_charge_amount" <= $5 OR round(`)
	assert.Contains(
		t,
		sql,
		`round(($6::numeric * CAST(coalesce(sum(qf_mi."distance"), $7) AS numeric)), $8)`,
	)
	assert.Contains(t, sql, `* $9::numeric) / s."total_charge_amount"), $10) < $11)`)
	assert.Equal(t, []any{0, float64(0), 0}, args[2:5])
	assert.Equal(t, []any{0, int64(2), 100, int64(2)}, args[6:10])
	assert.Contains(t, args, decimal.RequireFromString("1.85"))
	assert.Contains(t, args, decimal.NewFromInt(12))

	basis.Margin.IncludeDeadheadMiles = false
	loaded, _ := renderQuickFilter(t, basis, quickfilter.Quick(quickfilter.QuickFilterLowMargin))
	assert.Contains(t, loaded, `coalesce(sum(qf_mi."distance") FILTER (WHERE qf_mi."loaded"), $7)`)
}

func TestQuickFilterCondition_DetentionFollowsTheEngine(t *testing.T) {
	t.Parallel()

	basis := quickFilterTestBasis(t)
	dwell, dwellArgs := renderQuickFilter(
		t,
		basis,
		quickfilter.Quick(quickfilter.QuickFilterDetention),
	)
	assert.Contains(t, dwell, `coalesce(qf_dw."actual_arrival", $3) > $4`)
	assert.Contains(t, dwell, `coalesce(qf_dw."actual_departure", $5) = $6`)
	assert.Contains(t, dwell, `coalesce(qf_dw."count_detention_override", $7)`)
	assert.Equal(t, []any{0, int64(0), 0, int64(0), true}, dwellArgs[2:7])
	assert.Contains(t, dwellArgs, basis.Now.Unix()-30*60)

	basis.Detention.UsePolicyEngine = true
	engine, engineArgs := renderQuickFilter(
		t,
		basis,
		quickfilter.Quick(quickfilter.QuickFilterDetention),
	)
	assert.Contains(t, engine, `"detention_occurrences" qf_do`)
	assert.Contains(t, engine, `qf_do."shipment_id" = s."id"`)
	assert.Contains(t, engineArgs, basis.Now.Unix())
}

func TestQuickFilterCondition_ReeferAndReadyToBill(t *testing.T) {
	t.Parallel()

	basis := quickFilterTestBasis(t)
	reefer, _ := renderQuickFilter(t, basis, quickfilter.Quick(quickfilter.QuickFilterReefer))
	assert.Contains(t, reefer, `s."temperature_min" IS NOT NULL`)
	assert.Contains(
		t,
		reefer,
		`qf_mpf."equipment_type_ids" @> jsonb_build_array(s."trailer_type_id")`,
	)

	bill, _ := renderQuickFilter(t, basis, quickfilter.Quick(quickfilter.QuickFilterReadyToBill))
	assert.Contains(t, bill, `s."billing_transfer_status" IS NULL`)
	assert.Contains(t, bill, `s."status" IN ('Completed', 'ReadyToInvoice')`)
}

func TestQuickFilterCondition_RequiresBasis(t *testing.T) {
	t.Parallel()

	_, err := QuickFilterCondition(nil, quickfilter.Quick(quickfilter.QuickFilterLate))
	require.ErrorIs(t, err, ErrQuickFilterBasisMissing)

	basis := quickFilterTestBasis(t)
	basis.Margin = nil
	_, err = QuickFilterCondition(basis, quickfilter.Quick(quickfilter.QuickFilterLowMargin))
	require.ErrorIs(t, err, ErrQuickFilterMarginMissing)

	_, err = QuickFilterCondition(basis, quickfilter.QuickFilterSpec{
		Filter: quickfilter.QuickFilterDeliveryHour,
	})
	require.ErrorIs(t, err, ErrQuickFilterParameterMissing)
}

func TestAggregateScope_AndsEveryQuickFilter(t *testing.T) {
	t.Parallel()

	q, err := AggregateScope(
		&pagination.QueryOptions{},
		&shipmentports.ShipmentOptions{
			IncludeCustomer: true,
			QuickFilters: []quickfilter.QuickFilterSpec{
				quickfilter.Quick(quickfilter.QuickFilterLate),
				quickfilter.Quick(quickfilter.QuickFilterReefer),
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
