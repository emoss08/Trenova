package tooldispatch

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStepData_KeepsOnlyAnObjectSmallEnoughToShowAgain(t *testing.T) {
	t.Parallel()

	assert.Nil(t, stepData(nil))
	assert.Nil(t, stepData([]any{"a"}), "a bare list builds no artifact")
	assert.Nil(t, stepData(map[string]any{"cycle": make(chan int)}))
	assert.Nil(t, stepData(map[string]any{"blob": string(make([]byte, maxStoredStepDataBytes))}),
		"a result past the bound is dropped, not truncated")
	assert.Equal(t, map[string]any{"count": float64(2)}, stepData(map[string]any{"count": 2}))
}
