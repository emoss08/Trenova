package worker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAvailabilityPreferenceValues_AreEveryValidPreference(t *testing.T) {
	t.Parallel()

	values := AvailabilityPreferenceValues()
	assert.Len(t, values, 3)
	for _, value := range values {
		assert.True(t, value.IsValid(), value)
	}
	assert.False(t, AvailabilityPreference("Maybe").IsValid())
}
