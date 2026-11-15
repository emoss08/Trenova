package schedule

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/temporalconfig"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
)

const (
	testFeature  = platformcatalog.FeatureCoreTMS
	addOnFeature = platformcatalog.FeatureAccounting
)

type fakeGate struct {
	mu    sync.Mutex
	off   map[platformcatalog.FeatureKey]struct{}
	err   error
	calls int
}

func allOn() *fakeGate {
	return &fakeGate{off: map[platformcatalog.FeatureKey]struct{}{}}
}

func without(features ...platformcatalog.FeatureKey) *fakeGate {
	gate := allOn()
	gate.turnOff(features...)
	return gate
}

func (g *fakeGate) turnOff(features ...platformcatalog.FeatureKey) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, feature := range features {
		g.off[feature] = struct{}{}
	}
}

func (g *fakeGate) turnOn(features ...platformcatalog.FeatureKey) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, feature := range features {
		delete(g.off, feature)
	}
}

func (g *fakeGate) fail(err error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.err = err
}

func (g *fakeGate) Enabled(_ context.Context, feature platformcatalog.FeatureKey) (bool, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.calls++
	if g.err != nil {
		return false, g.err
	}
	_, off := g.off[feature]
	return !off, nil
}

func enabled(s *Schedule) desired {
	return desired{Schedule: s}
}

func withheldBy(s *Schedule) desired {
	return desired{Schedule: s, withheld: true}
}

func addOnSchedule(id string) *Schedule {
	return &Schedule{
		ID:        id,
		Feature:   addOnFeature,
		Spec:      Every(time.Hour),
		Workflow:  dummyWorkflow,
		TaskQueue: "test-queue",
	}
}

func reconcileOnce(
	t *testing.T,
	sched *Schedule,
	gate *fakeGate,
	sc *mockScheduleClient,
) (*ReconcileResult, error) {
	t.Helper()

	logger := newTestLogger()
	registry := NewRegistry(logger)
	registry.RegisterProvider(&testProvider{schedules: []*Schedule{sched}})

	return NewReconciler(&mockTemporalClient{scheduleClient: sc}, registry, gate, logger).
		Reconcile(t.Context())
}

func existingWith(
	id string,
	state client.ScheduleState,
) (*mockScheduleClient, *mockScheduleHandle) {
	handle := &mockScheduleHandle{id: id, invokeUpdate: true, current: state}
	sc := &mockScheduleClient{
		listIter: &mockScheduleListIterator{
			entries: []*client.ScheduleListEntry{{ID: id, Note: state.Note}},
		},
		handles: map[string]*mockScheduleHandle{id: handle},
	}
	return sc, handle
}

func TestReconcile_WithheldFeatureCreatesTheSchedulePausedWithTheReason(t *testing.T) {
	t.Parallel()

	sched := addOnSchedule("late-charges")
	sc := &mockScheduleClient{listIter: &mockScheduleListIterator{}}

	result, err := reconcileOnce(t, sched, without(addOnFeature), sc)
	require.NoError(t, err)

	assert.Equal(t, []string{"late-charges"}, result.Created)
	assert.Equal(t, []platformcatalog.FeatureKey{addOnFeature}, result.Withheld)
	require.Len(t, sc.created, 1)
	assert.True(t, sc.created[0].Paused)
	assert.True(
		t,
		strings.HasPrefix(
			sc.created[0].Note,
			"Paused because the plan does not include Accounting.",
		),
		sc.created[0].Note,
	)
	hash, ok := hashFromNote(sc.created[0].Note)
	require.True(t, ok)
	assert.Equal(t, withheldBy(sched).hash(), hash)
	assert.NotEqual(t, sched.Hash(), hash)
}

func TestReconcile_TurningTheFeatureOnUnpausesTheSchedule(t *testing.T) {
	t.Parallel()

	sched := addOnSchedule("late-charges")
	note := withheldBy(sched).note()
	sc, handle := existingWith(sched.ID, client.ScheduleState{Paused: true, Note: note})

	result, err := reconcileOnce(t, sched, allOn(), sc)
	require.NoError(t, err)

	assert.Equal(t, []string{sched.ID}, result.Updated)
	assert.Empty(t, result.Withheld)
	require.NotNil(t, handle.lastUpdate)
	assert.False(t, handle.lastUpdate.Schedule.State.Paused)
	assert.Equal(t, registryNote(sched.Hash()), handle.lastUpdate.Schedule.State.Note)
}

func TestReconcile_TurningTheFeatureOffPausesTheSchedule(t *testing.T) {
	t.Parallel()

	sched := addOnSchedule("late-charges")
	note := registryNote(sched.Hash())
	sc, handle := existingWith(sched.ID, client.ScheduleState{Note: note})

	result, err := reconcileOnce(t, sched, without(addOnFeature), sc)
	require.NoError(t, err)

	assert.Equal(t, []string{sched.ID}, result.Updated)
	require.NotNil(t, handle.lastUpdate)
	assert.True(t, handle.lastUpdate.Schedule.State.Paused)
	assert.Equal(t, withheldBy(sched).note(), handle.lastUpdate.Schedule.State.Note)
}

func TestReconcile_TurningTheFeatureOffPausesAScheduleAPersonUnpaused(t *testing.T) {
	t.Parallel()

	sched := addOnSchedule("late-charges")
	sc, handle := existingWith(
		sched.ID,
		client.ScheduleState{Paused: false, Note: "Unpaused via Go SDK"},
	)

	_, err := reconcileOnce(t, sched, without(addOnFeature), sc)
	require.NoError(t, err)

	require.NotNil(t, handle.lastUpdate)
	assert.True(t, handle.lastUpdate.Schedule.State.Paused)
	assert.Equal(t, withheldBy(sched).note(), handle.lastUpdate.Schedule.State.Note)
}

func TestReconcile_ADeclaredPauseStaysWhenTheFeatureIsOn(t *testing.T) {
	t.Parallel()

	sched := addOnSchedule("late-charges")
	sched.Paused = true
	note := withheldBy(sched).note()
	sc, handle := existingWith(sched.ID, client.ScheduleState{Paused: true, Note: note})

	result, err := reconcileOnce(t, sched, allOn(), sc)
	require.NoError(t, err)

	assert.Equal(t, []string{sched.ID}, result.Updated)
	require.NotNil(t, handle.lastUpdate)
	assert.True(t, handle.lastUpdate.Schedule.State.Paused)
	assert.Equal(t, registryNote(sched.Hash()), handle.lastUpdate.Schedule.State.Note)
}

func TestReconcile_AHandPauseOnAnEnabledScheduleSurvives(t *testing.T) {
	t.Parallel()

	sched := addOnSchedule("late-charges")
	sc, handle := existingWith(
		sched.ID,
		client.ScheduleState{Paused: true, Note: "Paused while the ledger is migrated"},
	)

	_, err := reconcileOnce(t, sched, allOn(), sc)
	require.NoError(t, err)

	require.NotNil(t, handle.lastUpdate)
	kept := handle.lastUpdate.Schedule.State
	assert.True(t, kept.Paused, "the reconciler must not undo a pause a person made")
	assert.True(t, strings.HasPrefix(kept.Note, "Paused while the ledger is migrated"), kept.Note)
	hash, ok := hashFromNote(kept.Note)
	require.True(t, ok)
	assert.Equal(t, sched.Hash(), hash)

	again, handleAgain := existingWith(sched.ID, *kept)
	result, err := reconcileOnce(t, sched, allOn(), again)
	require.NoError(t, err)

	assert.Equal(t, []string{sched.ID}, result.Skipped, "a pass with no change leaves it alone")
	assert.False(t, handleAgain.updateCalled)
}

func TestReconcile_AHandPauseIsKeptWhenTheDeclarationChanges(t *testing.T) {
	t.Parallel()

	sched := addOnSchedule("late-charges")
	before := *sched
	before.Spec = Every(2 * time.Hour)
	kept := keptNote("Paused while the ledger is migrated", before.Hash())
	sc, handle := existingWith(sched.ID, client.ScheduleState{Paused: true, Note: kept})

	result, err := reconcileOnce(t, sched, allOn(), sc)
	require.NoError(t, err)

	assert.Equal(t, []string{sched.ID}, result.Updated)
	require.NotNil(t, handle.lastUpdate)
	assert.True(t, handle.lastUpdate.Schedule.State.Paused)
	assert.Equal(
		t,
		keptNote("Paused while the ledger is migrated", sched.Hash()),
		handle.lastUpdate.Schedule.State.Note,
	)
}

func TestReconcile_AGateErrorFailsThePassAndChangesNothing(t *testing.T) {
	t.Parallel()

	sched := addOnSchedule("late-charges")
	gate := allOn()
	gate.fail(errors.New("managed feature state unavailable"))
	sc, handle := existingWith(
		sched.ID,
		client.ScheduleState{Paused: true, Note: withheldBy(sched).note()},
	)

	result, err := reconcileOnce(t, sched, gate, sc)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "managed feature state unavailable")
	assert.Zero(t, sc.listCalls.Load())
	assert.Empty(t, sc.created)
	assert.False(t, handle.updateCalled)
	assert.False(t, handle.deleteCalled)
}

func TestReconcile_EverythingOnMatchesTheDeclaration(t *testing.T) {
	t.Parallel()

	sched := addOnSchedule("late-charges")
	sc := &mockScheduleClient{listIter: &mockScheduleListIterator{}}

	result, err := reconcileOnce(t, sched, allOn(), sc)
	require.NoError(t, err)

	assert.Equal(t, []string{sched.ID}, result.Created)
	assert.Empty(t, result.Withheld)
	require.Len(t, sc.created, 1)
	assert.False(t, sc.created[0].Paused)
	assert.Equal(t, registryNote(sched.Hash()), sc.created[0].Note)
	assert.Equal(t, sched.Hash(), enabled(sched).hash(), "every feature on changes no hash")
}

func TestReconcile_AsksTheGateOncePerFeature(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)
	registry.RegisterProvider(&testProvider{schedules: []*Schedule{
		addOnSchedule("a"),
		addOnSchedule("b"),
		addOnSchedule("c"),
	}})
	gate := allOn()
	sc := &mockScheduleClient{listIter: &mockScheduleListIterator{}}

	_, err := NewReconciler(&mockTemporalClient{scheduleClient: sc}, registry, gate, logger).
		Reconcile(t.Context())
	require.NoError(t, err)

	assert.Equal(t, 1, gate.calls)
}

func newPlanScheduler(t *testing.T, gate *fakeGate) (*Scheduler, *mockScheduleClient) {
	t.Helper()

	logger := newTestLogger()
	registry := NewRegistry(logger)
	registry.RegisterProvider(&testProvider{schedules: []*Schedule{addOnSchedule("late-charges")}})
	require.NoError(t, registry.CollectSchedules())

	sc := &mockScheduleClient{listIter: &mockScheduleListIterator{}}
	s := &Scheduler{
		reconciler: NewReconciler(&mockTemporalClient{scheduleClient: sc}, registry, gate, logger),
		registry:   registry,
		config:     &temporalconfig.Config{},
		logger:     logger.Named("scheduler"),
		interval:   time.Hour,
	}

	return s, sc
}

func TestScheduler_APlanChangeReconcilesOnce(t *testing.T) {
	t.Parallel()

	gate := allOn()
	s, sc := newPlanScheduler(t, gate)

	gate.turnOff(addOnFeature)
	applied := s.checkFeatures(t.Context(), nil)

	assert.Equal(t, []platformcatalog.FeatureKey{addOnFeature}, applied)
	assert.Equal(t, int32(1), sc.listCalls.Load())

	applied = s.checkFeatures(t.Context(), applied)

	assert.Equal(t, []platformcatalog.FeatureKey{addOnFeature}, applied)
	assert.Equal(t, int32(1), sc.listCalls.Load(), "the same plan reconciles nothing")
}

func TestScheduler_NoPlanChangeReconcilesNothing(t *testing.T) {
	t.Parallel()

	gate := without(addOnFeature)
	s, sc := newPlanScheduler(t, gate)

	applied := s.checkFeatures(t.Context(), []platformcatalog.FeatureKey{addOnFeature})

	assert.Equal(t, []platformcatalog.FeatureKey{addOnFeature}, applied)
	assert.Zero(t, sc.listCalls.Load())
	assert.Positive(t, gate.calls)
}

func TestScheduler_AFailedReconcileIsRetriedOnTheNextCheck(t *testing.T) {
	t.Parallel()

	gate := allOn()
	s, sc := newPlanScheduler(t, gate)
	gate.turnOff(addOnFeature)

	sc.listErr = errors.New("temporal unavailable")
	applied := s.checkFeatures(t.Context(), nil)

	assert.Empty(t, applied, "a failed pass keeps what was last applied")
	assert.Equal(t, int32(1), sc.listCalls.Load())

	sc.listErr = nil
	applied = s.checkFeatures(t.Context(), applied)

	assert.Equal(t, []platformcatalog.FeatureKey{addOnFeature}, applied)
	assert.Equal(t, int32(2), sc.listCalls.Load())
}

func TestScheduler_AGateErrorReconcilesNothing(t *testing.T) {
	t.Parallel()

	gate := allOn()
	s, sc := newPlanScheduler(t, gate)
	gate.fail(errors.New("managed feature state unavailable"))

	applied := s.checkFeatures(t.Context(), nil)

	assert.Empty(t, applied)
	assert.Zero(t, sc.listCalls.Load())
}

func TestScheduler_ThePlanIsWatchedFromStartUntilStop(t *testing.T) {
	t.Parallel()

	gate := allOn()
	s, sc := newPlanScheduler(t, gate)
	s.interval = time.Millisecond

	require.NoError(t, s.Start(t.Context()))
	assert.Equal(t, int32(1), sc.listCalls.Load())

	gate.turnOff(addOnFeature)
	require.Eventually(t, func() bool {
		return sc.listCalls.Load() == 2
	}, 5*time.Second, time.Millisecond)

	require.NoError(t, s.Stop(t.Context()))
	select {
	case <-s.done:
	default:
		t.Fatal("Stop returned while the plan check was still running")
	}

	calls := sc.listCalls.Load()
	gate.turnOn(addOnFeature)
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, calls, sc.listCalls.Load(), "nothing reconciles after Stop")
}

func TestScheduler_StopBeforeStartIsANoOp(t *testing.T) {
	t.Parallel()

	s, _ := newPlanScheduler(t, allOn())

	assert.NoError(t, s.Stop(t.Context()))
}
