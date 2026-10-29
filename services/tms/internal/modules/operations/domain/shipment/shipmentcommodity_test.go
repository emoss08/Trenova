package shipment_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/stretchr/testify/assert"
)

func TestHasDimensions_RequiresAllThree(t *testing.T) {
	t.Parallel()

	assert.True(t, line(ptr(10), ptr(8), ptr(6), nil).HasDimensions())
	assert.False(t, line(ptr(10), ptr(8), nil, nil).HasDimensions())
	assert.False(t, line(nil, ptr(8), ptr(6), nil).HasDimensions())
	assert.False(t, line(nil, nil, nil, nil).HasDimensions())
}

func ptr(v float64) *float64 { return &v }

func line(length, width, height *float64, com *commodity.Commodity) *shipment.ShipmentCommodity {
	return &shipment.ShipmentCommodity{
		LengthFeet: length,
		WidthFeet:  width,
		HeightFeet: height,
		Commodity:  com,
	}
}
