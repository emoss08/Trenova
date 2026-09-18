package carrierintelligencejobs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A schedule whose args do not match its workflow signature fails the whole
// worker at startup — not at the first fire — so every other domain's jobs go
// down with it. That is cheap to catch here and expensive to catch in a deploy.
func TestSchedulesAreValid(t *testing.T) {
	t.Parallel()

	schedules := NewScheduleProvider().GetSchedules()
	require.NotEmpty(t, schedules)

	for _, s := range schedules {
		t.Run(s.ID, func(t *testing.T) {
			t.Parallel()

			require.NoError(t, s.Validate())
		})
	}
}

// Each of these workflows pages through tenants and continues as new with the
// cursor it reached. A scheduled start has no cursor, so it must begin at the
// first page rather than inherit one from whatever ran last.
func TestSchedulesStartAtTheFirstPage(t *testing.T) {
	t.Parallel()

	for _, s := range NewScheduleProvider().GetSchedules() {
		t.Run(s.ID, func(t *testing.T) {
			t.Parallel()

			require.Len(t, s.Args, 1)

			input, ok := s.Args[0].(*FanOutInput)
			require.True(t, ok, "the workflow takes a *FanOutInput")
			require.NotNil(t, input)
			assert.Nil(t, input.After, "a scheduled run starts from the beginning")
		})
	}
}
