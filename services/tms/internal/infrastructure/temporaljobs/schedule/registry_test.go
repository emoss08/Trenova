package schedule

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type testProvider struct {
	schedules []*Schedule
}

func (p *testProvider) GetSchedules() []*Schedule {
	return p.schedules
}

func newTestLogger() *zap.Logger {
	return zap.NewNop()
}

func TestRegistry_RegisterProvider(t *testing.T) {
	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider := &testProvider{
		schedules: []*Schedule{
			{
				ID:        "test-1",
				Spec:      Every(30 * time.Minute),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
		},
	}

	registry.RegisterProvider(provider)

	assert.Equal(t, 1, registry.ProviderCount())
}

func TestRegistry_CollectSchedules(t *testing.T) {
	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider1 := &testProvider{
		schedules: []*Schedule{
			{
				ID:        "schedule-1",
				Spec:      Every(30 * time.Minute),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
		},
	}

	provider2 := &testProvider{
		schedules: []*Schedule{
			{
				ID:        "schedule-2",
				Spec:      Cron("0 9 * * *"),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
		},
	}

	registry.RegisterProvider(provider1)
	registry.RegisterProvider(provider2)

	err := registry.CollectSchedules()
	require.NoError(t, err)

	assert.Equal(t, 2, registry.Count())

	schedules := registry.GetSchedules()
	assert.Contains(t, schedules, "schedule-1")
	assert.Contains(t, schedules, "schedule-2")
}

func TestRegistry_CollectSchedules_DuplicateID(t *testing.T) {
	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider1 := &testProvider{
		schedules: []*Schedule{
			{
				ID:          "duplicate-id",
				Description: "First schedule",
				Spec:        Every(30 * time.Minute),
				Workflow:    dummyWorkflow,
				TaskQueue:   "test-queue",
			},
		},
	}

	provider2 := &testProvider{
		schedules: []*Schedule{
			{
				ID:          "duplicate-id",
				Description: "Second schedule",
				Spec:        Cron("0 9 * * *"),
				Workflow:    dummyWorkflow,
				TaskQueue:   "test-queue",
			},
		},
	}

	registry.RegisterProvider(provider1)
	registry.RegisterProvider(provider2)

	err := registry.CollectSchedules()
	assert.ErrorIs(t, err, ErrDuplicateScheduleID)
}

func TestRegistry_CollectSchedules_InvalidSchedule(t *testing.T) {
	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider := &testProvider{
		schedules: []*Schedule{
			{
				ID:        "",
				Spec:      Every(30 * time.Minute),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
		},
	}

	registry.RegisterProvider(provider)

	err := registry.CollectSchedules()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid schedule")
}

func TestRegistry_GetSchedule(t *testing.T) {
	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider := &testProvider{
		schedules: []*Schedule{
			{
				ID:        "existing-schedule",
				Spec:      Every(30 * time.Minute),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
		},
	}

	registry.RegisterProvider(provider)
	err := registry.CollectSchedules()
	require.NoError(t, err)

	schedule, exists := registry.GetSchedule("existing-schedule")
	assert.True(t, exists)
	assert.Equal(t, "existing-schedule", schedule.ID)

	_, exists = registry.GetSchedule("non-existent")
	assert.False(t, exists)
}

func TestRegistry_GetScheduleIDs(t *testing.T) {
	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider := &testProvider{
		schedules: []*Schedule{
			{
				ID:        "schedule-a",
				Spec:      Every(30 * time.Minute),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
			{
				ID:        "schedule-b",
				Spec:      Every(1 * time.Hour),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
		},
	}

	registry.RegisterProvider(provider)
	err := registry.CollectSchedules()
	require.NoError(t, err)

	ids := registry.GetScheduleIDs()
	assert.Len(t, ids, 2)
	assert.Contains(t, ids, "schedule-a")
	assert.Contains(t, ids, "schedule-b")
}

func TestProviderFunc(t *testing.T) {
	schedules := []*Schedule{
		{
			ID:        "func-schedule",
			Spec:      Every(1 * time.Hour),
			Workflow:  dummyWorkflow,
			TaskQueue: "test-queue",
		},
	}

	provider := ProviderFunc(func() []*Schedule {
		return schedules
	})

	result := provider.GetSchedules()
	assert.Equal(t, schedules, result)
}
