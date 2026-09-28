package carrierintel_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/stretchr/testify/assert"
)

func TestLookupDepthValues_AreEveryValidDepth(t *testing.T) {
	t.Parallel()

	values := carrierintel.LookupDepthValues()
	assert.Len(t, values, 3)
	for _, value := range values {
		assert.True(t, value.IsValid(), value)
	}
	assert.False(t, carrierintel.LookupDepth("Deep").IsValid())
}

func TestUnitTypeValues_AreEveryValidUnitType(t *testing.T) {
	t.Parallel()

	values := carrierintel.UnitTypeValues()
	assert.Len(t, values, 3)
	for _, value := range values {
		assert.True(t, value.IsValid(), value)
	}
	assert.False(t, carrierintel.UnitType("Van").IsValid())
}
