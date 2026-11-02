package sessionwatch

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/ports/sessionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeLiveness struct {
	mu      sync.Mutex
	ended   map[pulid.ID]bool
	secrets map[pulid.ID][]string
	unknown map[pulid.ID]bool
	err     error
	checked [][]pulid.ID
}

func (f *fakeLiveness) Standings(
	_ context.Context,
	sessionIDs []pulid.ID,
) ([]sessionports.SessionStanding, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.checked = append(f.checked, append([]pulid.ID(nil), sessionIDs...))
	if f.err != nil {
		return nil, f.err
	}
	standings := make([]sessionports.SessionStanding, 0, len(sessionIDs))
	for _, sessionID := range sessionIDs {
		standing := sessionports.SessionStanding{ID: sessionID, Ended: f.ended[sessionID]}
		if !standing.Ended && !f.unknown[sessionID] {
			standing.SecretsKnown = true
			standing.SecretHashes = f.secrets[sessionID]
			if standing.SecretHashes == nil {
				standing.SecretHashes = []string{currentSecret}
			}
		}
		standings = append(standings, standing)
	}
	return standings, nil
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
	liveSession   = pulid.ID("ses_01JQ0000000000000000000001")
	endedSession  = pulid.ID("ses_01JQ0000000000000000000002")
	currentSecret = "current-secret-hash"
	oldSecret     = "old-secret-hash"
)

func TestSweep_EndsEveryStreamOfASessionThatEnded(t *testing.T) {
	t.Parallel()

	liveness := &fakeLiveness{ended: map[pulid.ID]bool{endedSession: true}}
	w := NewWatch(liveness, time.Minute, zap.NewNop())
	firstTab, stopFirst := w.Watch(endedSession, currentSecret)
	secondTab, stopSecond := w.Watch(endedSession, currentSecret)
	otherSession, stopOther := w.Watch(liveSession, currentSecret)
	defer stopFirst()
	defer stopSecond()
	defer stopOther()

	w.Sweep(t.Context())

	assert.True(t, isClosed(firstTab.Done()))
	assert.True(t, isClosed(secondTab.Done()))
	require.ErrorIs(t, firstTab.Cause(), ErrSessionEnded)
	assert.False(t, isClosed(otherSession.Done()))
	require.Len(t, liveness.checked, 1)
	assert.ElementsMatch(t, []pulid.ID{endedSession, liveSession}, liveness.checked[0])
}

func TestSweep_EndsOnlyTheStreamsOpenedWithASupersededSecret(t *testing.T) {
	t.Parallel()

	liveness := &fakeLiveness{secrets: map[pulid.ID][]string{liveSession: {currentSecret}}}
	w := NewWatch(liveness, time.Minute, zap.NewNop())
	stale, stopStale := w.Watch(liveSession, oldSecret)
	fresh, stopFresh := w.Watch(liveSession, currentSecret)
	defer stopStale()
	defer stopFresh()

	w.Sweep(t.Context())

	assert.True(t, isClosed(stale.Done()))
	require.ErrorIs(t, stale.Cause(), ErrSessionRotated)
	assert.False(t, isClosed(fresh.Done()))
	require.Len(t, liveness.checked, 1)
	assert.Equal(t, []pulid.ID{liveSession}, liveness.checked[0])
}

func TestSweep_KeepsAStreamWhoseSecretIsStillInItsGrace(t *testing.T) {
	t.Parallel()

	liveness := &fakeLiveness{
		secrets: map[pulid.ID][]string{liveSession: {currentSecret, oldSecret}},
	}
	w := NewWatch(liveness, time.Minute, zap.NewNop())
	graced, stop := w.Watch(liveSession, oldSecret)
	defer stop()

	w.Sweep(t.Context())

	assert.False(t, isClosed(graced.Done()))
}

func TestSweep_KeepsStreamsOpenWhenTheSecretsCannotBeRead(t *testing.T) {
	t.Parallel()

	liveness := &fakeLiveness{unknown: map[pulid.ID]bool{liveSession: true}}
	w := NewWatch(liveness, time.Minute, zap.NewNop())
	stream, stop := w.Watch(liveSession, oldSecret)
	defer stop()

	w.Sweep(t.Context())

	assert.False(t, isClosed(stream.Done()))
}

func TestSweep_StopsCheckingASessionOnceItsLastStreamCloses(t *testing.T) {
	t.Parallel()

	liveness := &fakeLiveness{}
	w := NewWatch(liveness, time.Minute, zap.NewNop())
	_, stopFirst := w.Watch(liveSession, currentSecret)
	_, stopSecond := w.Watch(liveSession, currentSecret)

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
	ended, stop := w.Watch(endedSession, currentSecret)
	defer stop()

	w.Sweep(t.Context())

	assert.False(t, isClosed(ended.Done()))

	liveness.err = nil
	w.Sweep(t.Context())

	assert.True(t, isClosed(ended.Done()))
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
	_, stopEnded := w.Watch(endedSession, currentSecret)
	w.Sweep(t.Context())

	liveness.ended = map[pulid.ID]bool{}
	reopened, stopReopened := w.Watch(endedSession, currentSecret)
	defer stopReopened()
	stopEnded()
	w.Sweep(t.Context())

	assert.False(t, isClosed(reopened.Done()))
	assert.Equal(t, []pulid.ID{endedSession}, liveness.checked[len(liveness.checked)-1])
}

func TestContext_EndsWithItsSession(t *testing.T) {
	t.Parallel()

	liveness := &fakeLiveness{ended: map[pulid.ID]bool{endedSession: true}}
	w := NewWatch(liveness, time.Minute, zap.NewNop())
	ctx, cancel := w.Context(t.Context(), endedSession, currentSecret)
	defer cancel()

	w.Sweep(t.Context())

	select {
	case <-ctx.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("the context outlived its session")
	}
	assert.ErrorIs(t, context.Cause(ctx), ErrSessionEnded)
}

func TestContext_EndsWhenItsSecretIsSuperseded(t *testing.T) {
	t.Parallel()

	w := NewWatch(&fakeLiveness{}, time.Minute, zap.NewNop())
	ctx, cancel := w.Context(t.Context(), liveSession, oldSecret)
	defer cancel()

	w.Sweep(t.Context())

	select {
	case <-ctx.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("the context outlived its secret")
	}
	assert.ErrorIs(t, context.Cause(ctx), ErrSessionRotated)
}

func TestContext_StopsWatchingWhenCancelled(t *testing.T) {
	t.Parallel()

	liveness := &fakeLiveness{}
	w := NewWatch(liveness, time.Minute, zap.NewNop())
	ctx, cancel := w.Context(t.Context(), liveSession, currentSecret)

	cancel()
	w.Sweep(t.Context())

	assert.NotErrorIs(t, context.Cause(ctx), ErrSessionEnded)
	assert.Empty(t, liveness.checked)
}

func TestContext_WithoutASessionFollowsOnlyItsParent(t *testing.T) {
	t.Parallel()

	w := NewWatch(&fakeLiveness{}, time.Minute, zap.NewNop())
	parent, cancelParent := context.WithCancel(t.Context())
	ctx, cancel := w.Context(parent, pulid.Nil, "")
	defer cancel()

	cancelParent()

	<-ctx.Done()
	assert.ErrorIs(t, context.Cause(ctx), context.Canceled)
}

func TestSessionContext_OutlivesARotationButNotTheSession(t *testing.T) {
	t.Parallel()

	liveness := &fakeLiveness{secrets: map[pulid.ID][]string{liveSession: {currentSecret}}}
	w := NewWatch(liveness, time.Minute, zap.NewNop())
	ctx, cancel := w.SessionContext(t.Context(), liveSession)
	defer cancel()

	w.Sweep(t.Context())
	assert.NoError(t, ctx.Err())

	liveness.ended = map[pulid.ID]bool{liveSession: true}
	w.Sweep(t.Context())

	select {
	case <-ctx.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("the context outlived its session")
	}
	assert.ErrorIs(t, context.Cause(ctx), ErrSessionEnded)
}
