package paymentreminder

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStepReached(t *testing.T) {
	t.Parallel()

	schedule := []int32{1, 15, 30}
	assert.Equal(t, 0, StepReached(0, schedule), "due today")
	assert.Equal(t, 1, StepReached(1, schedule))
	assert.Equal(t, 1, StepReached(14, schedule))
	assert.Equal(t, 2, StepReached(15, schedule))
	assert.Equal(t, 3, StepReached(90, schedule), "past the last step stays on it")
	assert.Equal(t, 0, StepReached(40, nil))
}

func TestIsFinal(t *testing.T) {
	t.Parallel()

	assert.True(t, IsFinal(3, []int32{1, 15, 30}))
	assert.False(t, IsFinal(2, []int32{1, 15, 30}))
	assert.False(t, IsFinal(ManualStep, []int32{1}))
}

func TestDaysPastDue(t *testing.T) {
	t.Parallel()

	day := int64(86_400)
	assert.Equal(t, 0, DaysPastDue(10*day, 10*day))
	assert.Equal(t, 0, DaysPastDue(10*day, 11*day-1), "not a whole day late yet")
	assert.Equal(t, 1, DaysPastDue(10*day, 11*day))
	assert.Equal(t, 0, DaysPastDue(0, 11*day), "no due date")
}
