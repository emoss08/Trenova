package rateagreement

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValues_AreEveryValidValue(t *testing.T) {
	t.Parallel()

	parties := PartyTypeValues()
	assert.Len(t, parties, 2)
	for _, value := range parties {
		assert.True(t, value.IsValid(), value)
	}

	types := AgreementTypeValues()
	assert.Len(t, types, 5)
	for _, value := range types {
		assert.True(t, value.IsValid(), value)
	}

	statuses := StatusValues()
	assert.Len(t, statuses, 6)
	for _, value := range statuses {
		assert.True(t, value.IsValid(), value)
	}

	directions := DirectionValues()
	assert.Len(t, directions, 2)
	for _, value := range directions {
		assert.True(t, value.IsValid(), value)
	}

	assert.False(t, Status("Pending").IsValid())
}
