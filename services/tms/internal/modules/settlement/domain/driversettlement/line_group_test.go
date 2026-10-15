package driversettlement_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ref(raw string) *pulid.ID {
	value := pulid.ID(raw)
	return &value
}

func dec(raw string) decimal.Decimal {
	return decimal.RequireFromString(raw)
}

func earning(shipment, description string, miles, rate string, amount int64) *driversettlement.SettlementLine {
	return &driversettlement.SettlementLine{
		Category:      driversettlement.LineCategoryEarning,
		ComponentKind: driverpay.ComponentKindLinehaul,
		Method:        driverpay.CalcMethodPerLoadedMile,
		Description:   description,
		Quantity:      dec(miles),
		Rate:          dec(rate),
		AmountMinor:   amount,
		ShipmentID:    ref(shipment),
		PayEventID:    ref("dpe_" + shipment),
	}
}

func TestLineGroupsSumsLinesPaidTheSameWayAcrossLoads(t *testing.T) {
	t.Parallel()

	groups := driversettlement.LineGroups([]*driversettlement.SettlementLine{
		earning("shp_1", "Linehaul S-1", "760", "0.62", 47120),
		earning("shp_2", "Linehaul S-2", "395", "0.62", 24490),
		earning("shp_2", "Linehaul S-2 leg 2", "985", "0.62", 61070),
	})

	require.Len(t, groups, 1)
	assert.Equal(t, driversettlement.LineGroup{
		Category:      driversettlement.LineCategoryEarning,
		ComponentKind: driverpay.ComponentKindLinehaul,
		Method:        driverpay.CalcMethodPerLoadedMile,
		Rate:          decimal.NewNullDecimal(dec("0.62")),
		Quantity:      dec("2140"),
		AmountMinor:   132680,
		LineCount:     3,
		ShipmentCount: 2,
	}, groups[0])
}

func TestLineGroupsKeepsARateAndADescriptionOnlyWhenEveryLineSharesThem(t *testing.T) {
	t.Parallel()

	groups := driversettlement.LineGroups([]*driversettlement.SettlementLine{
		earning("shp_1", "Linehaul", "100", "0.62", 6200),
		earning("shp_2", "Linehaul", "100", "0.58", 5800),
	})

	require.Len(t, groups, 1)
	assert.False(t, groups[0].Rate.Valid)
	assert.Equal(t, "Linehaul", groups[0].Description)
}

func TestLineGroupsKeepsEachDeductionAdvanceAndEscrowApart(t *testing.T) {
	t.Parallel()

	groups := driversettlement.LineGroups([]*driversettlement.SettlementLine{
		{
			Category:             driversettlement.LineCategoryDeduction,
			Description:          "Health insurance",
			AmountMinor:          -9600,
			RecurringDeductionID: ref("rded_med"),
		},
		{
			Category:             driversettlement.LineCategoryDeduction,
			Description:          "Phone plan",
			AmountMinor:          -3400,
			RecurringDeductionID: ref("rded_phone"),
		},
		{
			Category:    driversettlement.LineCategoryAdvanceRecovery,
			Description: "Advance ADV-0088",
			Rate:        dec("200"),
			AmountMinor: -20000,
			AdvanceID:   ref("padv_88"),
		},
		{
			Category:        driversettlement.LineCategoryEscrowContribution,
			Description:     "Maintenance escrow",
			AmountMinor:     -10000,
			EscrowAccountID: ref("esc_1"),
		},
	})

	require.Len(t, groups, 4)
	assert.Equal(t, "Health insurance", groups[0].Description)
	assert.Equal(t, ref("rded_med"), groups[0].RecurringDeductionID)
	assert.Equal(t, "Phone plan", groups[1].Description)
	assert.Equal(t, ref("padv_88"), groups[2].AdvanceID)
	assert.Equal(t, ref("esc_1"), groups[3].EscrowAccountID)
	assert.Equal(t, int64(-10000), groups[3].AmountMinor)
}

func TestLineGroupsMarksPayAddedByHandAndKeepsEachSuchLineToItself(t *testing.T) {
	t.Parallel()

	groups := driversettlement.LineGroups([]*driversettlement.SettlementLine{
		{
			Category:    driversettlement.LineCategoryAdjustment,
			Description: "Layover - Detroit 6/12",
			AmountMinor: 15000,
		},
		{
			Category:    driversettlement.LineCategoryAdjustment,
			Description: "Breakdown pay",
			AmountMinor: 7500,
		},
		{
			Category:           driversettlement.LineCategoryAdjustment,
			Description:        "Safety bonus",
			AmountMinor:        5000,
			RecurringEarningID: ref("rern_safety"),
		},
	})

	require.Len(t, groups, 3)
	assert.True(t, groups[0].Manual)
	assert.Equal(t, "Layover - Detroit 6/12", groups[0].Description)
	assert.True(t, groups[1].Manual)
	assert.Equal(t, int64(7500), groups[1].AmountMinor)
	assert.False(t, groups[2].Manual)
}

func TestLineGroupsOrdersEarningsBeforeReimbursementsBeforeDeductions(t *testing.T) {
	t.Parallel()

	groups := driversettlement.LineGroups([]*driversettlement.SettlementLine{
		{Category: driversettlement.LineCategoryDeduction, Description: "Fuel card", AmountMinor: -100},
		{Category: driversettlement.LineCategoryReimbursement, Description: "Tolls", AmountMinor: 100},
		{Category: driversettlement.LineCategoryCarryForward, Description: "Carried in", AmountMinor: -50},
		nil,
		earning("shp_1", "Linehaul", "10", "1", 1000),
		{Category: driversettlement.LineCategoryGuaranteeTopUp, Description: "Weekly minimum", AmountMinor: 900},
	})

	categories := make([]driversettlement.LineCategory, 0, len(groups))
	for _, group := range groups {
		categories = append(categories, group.Category)
	}
	assert.Equal(t, []driversettlement.LineCategory{
		driversettlement.LineCategoryEarning,
		driversettlement.LineCategoryGuaranteeTopUp,
		driversettlement.LineCategoryReimbursement,
		driversettlement.LineCategoryCarryForward,
		driversettlement.LineCategoryDeduction,
	}, categories)
}

func TestLoadLinesCarriesEveryLineToTheLoadItBelongsTo(t *testing.T) {
	t.Parallel()

	lines := driversettlement.LoadLines([]*driversettlement.SettlementLine{
		earning("shp_1", "Linehaul", "10", "1", 1000),
		nil,
		{Category: driversettlement.LineCategoryDeduction, AmountMinor: -100},
	})

	require.Len(t, lines, 2)
	assert.Equal(t, ref("shp_1"), lines[0].ShipmentID)
	assert.Equal(t, int64(1000), lines[0].AmountMinor)
	assert.Nil(t, lines[1].ShipmentID)
}
