package fuelpurchase

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuantityUnitValues_AreEveryValidUnit(t *testing.T) {
	t.Parallel()

	values := QuantityUnitValues()
	assert.Len(t, values, 2)
	for _, value := range values {
		assert.True(t, value.IsValid(), value)
	}
	assert.False(t, QuantityUnit("Barrel").IsValid())
}
