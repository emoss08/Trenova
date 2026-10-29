package httpmetrics

import (
	"github.com/emoss08/trenova/internal/infrastructure/observability/metricbase"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

const (
	RateLimitOutcomeAllowed = "allowed"
	RateLimitOutcomeLimited = "limited"
)

type RateLimit struct {
	metricbase.Base
	decisionsTotal     *prometheus.CounterVec
	storeFailuresTotal *prometheus.CounterVec
}

func NewRateLimit(registry *prometheus.Registry, logger *zap.Logger, enabled bool) *RateLimit {
	m := &RateLimit{
		Base: metricbase.NewBase(registry, logger, enabled),
	}

	if !enabled {
		return m
	}

	m.decisionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			metricbase.Namespace: metricbase.Namespace,
			Subsystem:            "ratelimit",
			Name:                 "decisions_total",
			Help:                 "Rate limit decisions by scope and outcome",
		},
		[]string{"scope", "outcome"},
	)

	m.storeFailuresTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			metricbase.Namespace: metricbase.Namespace,
			Subsystem:            "ratelimit",
			Name:                 "store_failures_total",
			Help:                 "Rate limit store failures by the failure mode that was applied",
		},
		[]string{"failure_mode"},
	)

	m.MustRegister(m.decisionsTotal, m.storeFailuresTotal)

	return m
}

func (m *RateLimit) RecordDecision(scope string, allowed bool) {
	m.IfEnabled(func() {
		outcome := RateLimitOutcomeAllowed
		if !allowed {
			outcome = RateLimitOutcomeLimited
		}
		m.decisionsTotal.WithLabelValues(scope, outcome).Inc()
	})
}

func (m *RateLimit) RecordStoreFailure(failureMode string) {
	m.IfEnabled(func() {
		m.storeFailuresTotal.WithLabelValues(failureMode).Inc()
	})
}
