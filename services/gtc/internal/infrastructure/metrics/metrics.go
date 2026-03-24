package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	EventsReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gtc",
			Name:      "events_received_total",
			Help:      "Total number of CDC events received from WAL",
		},
		[]string{"schema", "table", "operation"},
	)

	EventsProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gtc",
			Name:      "events_processed_total",
			Help:      "Total number of CDC events processed by sinks",
		},
		[]string{"sink", "status"},
	)

	EventProcessingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "gtc",
			Name:      "event_processing_duration_seconds",
			Help:      "Duration of event processing by sink",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"sink"},
	)

	SinkErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gtc",
			Name:      "sink_errors_total",
			Help:      "Total number of sink processing errors",
		},
		[]string{"sink", "error_type"},
	)

	CircuitBreakerState = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "gtc",
			Name:      "circuit_breaker_state",
			Help:      "Circuit breaker state (0=closed, 1=half-open, 2=open)",
		},
		[]string{"sink"},
	)

	WALLagBytes = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "gtc",
			Name:      "wal_lag_bytes",
			Help:      "Replication lag in bytes",
		},
	)

	ActiveSinks = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "gtc",
			Name:      "active_sinks",
			Help:      "Number of active sinks",
		},
	)

	RetryAttempts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gtc",
			Name:      "retry_attempts_total",
			Help:      "Total retry attempts per sink",
		},
		[]string{"sink"},
	)

	EventsExcluded = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gtc",
			Name:      "events_excluded_total",
			Help:      "Total number of events excluded by table filter",
		},
		[]string{"schema", "table"},
	)

	InFlightEvents = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "gtc",
			Name:      "inflight_events",
			Help:      "Number of events currently being processed",
		},
	)

	LastProcessedLSN = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "gtc",
			Name:      "last_processed_lsn",
			Help:      "Last processed LSN as uint64",
		},
	)

	ReplicationSlotActive = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "gtc",
			Name:      "replication_slot_active",
			Help:      "Replication slot activity state (1=active, 0=inactive)",
		},
		[]string{"slot"},
	)

	ReplicationSlotLagBytes = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "gtc",
			Name:      "replication_slot_lag_bytes",
			Help:      "Replication slot lag in bytes based on current WAL vs restart_lsn",
		},
		[]string{"slot"},
	)

	CheckpointLSNBytes = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "gtc",
			Name:      "checkpoint_lsn_bytes",
			Help:      "Last durably persisted checkpoint LSN as a uint64 byte offset",
		},
	)
)
