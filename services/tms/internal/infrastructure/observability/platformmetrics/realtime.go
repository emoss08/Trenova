package platformmetrics

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/observability/metricbase"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

const realtimeSubsystem = "realtime"

type Realtime struct {
	metricbase.Base
	connections       prometheus.Gauge
	connectionsOpened *prometheus.CounterVec
	connectionsClosed *prometheus.CounterVec
	published         *prometheus.CounterVec
	publishDropped    *prometheus.CounterVec
	publishBatch      prometheus.Histogram
	publishLatency    prometheus.Histogram
	delivered         *prometheus.CounterVec
	replays           *prometheus.CounterVec
	fanoutErrors      prometheus.Counter
	presenceSwept     prometheus.Counter
}

func NewRealtime(registry *prometheus.Registry, logger *zap.Logger, enabled bool) *Realtime {
	m := &Realtime{Base: metricbase.NewBase(registry, logger, enabled)}
	if !enabled {
		return m
	}

	m.connections = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metricbase.Namespace,
		Subsystem: realtimeSubsystem,
		Name:      "connections",
		Help:      "Event streams currently open on this instance",
	})
	m.connectionsOpened = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricbase.Namespace,
		Subsystem: realtimeSubsystem,
		Name:      "connections_opened_total",
		Help:      "Event stream open attempts by result",
	}, []string{metricbase.LabelResult})
	m.connectionsClosed = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricbase.Namespace,
		Subsystem: realtimeSubsystem,
		Name:      "connections_closed_total",
		Help:      "Event streams closed by reason",
	}, []string{metricbase.LabelReason})
	m.published = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricbase.Namespace,
		Subsystem: realtimeSubsystem,
		Name:      "published_total",
		Help:      "Events written to the realtime bus by event and result",
	}, []string{"event", metricbase.LabelResult})
	m.publishDropped = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricbase.Namespace,
		Subsystem: realtimeSubsystem,
		Name:      "publish_dropped_total",
		Help:      "Events dropped before reaching the realtime bus by reason",
	}, []string{metricbase.LabelReason})
	m.publishBatch = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metricbase.Namespace,
		Subsystem: realtimeSubsystem,
		Name:      "publish_batch_size",
		Help:      "Events written per pipelined flush",
		Buckets:   []float64{1, 2, 5, 10, 25, 50, 100, 250, 500},
	})
	m.publishLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metricbase.Namespace,
		Subsystem: realtimeSubsystem,
		Name:      "publish_flush_seconds",
		Help:      "Duration of one pipelined flush to the realtime bus",
		Buckets:   metricbase.HTTPDurationBuckets,
	})
	m.delivered = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricbase.Namespace,
		Subsystem: realtimeSubsystem,
		Name:      "delivered_total",
		Help:      "Frames handed to local event streams by event",
	}, []string{"event"})
	m.replays = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricbase.Namespace,
		Subsystem: realtimeSubsystem,
		Name:      "replays_total",
		Help:      "Resume attempts by result",
	}, []string{metricbase.LabelResult})
	m.fanoutErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metricbase.Namespace,
		Subsystem: realtimeSubsystem,
		Name:      "fanout_errors_total",
		Help:      "Failed reads from the realtime bus",
	})
	m.presenceSwept = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metricbase.Namespace,
		Subsystem: realtimeSubsystem,
		Name:      "presence_swept_total",
		Help:      "Presence members removed after their connection stopped refreshing them",
	})

	m.MustRegister(
		m.connections,
		m.connectionsOpened,
		m.connectionsClosed,
		m.published,
		m.publishDropped,
		m.publishBatch,
		m.publishLatency,
		m.delivered,
		m.replays,
		m.fanoutErrors,
		m.presenceSwept,
	)

	return m
}

func (m *Realtime) RecordConnectionOpened(result string) {
	if m == nil {
		return
	}
	m.IfEnabled(func() {
		m.connectionsOpened.WithLabelValues(result).Inc()
		if result == "accepted" {
			m.connections.Inc()
		}
	})
}

func (m *Realtime) RecordConnectionClosed(reason string) {
	if m == nil {
		return
	}
	m.IfEnabled(func() {
		m.connections.Dec()
		m.connectionsClosed.WithLabelValues(reason).Inc()
	})
}

func (m *Realtime) RecordPublished(event, result string, count int) {
	if m == nil {
		return
	}
	m.IfEnabled(func() {
		m.published.WithLabelValues(event, result).Add(float64(count))
	})
}

func (m *Realtime) RecordPublishDropped(reason string) {
	if m == nil {
		return
	}
	m.IfEnabled(func() {
		m.publishDropped.WithLabelValues(reason).Inc()
	})
}

func (m *Realtime) RecordFlush(size int, duration time.Duration) {
	if m == nil {
		return
	}
	m.IfEnabled(func() {
		m.publishBatch.Observe(float64(size))
		m.publishLatency.Observe(duration.Seconds())
	})
}

func (m *Realtime) RecordDelivered(event string) {
	if m == nil {
		return
	}
	m.IfEnabled(func() {
		m.delivered.WithLabelValues(event).Inc()
	})
}

func (m *Realtime) RecordReplay(result string) {
	if m == nil {
		return
	}
	m.IfEnabled(func() {
		m.replays.WithLabelValues(result).Inc()
	})
}

func (m *Realtime) RecordFanoutError() {
	if m == nil {
		return
	}
	m.IfEnabled(func() {
		m.fanoutErrors.Inc()
	})
}

func (m *Realtime) RecordPresenceSwept(count int) {
	if m == nil {
		return
	}
	m.IfEnabled(func() {
		m.presenceSwept.Add(float64(count))
	})
}
