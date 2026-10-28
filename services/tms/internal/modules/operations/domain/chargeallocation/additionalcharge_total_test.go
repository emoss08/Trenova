package chargeallocation_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/chargeallocation"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdditionalChargeTotals_PricesEachChargeByItsMethod(t *testing.T) {
	t.Parallel()

	base := decimal.NewFromInt(2450)
	tests := []struct {
		name     string
		charge   *shipment.AdditionalCharge
		expected string
	}{
		{"nil charge", nil, "0"},
		{
			"flat once",
			&shipment.AdditionalCharge{
				Method: accessorialcharge.MethodFlat,
				Amount: decimal.NewFromInt(150),
				Unit:   1,
			},
			"150",
		},
		{
			"flat with zero unit still bills once",
			&shipment.AdditionalCharge{
				Method: accessorialcharge.MethodFlat,
				Amount: decimal.NewFromInt(150),
				Unit:   0,
			},
			"150",
		},
		{
			"flat repeated",
			&shipment.AdditionalCharge{
				Method: accessorialcharge.MethodFlat,
				Amount: decimal.NewFromInt(150),
				Unit:   2,
			},
			"300",
		},
		{
			"per unit",
			&shipment.AdditionalCharge{
				Method: accessorialcharge.MethodPerUnit,
				Amount: decimal.NewFromInt(75),
				Unit:   2,
			},
			"150",
		},
		{
			"per unit without units bills nothing",
			&shipment.AdditionalCharge{
				Method: accessorialcharge.MethodPerUnit,
				Amount: decimal.NewFromInt(75),
				Unit:   0,
			},
			"0",
		},
		{
			"percentage of base",
			&shipment.AdditionalCharge{
				Method: accessorialcharge.MethodPercentage,
				Amount: decimal.NewFromInt(10),
				Unit:   1,
			},
			"245",
		},
		{
			"unknown method",
			&shipment.AdditionalCharge{
				Method: accessorialcharge.Method("Tiered"),
				Amount: decimal.NewFromInt(10),
			},
			"0",
		},
	}

	charges := make([]*shipment.AdditionalCharge, 0, len(tests))
	sum := decimal.Zero
	for _, tt := range tests {
		charges = append(charges, tt.charge)
		sum = sum.Add(decimal.RequireFromString(tt.expected))
	}

	totals, err := chargeallocation.AdditionalChargeTotals(t.Context(), charges, base)
	require.NoError(t, err)
	require.Len(t, totals.Each, len(tests))

	for i, tt := range tests {
		assert.True(
			t,
			totals.Each[i].Equal(decimal.RequireFromString(tt.expected)),
			"%s: got %s", tt.name, totals.Each[i],
		)
	}
	assert.True(t, totals.Sum.Equal(sum), "sum %s", totals.Sum)
}

func TestAdditionalChargeTotals_NoChargesSumToZero(t *testing.T) {
	t.Parallel()

	totals, err := chargeallocation.AdditionalChargeTotals(
		t.Context(),
		nil,
		decimal.NewFromInt(2450),
	)
	require.NoError(t, err)

	assert.Empty(t, totals.Each)
	assert.True(t, totals.Sum.IsZero())
}
