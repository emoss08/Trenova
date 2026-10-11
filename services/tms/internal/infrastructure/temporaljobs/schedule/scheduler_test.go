package schedule

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx/fxtest"
)

func TestNewScheduler_WithProviders(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()

	provider := &testProvider{
		schedules: []*Schedule{
			{
				ID:        "sched-1",
				Spec:      Every(30 * time.Minute),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
		},
	}

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{
			listIter: &mockScheduleListIterator{
				entries: []*client.ScheduleListEntry{},
			},
			createHandle: &mockScheduleHandle{id: "sched-1"},
		},
	}

	lc := fxtest.NewLifecycle(t)

	s := NewScheduler(SchedulerParams{
		Client:    mc,
		Config:    &config.Config{},
		Logger:    logger,
		LC:        lc,
		Providers: []Provider{provider},
	})

	require.NotNil(t, s)
	assert.NotNil(t, s.GetRegistry())
	assert.NotNil(t, s.GetReconciler())
	assert.Equal(t, 1, s.GetRegistry().ProviderCount())
}

func TestNewScheduler_NoProviders(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{},
	}

	lc := fxtest.NewLifecycle(t)

	s := NewScheduler(SchedulerParams{
		Client:    mc,
		Config:    &config.Config{},
		Logger:    logger,
		LC:        lc,
		Providers: []Provider{},
	})

	require.NotNil(t, s)
	assert.Equal(t, 0, s.GetRegistry().ProviderCount())
}

func TestScheduler_Start_NoProviders(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{},
	}

	reconciler := NewReconciler(mc, registry, logger)

	s := &Scheduler{
		reconciler: reconciler,
		registry:   registry,
		config:     &config.Config{},
		logger:     logger.Named("scheduler"),
	}

	err := s.Start(t.Context())
	assert.NoError(t, err)
}

func TestScheduler_Start_WithProviders_Success(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider := &testProvider{
		schedules: []*Schedule{
			{
				ID:        "start-test",
				Spec:      Every(30 * time.Minute),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
		},
	}
	registry.RegisterProvider(provider)

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{
			listIter: &mockScheduleListIterator{
				entries: []*client.ScheduleListEntry{},
			},
			createHandle: &mockScheduleHandle{id: "start-test"},
		},
	}

	reconciler := NewReconciler(mc, registry, logger)

	s := &Scheduler{
		reconciler: reconciler,
		registry:   registry,
		config:     &config.Config{},
		logger:     logger.Named("scheduler"),
	}

	err := s.Start(t.Context())
	assert.NoError(t, err)
}

func TestScheduler_Start_WithProviders_Error(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider := &testProvider{
		schedules: []*Schedule{
			{
				ID:   "",
				Spec: Every(30 * time.Minute),
			},
		},
	}
	registry.RegisterProvider(provider)

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{},
	}

	reconciler := NewReconciler(mc, registry, logger)

	s := &Scheduler{
		reconciler: reconciler,
		registry:   registry,
		config:     &config.Config{},
		logger:     logger.Named("scheduler"),
	}

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()

	err := s.Start(ctx)
	assert.Error(t, err)
}

func TestScheduler_Stop_PersistOnStopTrue(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	s := &Scheduler{
		registry: registry,
		config: &config.Config{
			Temporal: config.TemporalConfig{
				Schedule: config.TemporalScheduleConfig{
					PersistOnStop: true,
				},
			},
		},
		logger: logger.Named("scheduler"),
	}

	err := s.Stop(t.Context())
	assert.NoError(t, err)
}

func TestScheduler_Stop_PersistOnStopFalse(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	s := &Scheduler{
		registry: registry,
		config: &config.Config{
			Temporal: config.TemporalConfig{
				Schedule: config.TemporalScheduleConfig{
					PersistOnStop: false,
				},
			},
		},
		logger: logger.Named("scheduler"),
	}

	err := s.Stop(t.Context())
	assert.NoError(t, err)
}

func TestScheduler_GetRegistry(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	s := &Scheduler{
		registry: registry,
		config:   &config.Config{},
		logger:   logger.Named("scheduler"),
	}

	assert.Equal(t, registry, s.GetRegistry())
}

func TestScheduler_GetReconciler(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{},
	}

	reconciler := NewReconciler(mc, registry, logger)

	s := &Scheduler{
		reconciler: reconciler,
		registry:   registry,
		config:     &config.Config{},
		logger:     logger.Named("scheduler"),
	}

	assert.Equal(t, reconciler, s.GetReconciler())
}

func TestScheduler_ForceReconcile(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider := &testProvider{
		schedules: []*Schedule{
			{
				ID:        "force-reconcile",
				Spec:      Every(30 * time.Minute),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
		},
	}
	registry.RegisterProvider(provider)

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{
			listIter: &mockScheduleListIterator{
				entries: []*client.ScheduleListEntry{},
			},
			createHandle: &mockScheduleHandle{id: "force-reconcile"},
		},
	}

	reconciler := NewReconciler(mc, registry, logger)

	s := &Scheduler{
		reconciler: reconciler,
		registry:   registry,
		config:     &config.Config{},
		logger:     logger.Named("scheduler"),
	}

	result, err := s.ForceReconcile(t.Context())

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Created, 1)
	assert.Contains(t, result.Created, "force-reconcile")
}

func TestScheduler_ForceReconcile_Error(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider := &testProvider{
		schedules: []*Schedule{
			{
				ID:        "force-fail",
				Spec:      Every(30 * time.Minute),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
		},
	}
	registry.RegisterProvider(provider)

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{
			listErr: errors.New("force reconcile failed"),
		},
	}

	reconciler := NewReconciler(mc, registry, logger)

	s := &Scheduler{
		reconciler: reconciler,
		registry:   registry,
		config:     &config.Config{},
		logger:     logger.Named("scheduler"),
	}

	result, err := s.ForceReconcile(t.Context())

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "force reconcile failed")
}
