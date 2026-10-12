package shipment

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFreightTermsValues_AreEveryValidTerm(t *testing.T) {
	t.Parallel()

	values := FreightTermsValues()

	assert.Len(t, values, 3)
	for _, value := range values {
		assert.True(t, value.IsValid(), value)
	}
	assert.False(t, FreightTerms("Paid").IsValid())
}
