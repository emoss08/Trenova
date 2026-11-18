package reconcile_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/operator/internal/logs"
	"github.com/emoss08/trenova/operator/internal/tms"
)

type noGrants struct{}

func (noGrants) AuthorizeLogPull(context.Context, string, tms.LogPullKind) error {
	return tms.ErrNoGrant
}

type fakeLogKeeping struct {
	mu     sync.Mutex
	relays int
	err    error
}

func (*fakeLogKeeping) Kept() bool { return false }

func (*fakeLogKeeping) Longest() time.Duration { return logs.DefaultCaps.Period }

func (*fakeLogKeeping) Last(context.Context, string, int, func(string) string) (string, error) {
	return "", logs.ErrNotKept
}

func (*fakeLogKeeping) Submit(*logs.Job) logs.Submitted { return logs.QueueFull }

func (*fakeLogKeeping) Finish(context.Context, uuid.UUID, *logs.Outcome) {}

func (*fakeLogKeeping) Run(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

func (f *fakeLogKeeping) Relay(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.relays++
	return f.err
}

func (f *fakeLogKeeping) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.relays
}

func TestLogRequestsAreRelayedOnlyWhileTheInstanceIsReady(t *testing.T) {
	t.Parallel()
	w := newWorld(t)

	w.loop.Tick(t.Context())
	require.Zero(t, w.logs.count(), "a TMS that is not ready is not asked")

	w.health.set(true)
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())
	require.Equal(t, 1, w.logs.count())
}

func TestALogRequestFailureNeverHoldsTheReportBack(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	w.health.set(true)
	w.logs.err = errors.New("the TMS answered 503")

	w.loop.Tick(t.Context())

	require.True(t, w.lastReport(t).Ready)
	require.Equal(t, 1, w.logs.count())
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())
	require.Equal(t, 2, w.logs.count(), "retried on the next tick")
}
