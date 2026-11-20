package managedusageservice_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/pulid"

	"github.com/emoss08/trenova/internal/modules/kernel/app/managedusageservice"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
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

var instance = managedscope.Instance()

type fakeRepo struct {
	mu     sync.Mutex
	state  *managedusage.State
	scoped map[string]*managedusage.State
	gets   int
	getsBy map[string]int
	err    error
}

func (f *fakeRepo) Get(
	_ context.Context,
	scope managedscope.Scope,
) (*managedusage.State, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.gets++
	if f.getsBy == nil {
		f.getsBy = map[string]int{}
	}
	f.getsBy[scope.Key()]++
	if f.err != nil {
		return nil, f.err
	}
	held := f.state
	if !scope.IsInstance() {
		held = f.scoped[scope.Key()]
	}
	if held == nil {
		return nil, managedusage.ErrNoState
	}
	c := *held
	return &c, nil
}

func (f *fakeRepo) Save(
	_ context.Context,
	scope managedscope.Scope,
	s *managedusage.State,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	c := *s
	if scope.IsInstance() {
		f.state = &c
		return nil
	}
	if f.scoped == nil {
		f.scoped = map[string]*managedusage.State{}
	}
	f.scoped[scope.Key()] = &c
	return nil
}

type fakeCounter struct {
	meter    platformcatalog.MeterKey
	counts   []managedusageports.HourCount
	req      *managedusageports.HourCountRequest
	byOrg    []managedusageports.OrganizationHourCount
	orgReq   *managedusageports.OrganizationsHourCountRequest
	orgAsked int
}

func (f *fakeCounter) Meter() platformcatalog.MeterKey { return f.meter }

func (f *fakeCounter) CountByHour(
	_ context.Context,
	req *managedusageports.HourCountRequest,
) ([]managedusageports.HourCount, error) {
	f.req = req
	return f.counts, nil
}

func (f *fakeCounter) CountByHourForOrganizations(
	_ context.Context,
	req *managedusageports.OrganizationsHourCountRequest,
) ([]managedusageports.OrganizationHourCount, error) {
	f.orgReq = req
	f.orgAsked++
	return f.byOrg, nil
}

type fakeLevel struct {
	meter    platformcatalog.MeterKey
	level    int64
	err      error
	levels   map[string]int64
	asked    []managedscope.Scope
	orgAsked int
}

func (f *fakeLevel) Meter() platformcatalog.MeterKey { return f.meter }

func (f *fakeLevel) Level(_ context.Context, scope managedscope.Scope) (int64, error) {
	f.asked = append(f.asked, scope)
	if level, ok := f.levels[scope.Key()]; ok {
		return level, f.err
	}
	return f.level, f.err
}

func (f *fakeLevel) LevelsForOrganizations(
	_ context.Context,
	ids []pulid.ID,
) ([]managedusageports.OrganizationLevel, error) {
	f.orgAsked++
	out := make([]managedusageports.OrganizationLevel, 0, len(ids))
	for _, id := range ids {
		if level, ok := f.levels[id.String()]; ok {
			out = append(out, managedusageports.OrganizationLevel{OrganizationID: id, Quantity: level})
		}
	}
	return out, f.err
}

func newService(
	repo *fakeRepo,
	counters []managedusageports.HourlyCounter,
	levels []managedusageports.LevelReader,
) *managedusageservice.Service {
	return newServiceIn(repo, false, counters, levels)
}

func newServiceIn(
	repo *fakeRepo,
	shared bool,
	counters []managedusageports.HourlyCounter,
	levels []managedusageports.LevelReader,
) *managedusageservice.Service {
	return managedusageservice.New(managedusageservice.Params{
		Repo:     repo,
		Scopes:   managedscope.NewResolver(shared),
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

	d, err := svc.Decide(t.Context(), instance, platformcatalog.MeterShipmentsTotal, 1)
	require.NoError(t, err)
	assert.True(t, d.Allowed)
	assert.False(t, d.Managed, "no state yet: nothing is limited")

	change, err := svc.SetState(t.Context(), instance, &managedusage.State{
		WindowStart: now - hour,
		WindowEnd:   now + hour,
		Meters: []managedusage.Meter{{
			Meter:     platformcatalog.MeterShipmentsTotal,
			Unit:      managedusage.UnitCount,
			Quantity:  5,
			Allowance: 5,
			Stopped:   true,
		}},
	})
	require.NoError(t, err)
	assert.Nil(t, change.Previous, "the first state has nothing before it")
	assert.Equal(t, repo.state.ReceivedAt, change.Current.ReceivedAt)
	assert.Equal(t, now, repo.state.ReceivedAt)
	assert.Equal(t, managedscope.InstanceKey, repo.state.ScopeKey)

	d, err = svc.Decide(t.Context(), instance, platformcatalog.MeterShipmentsTotal, 1)
	require.NoError(t, err)
	assert.False(t, d.Allowed, "the write invalidated the cached state")
	assert.Equal(t, managedusage.ReasonStopped, d.Reason)
}

func TestTheStateIsCached(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	svc := newService(repo, nil, nil)
	for range 5 {
		_, err := svc.Decide(t.Context(), instance, platformcatalog.MeterShipmentsTotal, 1)
		require.NoError(t, err)
	}
	assert.Equal(t, 1, repo.gets)
}

func TestAReadFailureIsReturned(t *testing.T) {
	t.Parallel()

	svc := newService(&fakeRepo{err: errors.New("down")}, nil, nil)
	_, err := svc.Decide(t.Context(), instance, platformcatalog.MeterShipmentsTotal, 1)
	require.Error(t, err)
}

func TestSetStateRefusesAnInvalidState(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	svc := newService(repo, nil, nil)
	_, err := svc.SetState(t.Context(), instance, &managedusage.State{WindowStart: 1})
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

	readings, err := svc.Readings(t.Context(), instance, current-2*hour)
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
			_, err := svc.Readings(t.Context(), instance, tt.since)
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
	_, err := svc.Readings(t.Context(), instance, current)
	require.Error(t, err)
}

func storageState(quantity, allowance int64, overage bool) *managedusage.State {
	return &managedusage.State{
		WindowStart: now - hour,
		WindowEnd:   now + hour,
		Overage:     overage,
		Meters: []managedusage.Meter{{
			Meter:     platformcatalog.MeterDocumentStorageBytes,
			Unit:      managedusage.UnitBytes,
			Quantity:  quantity,
			Allowance: allowance,
		}},
	}
}

func TestALevelMeterIsDecidedOnTheLiveLevelNotTheRelayedQuantity(t *testing.T) {
	t.Parallel()

	storage := &fakeLevel{meter: platformcatalog.MeterDocumentStorageBytes, level: 400}
	svc := newService(
		&fakeRepo{state: storageState(1000, 1000, false)},
		nil,
		[]managedusageports.LevelReader{storage},
	)

	d, err := svc.Decide(t.Context(), instance, platformcatalog.MeterDocumentStorageBytes, 600)
	require.NoError(t, err)
	assert.True(t, d.Allowed, "deleting documents frees room before the next relay")
	assert.Equal(t, int64(400), d.Used)

	d, err = svc.Decide(t.Context(), instance, platformcatalog.MeterDocumentStorageBytes, 601)
	require.NoError(t, err)
	assert.False(t, d.Allowed)
	assert.Equal(t, managedusage.ReasonAllowance, d.Reason)
}

func TestALevelMeterTheStateDoesNotNameIsNotRead(t *testing.T) {
	t.Parallel()

	orgs := &fakeLevel{meter: platformcatalog.MeterOrganizationsTotal, err: errors.New("unread")}
	svc := newService(
		&fakeRepo{state: storageState(0, 1000, false)},
		nil,
		[]managedusageports.LevelReader{orgs},
	)

	d, err := svc.Decide(t.Context(), instance, platformcatalog.MeterOrganizationsTotal, 1)
	require.NoError(t, err)
	assert.True(t, d.Allowed)
	assert.False(t, d.Managed)
}

func TestSetStateReportsThePreviousState(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{state: storageState(100, 1000, false)}
	svc := newService(repo, nil, nil)

	change, err := svc.SetState(t.Context(), instance, storageState(900, 1000, false))
	require.NoError(t, err)
	require.NotNil(t, change.Previous)
	previous, _ := change.Previous.Meter(platformcatalog.MeterDocumentStorageBytes)
	current, _ := change.Current.Meter(platformcatalog.MeterDocumentStorageBytes)
	assert.Equal(t, int64(100), previous.Quantity)
	assert.Equal(t, int64(900), current.Quantity)
}

type recordingObserver struct {
	changes []*managedusageports.StateChange
	err     error
}

func (r *recordingObserver) StateChanged(
	_ context.Context,
	change *managedusageports.StateChange,
) error {
	r.changes = append(r.changes, change)
	return r.err
}

func TestTheReceiverTellsEveryObserverAndAFailingOneDoesNotFailTheWrite(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	failing := &recordingObserver{err: errors.New("down")}
	working := &recordingObserver{}
	receiver := managedusageservice.NewReceiver(managedusageservice.ReceiverParams{
		Usage:     newService(repo, nil, nil),
		Observers: []managedusageports.StateObserver{failing, working},
		Logger:    zap.NewNop(),
	})

	require.NoError(t, receiver.SetState(t.Context(), instance, storageState(10, 1000, false)))
	require.Len(t, failing.changes, 1)
	require.Len(t, working.changes, 1)
	assert.NotNil(t, repo.state)
}

func TestTheReceiverTellsNoObserverOfAnInvalidState(t *testing.T) {
	t.Parallel()

	observer := &recordingObserver{}
	receiver := managedusageservice.NewReceiver(managedusageservice.ReceiverParams{
		Usage:     newService(&fakeRepo{}, nil, nil),
		Observers: []managedusageports.StateObserver{observer},
		Logger:    zap.NewNop(),
	})

	require.Error(t, receiver.SetState(t.Context(), instance, &managedusage.State{WindowStart: 1}))
	assert.Empty(t, observer.changes)
}

func TestReadingsListTheFourAppMeters(t *testing.T) {
	t.Parallel()

	current := now - now%hour
	svc := newService(
		&fakeRepo{},
		[]managedusageports.HourlyCounter{
			&fakeCounter{meter: platformcatalog.MeterShipmentsTotal},
			&fakeCounter{meter: platformcatalog.MeterDocumentIntelligencePages},
		},
		[]managedusageports.LevelReader{
			&fakeLevel{meter: platformcatalog.MeterDocumentStorageBytes, level: 1},
			&fakeLevel{meter: platformcatalog.MeterOrganizationsTotal, level: 2},
		},
	)

	readings, err := svc.Readings(t.Context(), instance, current)
	require.NoError(t, err)
	meters := make([]platformcatalog.MeterKey, 0, len(readings))
	for _, r := range readings {
		meters = append(meters, r.Meter)
	}
	assert.ElementsMatch(t, []platformcatalog.MeterKey{
		platformcatalog.MeterShipmentsTotal,
		platformcatalog.MeterDocumentIntelligencePages,
		platformcatalog.MeterDocumentStorageBytes,
		platformcatalog.MeterOrganizationsTotal,
	}, meters)
}
