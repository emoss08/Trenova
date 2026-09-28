package shipment

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChargeAllocationMethodValues_AreEveryValidMethod(t *testing.T) {
	t.Parallel()

	values := ChargeAllocationMethodValues()
	assert.Equal(t,
		[]ChargeAllocationMethod{ChargeAllocationMethodPercent, ChargeAllocationMethodAmount},
		values)
	for _, value := range values {
		assert.True(t, value.IsValid(), value)
	}
	assert.False(t, ChargeAllocationMethod("Share").IsValid())
}
