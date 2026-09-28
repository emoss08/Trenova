package ifta

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIFTAEnumValues_AreEveryValidValue(t *testing.T) {
	t.Parallel()

	sources := MileageSourceValues()
	assert.Len(t, sources, 3)
	for _, value := range sources {
		assert.True(t, value.IsValid(), value)
	}

	statuses := ReturnStatusValues()
	assert.Len(t, statuses, 3)
	for _, value := range statuses {
		assert.True(t, value.IsValid(), value)
	}
	assert.False(t, ReturnStatus("Amended").IsValid())
}
