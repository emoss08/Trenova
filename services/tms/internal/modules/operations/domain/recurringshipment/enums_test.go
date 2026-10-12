package recurringshipment

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatusValues_AreEveryValidStatus(t *testing.T) {
	t.Parallel()

	values := StatusValues()
	assert.Len(t, values, 3)
	for _, value := range values {
		assert.True(t, value.IsValid(), value)
	}
	assert.False(t, Status("Archived").IsValid())
}

func TestExceptionPolicyValues_AreEveryValidPolicy(t *testing.T) {
	t.Parallel()

	values := ExceptionPolicyValues()
	assert.Len(t, values, 3)
	for _, value := range values {
		assert.True(t, value.IsValid(), value)
	}
	assert.False(t, ExceptionPolicy("Nearest").IsValid())
}
