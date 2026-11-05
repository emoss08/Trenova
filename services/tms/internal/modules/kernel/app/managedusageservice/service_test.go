package managedusageservice_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/kernel/app/managedusageservice"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedusageports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	hour = int64(3600)
	now  = int64(1_762_351_200 + 1_234)
)

type fakeRepo struct {
	mu    sync.Mutex
	state *managedusage.State
	gets  int
	err   error
}

func (f *fakeRepo) Get(context.Context) (*managedusage.State, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.gets++
	if f.err != nil {
		return nil, f.err
	}
	if f.state == nil {
		return nil, managedusage.ErrNoState
	}
	c := *f.state
	return &c, nil
}

func (f *fakeRepo) Save(_ context.Context, s *managedusage.State) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	c := *s
	f.state = &c
	return nil
}

type fakeCounter struct {
	meter  platformcatalog.MeterKey
	counts []managedusageports.HourCount
	req    *managedusageports.HourCountRequest
}

func (f *fakeCounter) Meter() platformcatalog.MeterKey { return f.meter }

func (f *fakeCounter) CountByHour(
	_ context.Context,
	req *managedusageports.HourCountRequest,
) ([]managedusageports.HourCount, error) {
	f.req = req
	return f.counts, nil
}

type fakeLevel struct {
	meter platformcatalog.MeterKey
	level int64
	err   error
}

func (f *fakeLevel) Meter() platformcatalog.MeterKey { return f.meter }

func (f *fakeLevel) Level(context.Context) (int64, error) { return f.level, f.err }

func newService(
	repo *fakeRepo,
	counters []managedusageports.HourlyCounter,
	levels []managedusageports.LevelReader,
) *managedusageservice.Service {
	return managedusageservice.New(managedusageservice.Params{
		Repo:     repo,
		Counters: counters,
		Levels:   levels,
		Clock:    func() int64 { return now },
		Logger:   zap.NewNop(),
	})
}

func TestSetStateThenDecide(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	svc := newService(repo, nil, nil)

	d, err := svc.Decide(t.Context(), platformcatalog.MeterShipmentsTotal, 1)
	require.NoError(t, err)
	assert.True(t, d.Allowed)
	assert.False(t, d.Managed, "no state yet: nothing is limited")

	require.NoError(t, svc.SetState(t.Context(), &managedusage.State{
		WindowStart: now - hour,
		WindowEnd:   now + hour,
		Meters: []managedusage.Meter{{
			Meter:     platformcatalog.MeterShipmentsTotal,
			Unit:      managedusage.UnitCount,
			Quantity:  5,
			Allowance: 5,
			Stopped:   true,
		}},
	}))
	assert.Equal(t, now, repo.state.ReceivedAt)
	assert.Equal(t, managedusage.CurrentStateID, repo.state.ID)

	d, err = svc.Decide(t.Context(), platformcatalog.MeterShipmentsTotal, 1)
	require.NoError(t, err)
	assert.False(t, d.Allowed, "the write invalidated the cached state")
	assert.Equal(t, managedusage.ReasonStopped, d.Reason)
}

func TestTheStateIsCached(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	svc := newService(repo, nil, nil)
	for range 5 {
		_, err := svc.Decide(t.Context(), platformcatalog.MeterAISpendCents, 1)
		require.NoError(t, err)
	}
	assert.Equal(t, 1, repo.gets)
}

func TestAReadFailureIsReturned(t *testing.T) {
	t.Parallel()

	svc := newService(&fakeRepo{err: errors.New("down")}, nil, nil)
	_, err := svc.Decide(t.Context(), platformcatalog.MeterAISpendCents, 1)
	require.Error(t, err)
}

func TestSetStateRefusesAnInvalidState(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	svc := newService(repo, nil, nil)
	err := svc.SetState(t.Context(), &managedusage.State{WindowStart: 1})
	require.Error(t, err)
	assert.Nil(t, repo.state)
}

func TestReadings(t *testing.T) {
	t.Parallel()

	current := now - now%hour
	shipments := &fakeCounter{
		meter: platformcatalog.MeterShipmentsTotal,
		counts: []managedusageports.HourCount{
			{Hour: current - 2*hour, Quantity: 3},
			{Hour: current, Quantity: 4},
		},
	}
	storage := &fakeLevel{meter: platformcatalog.MeterDocumentStorageBytes, level: 2048}
	svc := newService(
		&fakeRepo{},
		[]managedusageports.HourlyCounter{shipments},
		[]managedusageports.LevelReader{storage},
	)

	readings, err := svc.Readings(t.Context(), current-2*hour)
	require.NoError(t, err)
	assert.Equal(t, []managedusageservice.Reading{
		{Meter: platformcatalog.MeterShipmentsTotal, Hour: current - 2*hour, Quantity: 3},
		{Meter: platformcatalog.MeterShipmentsTotal, Hour: current - hour, Quantity: 0},
		{Meter: platformcatalog.MeterShipmentsTotal, Hour: current, Quantity: 4},
		{Meter: platformcatalog.MeterDocumentStorageBytes, Hour: current, Quantity: 2048},
	}, readings)
	assert.Equal(t, current-2*hour, shipments.req.Since)
	assert.Equal(t, current+hour, shipments.req.Until)
}

func TestReadingsRefuseABadSince(t *testing.T) {
	t.Parallel()

	current := now - now%hour
	svc := newService(&fakeRepo{}, nil, nil)
	tests := []struct {
		name  string
		since int64
		err   error
	}{
		{
			name:  "not on the hour",
			since: current - 60,
			err:   managedusageservice.ErrSinceNotOnTheHour,
		},
		{name: "in the future", since: current + hour, err: managedusageservice.ErrSinceInFuture},
		{
			name:  "too old",
			since: current - int64(36*24*time.Hour/time.Second),
			err:   managedusageservice.ErrSinceTooOld,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := svc.Readings(t.Context(), tt.since)
			require.ErrorIs(t, err, tt.err)
		})
	}
}

func TestReadingsReturnAReaderFailure(t *testing.T) {
	t.Parallel()

	current := now - now%hour
	svc := newService(&fakeRepo{}, nil, []managedusageports.LevelReader{
		&fakeLevel{meter: platformcatalog.MeterDocumentStorageBytes, err: errors.New("down")},
	})
	_, err := svc.Readings(t.Context(), current)
	require.Error(t, err)
}
