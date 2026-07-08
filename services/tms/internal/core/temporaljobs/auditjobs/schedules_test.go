package auditjobs

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/enums/v1"
)

func TestNewScheduleProvider(t *testing.T) {
	t.Parallel()

	provider := NewScheduleProvider(newTestConfig())
	require.NotNil(t, provider)
}

func TestScheduleProvider_GetSchedules(t *testing.T) {
	t.Parallel()

	provider := NewScheduleProvider(newTestConfig())
	schedules := provider.GetSchedules()

	require.Len(t, schedules, 3)
}

func TestScheduleProvider_AuditBufferFlush(t *testing.T) {
	t.Parallel()

	provider := NewScheduleProvider(newTestConfig())
	schedules := provider.GetSchedules()

	flushSchedule := schedules[0]

	assert.Equal(t, "audit-buffer-flush", flushSchedule.ID)
	assert.Equal(t, "Flush audit buffer from Redis for batch processing", flushSchedule.Description)
	assert.NotNil(t, flushSchedule.Workflow)
	assert.Equal(t, temporaltype.AuditTaskQueue, flushSchedule.TaskQueue)
	assert.Equal(t, enums.SCHEDULE_OVERLAP_POLICY_SKIP, flushSchedule.OverlapPolicy)
	assert.Equal(t, time.Minute, flushSchedule.Spec.Interval)
	assert.NotNil(t, flushSchedule.Memo)
	assert.Equal(t, "batch-processing", flushSchedule.Memo["purpose"])
	assert.Equal(t, "audit_redis_buffer", flushSchedule.Memo["target"])
}

func TestScheduleProvider_AuditBufferFlushConfigOverride(t *testing.T) {
	t.Parallel()

	provider := NewScheduleProvider(&config.Config{
		Audit: config.AuditConfig{
			BufferFlushInterval: 2 * time.Minute,
		},
	})
	schedules := provider.GetSchedules()

	require.NotEmpty(t, schedules)
	assert.Equal(t, 2*time.Minute, schedules[0].Spec.Interval)
}

func TestScheduleProvider_RetentionPurge(t *testing.T) {
	t.Parallel()

	provider := NewScheduleProvider(newTestConfig())
	schedules := provider.GetSchedules()

	retentionSchedule := schedules[1]

	assert.Equal(t, "audit-retention-purge", retentionSchedule.ID)
	assert.NotNil(t, retentionSchedule.Workflow)
	assert.Equal(t, temporaltype.AuditTaskQueue, retentionSchedule.TaskQueue)
	assert.Equal(t, enums.SCHEDULE_OVERLAP_POLICY_SKIP, retentionSchedule.OverlapPolicy)
}

func TestScheduleProvider_DLQRetry(t *testing.T) {
	t.Parallel()

	provider := NewScheduleProvider(newTestConfig())
	schedules := provider.GetSchedules()

	dlqSchedule := schedules[2]

	assert.Equal(t, "audit-dlq-retry", dlqSchedule.ID)
	assert.Equal(t, "Retry failed audit entries from dead-letter queue", dlqSchedule.Description)
	assert.NotNil(t, dlqSchedule.Workflow)
	assert.Equal(t, temporaltype.AuditTaskQueue, dlqSchedule.TaskQueue)
	assert.Equal(t, enums.SCHEDULE_OVERLAP_POLICY_SKIP, dlqSchedule.OverlapPolicy)
	assert.NotNil(t, dlqSchedule.Memo)
	assert.Equal(t, "dlq-retry", dlqSchedule.Memo["purpose"])
	assert.Equal(t, "audit_dlq", dlqSchedule.Memo["target"])
}

func TestScheduleProvider_AllSchedulesHaveRequiredFields(t *testing.T) {
	t.Parallel()

	provider := NewScheduleProvider(newTestConfig())
	schedules := provider.GetSchedules()

	for _, s := range schedules {
		t.Run(s.ID, func(t *testing.T) {
			t.Parallel()
			assert.NotEmpty(t, s.ID)
			assert.NotEmpty(t, s.Description)
			assert.NotNil(t, s.Workflow)
			assert.NotEmpty(t, s.TaskQueue)
			assert.NotNil(t, s.Spec)
			assert.NotNil(t, s.Memo)
		})
	}
}

func TestScheduleProvider_UniqueIDs(t *testing.T) {
	t.Parallel()

	provider := NewScheduleProvider(newTestConfig())
	schedules := provider.GetSchedules()

	ids := make(map[string]bool)
	for _, s := range schedules {
		assert.False(t, ids[s.ID], "duplicate schedule ID: %s", s.ID)
		ids[s.ID] = true
	}
}

func newTestConfig() *config.Config {
	return &config.Config{}
}
