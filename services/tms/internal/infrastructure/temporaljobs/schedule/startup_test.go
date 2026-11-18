package schedule

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/temporalconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type downScheduleClient struct {
	*mockScheduleClient
	down atomic.Bool
}

func (d *downScheduleClient) List(
	ctx context.Context,
	opts client.ScheduleListOptions,
) (client.ScheduleListIterator, error) {
	if d.down.Load() {
		d.listCalls.Add(1)
		return nil, errors.New("rpc error: code = Unavailable desc = connection refused")
	}
	return d.mockScheduleClient.List(ctx, opts)
}

func newStartupScheduler(t *testing.T, logger *zap.Logger) (*Scheduler, *downScheduleClient) {
	t.Helper()

	registry := NewRegistry(logger)
	registry.RegisterProvider(&testProvider{schedules: []*Schedule{{
		ID:        "late-start",
		Spec:      Every(30 * time.Minute),
		Workflow:  dummyWorkflow,
		TaskQueue: "test-queue",
		Feature:   testFeature,
	}}})

	sc := &downScheduleClient{mockScheduleClient: &mockScheduleClient{
		listIter:     &mockScheduleListIterator{},
		createHandle: &mockScheduleHandle{id: "late-start"},
	}}
	sc.down.Store(true)

	s := &Scheduler{
		reconciler: NewReconciler(
			&mockTemporalClient{scheduleClient: sc},
			registry,
			allOn(),
			logger,
		),
		registry: registry,
		config:   &temporalconfig.Config{},
		logger:   logger.Named("scheduler"),
		interval: time.Hour,
	}

	return s, sc
}

func TestScheduler_StartsWithTemporalDownAndReconcilesWhenItAnswers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		core, logs := observer.New(zap.InfoLevel)
		s, sc := newStartupScheduler(t, zap.New(core))

		require.NoError(t, s.Start(t.Context()), "start does not wait for Temporal")

		time.Sleep(3 * time.Minute)
		synctest.Wait()
		assert.Greater(t, sc.listCalls.Load(), int32(3), "it keeps retrying")
		assert.Empty(t, sc.created)
		assert.LessOrEqual(t, logs.FilterMessageSnippet("not reconciled yet").Len(), 4,
			"a retry line at most once a minute")

		sc.down.Store(false)
		time.Sleep(retryMaxInterval + retryMaxInterval/4)
		synctest.Wait()

		require.NoError(t, s.Stop(t.Context()))
		require.Len(t, sc.created, 1)
		assert.Equal(t, "late-start", sc.created[0].ID)
		assert.Equal(t, 1, logs.FilterMessage("schedule reconciliation complete").Len())
	})
}

func TestScheduler_StopEndsTheRetries(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s, sc := newStartupScheduler(t, zap.NewNop())

		require.NoError(t, s.Start(t.Context()))
		time.Sleep(10 * time.Second)
		require.NoError(t, s.Stop(t.Context()))

		calls := sc.listCalls.Load()
		time.Sleep(5 * time.Minute)
		assert.Equal(t, calls, sc.listCalls.Load())
	})
}

func TestScheduler_StartRefusesABadDeclaration(t *testing.T) {
	t.Parallel()

	registry := NewRegistry(zap.NewNop())
	registry.RegisterProvider(&testProvider{schedules: []*Schedule{{Spec: Every(time.Minute)}}})
	s := &Scheduler{registry: registry, logger: zap.NewNop()}

	require.Error(t, s.Start(t.Context()))
	assert.Nil(t, s.done, "nothing runs in the background")
}
