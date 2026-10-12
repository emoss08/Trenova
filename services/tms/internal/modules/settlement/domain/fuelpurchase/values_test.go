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

func TestFuelEnumValues_AreEveryValidValue(t *testing.T) {
	t.Parallel()

	providers := CardProviderValues()
	assert.Len(t, providers, 5)
	for _, value := range providers {
		assert.True(t, value.IsValid(), value)
	}

	statuses := CardStatusValues()
	assert.Len(t, statuses, 3)
	for _, value := range statuses {
		assert.True(t, value.IsValid(), value)
	}

	sources := PurchaseSourceValues()
	assert.Len(t, sources, 2)
	for _, value := range sources {
		assert.True(t, value.IsValid(), value)
	}

	imports := ImportStatusValues()
	assert.Len(t, imports, 5)
	for _, value := range imports {
		assert.True(t, value.IsValid(), value)
	}
}
