package shipmentjobs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScheduleProvider_AutoDelayWorkflowUsesPayloadArgs(t *testing.T) {
	t.Parallel()

	provider := NewScheduleProvider()
	schedules := provider.GetSchedules()

	require.Len(t, schedules, 2)
	require.Empty(t, schedules[0].Args)
	require.Empty(t, schedules[1].Args)
}
