package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// TurnDurationBuckets span what an answer actually takes.
//
// The short end matters because a refusal is decided in milliseconds and a
// histogram that starts at a second would report every refusal as identical.
// The long end runs to ten minutes because a reasoning model with several tool
// calls legitimately gets there, and a turn falling off the top of the range
// is exactly the case worth seeing.
var TurnDurationBuckets = []float64{
	.05, .25, .5, 1, 2.5, 5, 10, 20, 30, 60, 120, 300, 600,
}

// PhaseBuckets span the parts of a turn before its first token: a database
// read is milliseconds, a classifier call a second or two, and the guard gives
// up at three.
var PhaseBuckets = []float64{
	.005, .01, .025, .05, .1, .25, .5, 1, 2, 3, 5, 10, 30,
}

type Assistant struct {
	Base

	prepare       *prometheus.HistogramVec
	guard         *prometheus.HistogramVec
	modelCallWait prometheus.Histogram

	turnDuration     *prometheus.HistogramVec
	firstEvent       prometheus.Histogram
	turnTotal        *prometheus.CounterVec
	turnsStopped     *prometheus.CounterVec
	stepsReplayed    *prometheus.CounterVec
	streamAttached   *prometheus.CounterVec
	trajectoryEvents *prometheus.CounterVec
}

func NewAssistant(registry *prometheus.Registry, logger *zap.Logger, enabled bool) *Assistant {
	m := &Assistant{Base: NewBase(registry, logger, enabled)}
	if !enabled {
		return m
	}

	m.turnDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: "assistant",
			Name:      "turn_duration_seconds",
			Help:      "How long an assistant turn took, end to end",
			Buckets:   TurnDurationBuckets,
		},
		[]string{"outcome"},
	)

	// Separate from the duration because they answer different questions. The
	// duration says how long the answer took; this says how long the person
	// stared at nothing, which is what they actually experience, and what a
	// slow worker or a backed-up queue makes worse first.
	m.firstEvent = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: "assistant",
			Name:      "turn_first_event_seconds",
			Help:      "How long a reader watching a turn from its start waited for its first event",
			Buckets:   TurnDurationBuckets,
		},
	)

	m.turnTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "assistant",
			Name:      "turns_total",
			Help:      "Assistant turns by how they ended",
		},
		[]string{"outcome"},
	)

	m.turnsStopped = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "assistant",
			Name:      "turns_stopped_total",
			Help:      "Turns a person ended themselves, by whether anything had to be cancelled",
		},
		[]string{labelResult},
	)

	// The reliability figure the issue asks for: a replayed step is a write a
	// retry would have made twice.
	m.stepsReplayed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "assistant",
			Name:      "steps_replayed_total",
			Help:      "Operations a later attempt declined to repeat, by what the ledger knew about them",
		},
		[]string{"owner_kind", "state"},
	)

	// The recovery figure: whether coming back to a reply actually works, and
	// how often a reader arrives to find the stream already gone.
	// What a trajectory costs and, more usefully, what it loses. A dropped
	// event is invisible by design — the run carries on — so the only way it
	// stays visible at all is here.
	m.trajectoryEvents = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "assistant",
			Name:      "trajectory_events_total",
			Help:      "Trajectory events by owner kind and whether they were stored",
		},
		[]string{"owner_kind", labelResult},
	)

	m.streamAttached = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "assistant",
			Name:      "stream_attach_total",
			Help:      "Readers attaching to a turn, by what they found",
		},
		[]string{labelResult},
	)

	// The wait before the first token, part by part: preparing the turn
	// (reads, the scope check, the context), the scope check alone, and how
	// long a model call sat in the queue before a worker took it.
	m.prepare = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: "assistant",
			Name:      "prepare_seconds",
			Help:      "How long preparing a turn took, by whether the question was answered",
			Buckets:   PhaseBuckets,
		},
		[]string{"outcome"},
	)
	m.guard = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: "assistant",
			Name:      "guard_seconds",
			Help:      "How long the scope check took, by the stage that decided it",
			Buckets:   PhaseBuckets,
		},
		[]string{"stage"},
	)
	m.modelCallWait = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: "assistant",
			Name:      "model_call_wait_seconds",
			Help:      "How long a model call waited between being scheduled and a worker starting it",
			Buckets:   PhaseBuckets,
		},
	)

	m.mustRegister(
		m.prepare,
		m.guard,
		m.modelCallWait,
		m.turnDuration,
		m.firstEvent,
		m.turnTotal,
		m.turnsStopped,
		m.stepsReplayed,
		m.streamAttached,
		m.trajectoryEvents,
	)

	return m
}

// AssistantFrom is the registry's assistant collector, or a disabled one when
// there is no registry, which a test does and an install with metrics switched
// off does too. Every method on the result is safe.
func AssistantFrom(registry *Registry) *Assistant {
	if registry == nil || registry.Assistant == nil {
		return NewAssistant(nil, zap.NewNop(), false)
	}

	return registry.Assistant
}

// enabled is nil-safe on the receiver as well as the flag.
//
// A collector is something a caller may legitimately not have: a test builds
// its service without one, and an install with metrics switched off gets a
// disabled collector rather than a nil. Both have to be safe, or measuring
// something becomes a reason it can crash.
func (m *Assistant) enabled() bool {
	return m != nil && m.IsEnabled()
}

// RecordTurn files one finished turn.
func (m *Assistant) RecordTurn(outcome string, duration float64) {
	if !m.enabled() {
		return
	}

	m.turnTotal.WithLabelValues(outcome).Inc()
	m.turnDuration.WithLabelValues(outcome).Observe(duration)
}

// RecordFirstEvent files how long a reader who attached as a turn began
// waited for it to say anything.
func (m *Assistant) RecordFirstEvent(seconds float64) {
	if !m.enabled() {
		return
	}

	m.firstEvent.Observe(seconds)
}

// RecordTurnStopped files a turn somebody ended. result says whether there was
// an execution to cancel, which is how a stop that silently did nothing stays
// visible.
func (m *Assistant) RecordTurnStopped(result string) {
	if !m.enabled() {
		return
	}

	m.turnsStopped.WithLabelValues(result).Inc()
}

// RecordStepReplayed files an operation a later attempt did not repeat.
func (m *Assistant) RecordStepReplayed(ownerKind, state string) {
	if !m.enabled() {
		return
	}

	m.stepsReplayed.WithLabelValues(ownerKind, state).Inc()
}

// RecordTrajectoryWritten files events that reached the database.
func (m *Assistant) RecordTrajectoryWritten(ownerKind string, count int) {
	if !m.enabled() {
		return
	}

	m.trajectoryEvents.WithLabelValues(ownerKind, metricStatusSuccess).Add(float64(count))
}

// RecordTrajectoryDropped files events that did not.
func (m *Assistant) RecordTrajectoryDropped(ownerKind string, count int) {
	if !m.enabled() {
		return
	}

	m.trajectoryEvents.WithLabelValues(ownerKind, "dropped").Add(float64(count))
}

// RecordStreamAttach files a reader arriving at a turn.
func (m *Assistant) RecordStreamAttach(result string) {
	if !m.enabled() {
		return
	}

	m.streamAttached.WithLabelValues(result).Inc()
}

// RecordPrepare files how long preparing a turn took.
func (m *Assistant) RecordPrepare(outcome string, seconds float64) {
	if !m.enabled() {
		return
	}

	m.prepare.WithLabelValues(outcome).Observe(seconds)
}

// RecordGuard files how long the scope check took.
func (m *Assistant) RecordGuard(stage string, seconds float64) {
	if !m.enabled() {
		return
	}

	m.guard.WithLabelValues(stage).Observe(seconds)
}

// RecordModelCallWait files how long a model call waited for a worker.
func (m *Assistant) RecordModelCallWait(seconds float64) {
	if !m.enabled() {
		return
	}

	m.modelCallWait.Observe(seconds)
}
