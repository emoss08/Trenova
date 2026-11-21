package reviewprompt

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/stretchr/testify/assert"
)

func TestDistanceTotalsTheBilledDistance(t *testing.T) {
	t.Parallel()

	routed, billed, other := 520.0, 500.0, 80.0
	moves := []*shipment.ShipmentMove{
		{Distance: &routed, BillingDistance: &billed, DistanceUnits: "Miles"},
		{Distance: &other, DistanceUnits: "Miles"},
	}

	assert.Equal(t, "580 Miles", distance(moves))
}
