package sessionwatch

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeLiveness struct {
	mu      sync.Mutex
	ended   map[pulid.ID]bool
	err     error
	checked [][]pulid.ID
}

func (f *fakeLiveness) Ended(_ context.Context, sessionIDs []pulid.ID) ([]pulid.ID, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.checked = append(f.checked, append([]pulid.ID(nil), sessionIDs...))
	if f.err != nil {
		return nil, f.err
	}
	ended := make([]pulid.ID, 0)
	for _, sessionID := range sessionIDs {
		if f.ended[sessionID] {
			ended = append(ended, sessionID)
		}
	}
	return ended, nil
}

func isClosed(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

const (
	liveSession  = pulid.ID("ses_01JQ0000000000000000000001")
	endedSession = pulid.ID("ses_01JQ0000000000000000000002")
)

func TestSweep_EndsEveryStreamOfASessionThatEnded(t *testing.T) {
	t.Parallel()

	liveness := &fakeLiveness{ended: map[pulid.ID]bool{endedSession: true}}
	w := NewWatch(liveness, time.Minute, zap.NewNop())
	firstTab, stopFirst := w.Watch(endedSession)
	secondTab, stopSecond := w.Watch(endedSession)
	otherSession, stopOther := w.Watch(liveSession)
	defer stopFirst()
	defer stopSecond()
	defer stopOther()

	w.Sweep(t.Context())

	assert.True(t, isClosed(firstTab))
	assert.True(t, isClosed(secondTab))
	assert.False(t, isClosed(otherSession))
	require.Len(t, liveness.checked, 1)
	assert.ElementsMatch(t, []pulid.ID{endedSession, liveSession}, liveness.checked[0])
}

func TestSweep_StopsCheckingASessionOnceItsLastStreamCloses(t *testing.T) {
	t.Parallel()

	liveness := &fakeLiveness{}
	w := NewWatch(liveness, time.Minute, zap.NewNop())
	_, stopFirst := w.Watch(liveSession)
	_, stopSecond := w.Watch(liveSession)

	stopFirst()
	stopFirst()
	w.Sweep(t.Context())
	stopSecond()
	w.Sweep(t.Context())

	require.Len(t, liveness.checked, 1)
	assert.Equal(t, []pulid.ID{liveSession}, liveness.checked[0])
}

func TestSweep_KeepsStreamsOpenWhenTheCheckFails(t *testing.T) {
	t.Parallel()

	liveness := &fakeLiveness{
		ended: map[pulid.ID]bool{endedSession: true},
		err:   errors.New("redis unavailable"),
	}
	w := NewWatch(liveness, time.Minute, zap.NewNop())
	ended, stop := w.Watch(endedSession)
	defer stop()

	w.Sweep(t.Context())

	assert.False(t, isClosed(ended))

	liveness.err = nil
	w.Sweep(t.Context())

	assert.True(t, isClosed(ended))
}

func TestSweep_ChecksNothingWhenNoStreamIsOpen(t *testing.T) {
	t.Parallel()

	liveness := &fakeLiveness{}
	w := NewWatch(liveness, time.Minute, zap.NewNop())

	w.Sweep(t.Context())

	assert.Empty(t, liveness.checked)
}

func TestWatch_AStreamClosingAfterItsSessionEndedLeavesTheNextSessionAlone(t *testing.T) {
	t.Parallel()

	liveness := &fakeLiveness{ended: map[pulid.ID]bool{endedSession: true}}
	w := NewWatch(liveness, time.Minute, zap.NewNop())
	_, stopEnded := w.Watch(endedSession)
	w.Sweep(t.Context())

	liveness.ended = map[pulid.ID]bool{}
	reopened, stopReopened := w.Watch(endedSession)
	defer stopReopened()
	stopEnded()
	w.Sweep(t.Context())

	assert.False(t, isClosed(reopened))
	assert.Equal(t, []pulid.ID{endedSession}, liveness.checked[len(liveness.checked)-1])
}

func TestContext_EndsWithItsSession(t *testing.T) {
	t.Parallel()

	liveness := &fakeLiveness{ended: map[pulid.ID]bool{endedSession: true}}
	w := NewWatch(liveness, time.Minute, zap.NewNop())
	ctx, cancel := w.Context(t.Context(), endedSession)
	defer cancel()

	w.Sweep(t.Context())

	select {
	case <-ctx.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("the context outlived its session")
	}
	assert.ErrorIs(t, context.Cause(ctx), ErrSessionEnded)
}

func TestContext_StopsWatchingWhenCancelled(t *testing.T) {
	t.Parallel()

	liveness := &fakeLiveness{}
	w := NewWatch(liveness, time.Minute, zap.NewNop())
	ctx, cancel := w.Context(t.Context(), liveSession)

	cancel()
	w.Sweep(t.Context())

	assert.NotErrorIs(t, context.Cause(ctx), ErrSessionEnded)
	assert.Empty(t, liveness.checked)
}

func TestContext_WithoutASessionFollowsOnlyItsParent(t *testing.T) {
	t.Parallel()

	w := NewWatch(&fakeLiveness{}, time.Minute, zap.NewNop())
	parent, cancelParent := context.WithCancel(t.Context())
	ctx, cancel := w.Context(parent, pulid.Nil)
	defer cancel()

	cancelParent()

	<-ctx.Done()
	assert.ErrorIs(t, context.Cause(ctx), context.Canceled)
}
