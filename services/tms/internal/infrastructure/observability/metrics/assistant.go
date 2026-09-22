package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// TransportInProcess and TransportDurable name where a turn ran.
//
// Every figure here carries one of them, and that is the point. The issue this
// work comes from asks for latency, reliability, recovery and cost compared
// against the runtime being replaced — a comparison nobody can make from
// numbers that do not say which runtime produced them. Instrumenting only the
// new path would have measured it against nothing.
const (
	TransportInProcess = "inprocess"
	TransportDurable   = "durable"
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

type Assistant struct {
	Base

	turnDuration   *prometheus.HistogramVec
	firstEvent     *prometheus.HistogramVec
	turnTotal      *prometheus.CounterVec
	turnsStopped   *prometheus.CounterVec
	stepsReplayed  *prometheus.CounterVec
	streamPublish  *prometheus.CounterVec
	streamBytes    *prometheus.CounterVec
	streamAttached *prometheus.CounterVec
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
		[]string{"transport", "outcome"},
	)

	// Separate from the duration because they answer different questions. The
	// duration says how long the answer took; this says how long the person
	// stared at nothing, which is what they actually experience and the first
	// thing a durable hop could plausibly have made worse.
	m.firstEvent = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: "assistant",
			Name:      "turn_first_event_seconds",
			Help:      "How long a turn took to say anything at all",
			Buckets:   TurnDurationBuckets,
		},
		[]string{"transport"},
	)

	m.turnTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "assistant",
			Name:      "turns_total",
			Help:      "Assistant turns by where they ran and how they ended",
		},
		[]string{"transport", "outcome"},
	)

	m.turnsStopped = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "assistant",
			Name:      "turns_stopped_total",
			Help:      "Turns a person ended themselves, by whether anything had to be cancelled",
		},
		[]string{"transport", "result"},
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

	m.streamPublish = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "assistant",
			Name:      "stream_events_total",
			Help:      "Events published to turn streams, by event and whether the write landed",
		},
		[]string{"event", "result"},
	)

	// Sizing, not health: what a turn's stream costs decides the trim bound,
	// and coalescing text is only justified if this stays far below the
	// event count.
	m.streamBytes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "assistant",
			Name:      "stream_bytes_total",
			Help:      "Bytes published to turn streams",
		},
		[]string{"event"},
	)

	// The recovery figure: whether coming back to a reply actually works, and
	// how often a reader arrives to find the stream already gone.
	m.streamAttached = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "assistant",
			Name:      "stream_attach_total",
			Help:      "Readers attaching to a turn, by what they found",
		},
		[]string{"result"},
	)

	m.mustRegister(
		m.turnDuration,
		m.firstEvent,
		m.turnTotal,
		m.turnsStopped,
		m.stepsReplayed,
		m.streamPublish,
		m.streamBytes,
		m.streamAttached,
	)

	return m
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

// RecordTurn files one finished turn. firstEvent is negative for a turn that
// never said anything, which is not the same as one that answered instantly.
func (m *Assistant) RecordTurn(transport, outcome string, duration, firstEvent float64) {
	if !m.enabled() {
		return
	}

	m.turnTotal.WithLabelValues(transport, outcome).Inc()
	m.turnDuration.WithLabelValues(transport, outcome).Observe(duration)
	if firstEvent >= 0 {
		m.firstEvent.WithLabelValues(transport).Observe(firstEvent)
	}
}

// RecordFirstEvent files how long a turn took to say anything.
//
// It has its own method rather than riding on RecordTurn because the two are
// observed at different moments by different callers: this one by whoever
// holds the stream, as it happens, and the turn's outcome only once it has
// one. Folding them together would have meant inventing an outcome for a turn
// that has not finished.
func (m *Assistant) RecordFirstEvent(transport string, seconds float64) {
	if !m.enabled() {
		return
	}

	m.firstEvent.WithLabelValues(transport).Observe(seconds)
}

// RecordTurnStopped files a turn somebody ended. result says whether there was
// an execution to cancel, which is how a stop that silently did nothing stays
// visible.
func (m *Assistant) RecordTurnStopped(transport, result string) {
	if !m.enabled() {
		return
	}

	m.turnsStopped.WithLabelValues(transport, result).Inc()
}

// RecordStepReplayed files an operation a later attempt did not repeat.
func (m *Assistant) RecordStepReplayed(ownerKind, state string) {
	if !m.enabled() {
		return
	}

	m.stepsReplayed.WithLabelValues(ownerKind, state).Inc()
}

// RecordStreamPublish files one event written to a turn's stream.
func (m *Assistant) RecordStreamPublish(event string, bytes int, err error) {
	if !m.enabled() {
		return
	}

	result := metricStatusSuccess
	if err != nil {
		result = "error"
	}

	m.streamPublish.WithLabelValues(event, result).Inc()
	if err == nil {
		m.streamBytes.WithLabelValues(event).Add(float64(bytes))
	}
}

// RecordStreamAttach files a reader arriving at a turn.
func (m *Assistant) RecordStreamAttach(result string) {
	if !m.enabled() {
		return
	}

	m.streamAttached.WithLabelValues(result).Inc()
}
