package datausageservice_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/kernel/app/datausageservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/datausageports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
)

const start = int64(1_794_830_400)

type fakeClock struct {
	now atomic.Int64
}

func newClock() *fakeClock {
	c := &fakeClock{}
	c.now.Store(start)
	return c
}

func (c *fakeClock) Now() int64 { return c.now.Load() }

func (c *fakeClock) Advance(d time.Duration) { c.now.Add(int64(d / time.Second)) }

type fakeSource struct {
	figure  datausageports.Figure
	timeout time.Duration

	mu    sync.Mutex
	bytes int64
	err   error
	block bool
	calls atomic.Int32
}

func (f *fakeSource) Figure() datausageports.Figure { return f.figure }

func (f *fakeSource) Timeout() time.Duration {
	if f.timeout == 0 {
		return time.Second
	}
	return f.timeout
}

func (f *fakeSource) Measure(ctx context.Context) (int64, error) {
	f.calls.Add(1)
	f.mu.Lock()
	bytes, err, block := f.bytes, f.err, f.block
	f.mu.Unlock()
	if block {
		<-ctx.Done()
		return 0, ctx.Err()
	}
	return bytes, err
}

func (f *fakeSource) set(bytes int64, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.bytes, f.err = bytes, err
}

func newService(
	t *testing.T,
	clock *fakeClock,
	sources ...datausageports.Source,
) *datausageservice.Service {
	t.Helper()
	lc := fxtest.NewLifecycle(t)
	svc, err := datausageservice.New(datausageservice.Params{
		Sources: sources,
		Clock:   clock.Now,
		Logger:  zap.NewNop(),
		LC:      lc,
	})
	require.NoError(t, err)
	lc.RequireStart()
	t.Cleanup(lc.RequireStop)
	return svc
}

func calledTimes(t *testing.T, src *fakeSource, n int32) {
	t.Helper()
	require.Eventually(t, func() bool { return src.calls.Load() == n },
		2*time.Second, 5*time.Millisecond)
}

func measuredBy(t *testing.T, svc *datausageservice.Service, src *fakeSource, n int32) {
	t.Helper()
	require.Eventually(t, func() bool {
		svc.Usage(t.Context())
		return src.calls.Load() == n
	}, 2*time.Second, 5*time.Millisecond)
}

func TestTheFirstRequestMeasuresEverySource(t *testing.T) {
	t.Parallel()
	clock := newClock()
	db := &fakeSource{figure: datausageports.FigureDatabase, bytes: 5 << 30}
	docs := &fakeSource{figure: datausageports.FigureDocuments, bytes: 2 << 30}
	search := &fakeSource{figure: datausageports.FigureSearch, bytes: 0}
	svc := newService(t, clock, db, docs, search)

	got := svc.Usage(t.Context())

	require.NotNil(t, got.DatabaseBytes)
	require.NotNil(t, got.DocumentsBytes)
	require.NotNil(t, got.SearchBytes, "a measured empty index is zero, not unknown")
	assert.Equal(t, int64(5<<30), *got.DatabaseBytes)
	assert.Equal(t, int64(2<<30), *got.DocumentsBytes)
	assert.Equal(t, int64(0), *got.SearchBytes)
	assert.Equal(t, start, got.MeasuredAt)
}

func TestAFailingSourceDoesNotBlankTheOthers(t *testing.T) {
	t.Parallel()
	clock := newClock()
	db := &fakeSource{figure: datausageports.FigureDatabase, bytes: 100}
	docs := &fakeSource{figure: datausageports.FigureDocuments, err: errors.New("down")}
	search := &fakeSource{figure: datausageports.FigureSearch, err: datausageports.ErrNotMeasured}
	svc := newService(t, clock, db, docs, search)

	got := svc.Usage(t.Context())

	require.NotNil(t, got.DatabaseBytes)
	assert.Equal(t, int64(100), *got.DatabaseBytes)
	assert.Nil(t, got.DocumentsBytes)
	assert.Nil(t, got.SearchBytes, "search that is not configured is absent")
}

func TestANegativeSizeIsNotReported(t *testing.T) {
	t.Parallel()
	clock := newClock()
	db := &fakeSource{figure: datausageports.FigureDatabase, bytes: -1}
	svc := newService(t, clock, db)

	assert.Nil(t, svc.Usage(t.Context()).DatabaseBytes)
}

func TestSourcesAreMeasuredAtMostEveryFiveMinutes(t *testing.T) {
	t.Parallel()
	clock := newClock()
	db := &fakeSource{figure: datausageports.FigureDatabase, bytes: 100}
	svc := newService(t, clock, db)

	svc.Usage(t.Context())
	calledTimes(t, db, 1)

	for range 5 {
		clock.Advance(time.Minute - time.Second)
		svc.Usage(t.Context())
	}
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, int32(1), db.calls.Load(), "a request inside the five minutes measures nothing")

	clock.Advance(5 * time.Second)
	db.set(200, nil)
	measuredBy(t, svc, db, 2)

	require.Eventually(t, func() bool {
		return *svc.Usage(t.Context()).DatabaseBytes == 200
	}, 2*time.Second, 5*time.Millisecond)
	assert.Equal(
		t,
		int32(2),
		db.calls.Load(),
		"one measurement per five minutes, however often asked",
	)
}

func TestAFailedSourceKeepsItsLastValueForThirtyMinutes(t *testing.T) {
	t.Parallel()
	clock := newClock()
	db := &fakeSource{figure: datausageports.FigureDatabase, bytes: 100}
	docs := &fakeSource{figure: datausageports.FigureDocuments, bytes: 7}
	svc := newService(t, clock, db, docs)

	svc.Usage(t.Context())
	db.set(0, errors.New("the database did not answer"))

	for i := int32(2); i <= 6; i++ {
		clock.Advance(5 * time.Minute)
		measuredBy(t, svc, db, i)
		got := svc.Usage(t.Context())
		require.NotNil(t, got.DatabaseBytes, "still within 30 minutes")
		assert.Equal(t, int64(100), *got.DatabaseBytes)
		assert.Equal(t, start, got.MeasuredAt,
			"measuredAt is when the oldest figure in the answer was measured")
	}

	clock.Advance(5*time.Minute + time.Second)
	measuredBy(t, svc, db, 7)
	want := start + int64((30*time.Minute+time.Second)/time.Second)
	require.Eventually(t, func() bool {
		return svc.Usage(t.Context()).MeasuredAt == want
	}, 2*time.Second, 5*time.Millisecond)

	got := svc.Usage(t.Context())
	assert.Nil(t, got.DatabaseBytes, "past 30 minutes a figure that cannot be measured is absent")
	require.NotNil(t, got.DocumentsBytes)
	assert.Equal(t, int64(7), *got.DocumentsBytes)
}

func TestASlowSourceIsBoundedByItsTimeout(t *testing.T) {
	t.Parallel()
	clock := newClock()
	db := &fakeSource{figure: datausageports.FigureDatabase, bytes: 100}
	search := &fakeSource{
		figure:  datausageports.FigureSearch,
		timeout: 50 * time.Millisecond,
		block:   true,
	}
	svc := newService(t, clock, db, search)

	began := time.Now()
	got := svc.Usage(t.Context())

	assert.Less(t, time.Since(began), time.Second)
	require.NotNil(t, got.DatabaseBytes)
	assert.Nil(t, got.SearchBytes)
}

func TestTheFirstRequestWaitsAtMostTwoSeconds(t *testing.T) {
	t.Parallel()
	clock := newClock()
	db := &fakeSource{figure: datausageports.FigureDatabase, timeout: time.Minute, block: true}
	svc := newService(t, clock, db)

	began := time.Now()
	got := svc.Usage(t.Context())
	waited := time.Since(began)

	assert.GreaterOrEqual(t, waited, datausageservice.FirstWait-100*time.Millisecond)
	assert.Less(t, waited, datausageservice.FirstWait+time.Second)
	assert.Nil(t, got.DatabaseBytes)
	assert.Equal(t, start, got.MeasuredAt, "with nothing known the answer carries only the time")

	again := time.Now()
	svc.Usage(t.Context())
	assert.Less(t, time.Since(again), datausageservice.FirstWait,
		"a measurement already under way is not waited for twice")
	assert.Equal(t, int32(1), db.calls.Load())
}

func TestLaterRequestsNeverWaitForAMeasurement(t *testing.T) {
	t.Parallel()
	clock := newClock()
	db := &fakeSource{figure: datausageports.FigureDatabase, bytes: 100, timeout: time.Minute}
	svc := newService(t, clock, db)
	svc.Usage(t.Context())

	db.mu.Lock()
	db.block = true
	db.mu.Unlock()
	clock.Advance(5 * time.Minute)

	began := time.Now()
	got := svc.Usage(t.Context())
	assert.Less(t, time.Since(began), 100*time.Millisecond)
	require.NotNil(t, got.DatabaseBytes)
	assert.Equal(t, int64(100), *got.DatabaseBytes)
	calledTimes(t, db, 2)
}

func TestTwoSourcesForOneFigureAreRefused(t *testing.T) {
	t.Parallel()
	_, err := datausageservice.New(datausageservice.Params{
		Sources: []datausageports.Source{
			&fakeSource{figure: datausageports.FigureSearch},
			&fakeSource{figure: datausageports.FigureSearch},
		},
		Clock:  newClock().Now,
		Logger: zap.NewNop(),
		LC:     fxtest.NewLifecycle(t),
	})
	require.Error(t, err)
}
