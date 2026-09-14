package shipment_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestAdditionalChargeTotal(t *testing.T) {
	t.Parallel()

	base := decimal.NewFromInt(2450)
	tests := []struct {
		name     string
		charge   *shipment.AdditionalCharge
		expected string
	}{
		{"nil charge", nil, "0"},
		{"flat once", &shipment.AdditionalCharge{Method: accessorialcharge.MethodFlat, Amount: decimal.NewFromInt(150), Unit: 1}, "150"},
		{"flat with zero unit still bills once", &shipment.AdditionalCharge{Method: accessorialcharge.MethodFlat, Amount: decimal.NewFromInt(150), Unit: 0}, "150"},
		{"flat repeated", &shipment.AdditionalCharge{Method: accessorialcharge.MethodFlat, Amount: decimal.NewFromInt(150), Unit: 2}, "300"},
		{"per unit", &shipment.AdditionalCharge{Method: accessorialcharge.MethodPerUnit, Amount: decimal.NewFromInt(75), Unit: 2}, "150"},
		{"per unit without units bills nothing", &shipment.AdditionalCharge{Method: accessorialcharge.MethodPerUnit, Amount: decimal.NewFromInt(75), Unit: 0}, "0"},
		{"percentage of base", &shipment.AdditionalCharge{Method: accessorialcharge.MethodPercentage, Amount: decimal.NewFromInt(10), Unit: 1}, "245"},
		{"unknown method", &shipment.AdditionalCharge{Method: accessorialcharge.Method("Tiered"), Amount: decimal.NewFromInt(10)}, "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.charge.Total(base)
			assert.True(t, got.Equal(decimal.RequireFromString(tt.expected)), "got %s", got)
		})
	}
}

func TestAdditionalChargesTotalSumsEveryCharge(t *testing.T) {
	t.Parallel()

	total := shipment.AdditionalChargesTotal([]*shipment.AdditionalCharge{
		{Method: accessorialcharge.MethodFlat, Amount: decimal.NewFromInt(150), Unit: 1},
		nil,
		{Method: accessorialcharge.MethodPercentage, Amount: decimal.NewFromInt(10), Unit: 1},
	}, decimal.NewFromInt(2450))

	assert.True(t, total.Equal(decimal.NewFromInt(395)))
	assert.True(t, shipment.AdditionalChargesTotal(nil, decimal.NewFromInt(2450)).IsZero())
}
