package reconcile_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/operator/internal/controlplanetest"
	"github.com/emoss08/trenova/operator/internal/emailevents"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
)

type fakeEmailEvents struct {
	mu   sync.Mutex
	runs int
	err  error
}

func (f *fakeEmailEvents) Run(context.Context) (emailevents.Result, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.runs++
	return emailevents.Result{Listed: 1, Handled: 1}, f.err
}

func (f *fakeEmailEvents) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.runs
}

func (f *fakeEmailEvents) fail(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = err
}

func queueEmailEvent(w *world) {
	w.cp.QueueEmailEvent(instanceapi.EmailEvent{
		Id:         uuid.New(),
		Provider:   instanceapi.EmailEventResend,
		Payload:    `{"type":"email.bounced"}`,
		ReceivedAt: time.Now(),
	})
}

func TestNoEmailEventsAreFetchedWhenNoneWait(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	w.health.set(true)

	w.loop.Tick(t.Context())

	require.True(t, w.lastReport(t).Ready)
	require.Equal(t, 1, w.cp.CommandListings())
	require.Zero(t, w.events.count(), "the commands answer said none wait")
}

func TestWaitingEmailEventsAreRelayedOnTheTickThatSawThem(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	w.health.set(true)
	queueEmailEvent(w)

	w.loop.Tick(t.Context())

	require.Equal(t, 1, w.events.count())
}

func TestAFailedRelayIsRetriedOnTheNextTickAndHoldsNothingBack(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	w.health.set(true)
	queueEmailEvent(w)
	w.events.fail(errors.New("the TMS answered 503"))

	w.loop.Tick(t.Context())
	require.Equal(t, 1, w.events.count())
	require.Len(t, w.cp.Reports(), 1, "the status report went out")

	w.events.fail(nil)
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())
	require.Equal(t, 2, w.events.count(), "retried on the next tick")
}

func TestEmailEventsWaitWhileTheInstanceIsNotReady(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	queueEmailEvent(w)

	w.loop.Tick(t.Context())

	require.False(t, w.lastReport(t).Ready)
	require.Zero(t, w.events.count())
}

func TestEmailEventsWaitWhileTheInstanceIsSuspended(t *testing.T) {
	t.Parallel()
	spec := controlplanetest.Spec(uuid.New())
	spec.Lifecycle = instanceapi.Suspended
	w := newWorldWith(t, spec, nil)
	w.health.set(true)
	queueEmailEvent(w)

	w.loop.Tick(t.Context())

	require.True(t, w.lastReport(t).Ready)
	require.Zero(t, w.events.count())

	spec.Lifecycle = instanceapi.Active
	w.cp.SetSpec(spec)
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())
	require.Equal(t, 1, w.events.count(), "at once on resuming")
}
