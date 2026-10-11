package weatheralertjobs

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/enums/v1"
)

func TestScheduleProviderGetSchedules(t *testing.T) {
	t.Parallel()

	provider := NewScheduleProvider()
	schedules := provider.GetSchedules()

	require.Len(t, schedules, 1)
	assert.Equal(t, "weather-alert-poll", schedules[0].ID)
	assert.Equal(t, temporaltype.TaskQueueWeatherAlert.String(), schedules[0].TaskQueue)
	assert.Equal(t, 5*time.Minute, schedules[0].Spec.Interval)
	assert.Equal(t, enums.SCHEDULE_OVERLAP_POLICY_SKIP, schedules[0].OverlapPolicy)
}

func TestSchedules_MatchTheirWorkflowSignatures(t *testing.T) {
	t.Parallel()

	for _, s := range NewScheduleProvider().GetSchedules() {
		require.NoError(t, s.Validate(), s.ID)
	}
}
