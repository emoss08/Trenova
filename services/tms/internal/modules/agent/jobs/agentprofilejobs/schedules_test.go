package agentprofilejobs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSchedules_AreValid(t *testing.T) {
	t.Parallel()

	schedules := NewScheduleProvider().GetSchedules()
	require.NotEmpty(t, schedules)
	for _, s := range schedules {
		require.NoError(t, s.Validate(), s.ID)
	}
}
