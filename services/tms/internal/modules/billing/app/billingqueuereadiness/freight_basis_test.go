package billingqueuereadiness

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestFreightBasisDividesTheFreightByTheBilledMiles(t *testing.T) {
	t.Parallel()

	routed, billed, empty := 520.0, 500.0, 80.0
	shp := &shipment.Shipment{
		FreightChargeAmount: decimal.NewNullDecimal(decimal.NewFromInt(1000)),
		Moves: []*shipment.ShipmentMove{
			{Loaded: true, Distance: &routed, BillingDistance: &billed},
			{Loaded: false, Distance: &empty},
		},
	}

	assert.Equal(t, "500 mi × $2.00", freightBasis(shp))
}
