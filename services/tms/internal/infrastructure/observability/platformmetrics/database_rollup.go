package platformmetrics

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/observability/metricbase"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	rollupSubsystem = "rollup"
	rollupLabelName = "rollup"
	rollupLabelKind = "kind"
	rollupKindSlice = "slice"
	rollupKindWhole = "tenant"
)

func (m *Database) registerRollups() {
	m.rollupSlices = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricbase.Namespace,
			Subsystem: rollupSubsystem,
			Name:      "recomputed_total",
			Help:      "Rollup pieces recomputed from their source, by rollup and kind (one tenant's bucket, or a whole tenant)",
		},
		[]string{rollupLabelName, rollupLabelKind},
	)
	m.rollupRows = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricbase.Namespace,
			Subsystem: rollupSubsystem,
			Name:      "rows_written_total",
			Help:      "Rollup rows written by refreshes, by rollup",
		},
		[]string{rollupLabelName},
	)
	m.rollupPending = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricbase.Namespace,
			Subsystem: rollupSubsystem,
			Name:      "pending_marks",
			Help:      "Dirty marks waiting for a refresh after the last refresh run, by rollup",
		},
		[]string{rollupLabelName},
	)
	m.rollupDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metricbase.Namespace,
		Subsystem: rollupSubsystem,
		Name:      "refresh_duration_seconds",
		Help:      "Wall time of one rollup refresh run",
		Buckets:   prometheus.ExponentialBuckets(0.01, 2, 14),
	})
	m.MustRegister(m.rollupSlices, m.rollupRows, m.rollupPending, m.rollupDuration)
}

// RecordRollupRefresh counts what one refresh run did for one rollup.
func (m *Database) RecordRollupRefresh(rollup string, slices, tenants int, rows int64) {
	m.IfEnabled(func() {
		m.rollupSlices.WithLabelValues(rollup, rollupKindSlice).Add(float64(slices))
		m.rollupSlices.WithLabelValues(rollup, rollupKindWhole).Add(float64(tenants))
		m.rollupRows.WithLabelValues(rollup).Add(float64(rows))
	})
}

// ObserveRollupRefresh records the wall time of one refresh run.
func (m *Database) ObserveRollupRefresh(elapsed time.Duration) {
	m.IfEnabled(func() { m.rollupDuration.Observe(elapsed.Seconds()) })
}

// SetRollupPending records the backlog of a rollup once a refresh run ends.
func (m *Database) SetRollupPending(rollup string, marks int64) {
	m.IfEnabled(func() { m.rollupPending.WithLabelValues(rollup).Set(float64(marks)) })
}
