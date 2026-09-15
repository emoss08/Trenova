//revive:disable-next-line:var-naming
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

const (
	RateLimitOutcomeAllowed = "allowed"
	RateLimitOutcomeLimited = "limited"
)

type RateLimit struct {
	Base
	decisionsTotal     *prometheus.CounterVec
	storeFailuresTotal *prometheus.CounterVec
}

func NewRateLimit(registry *prometheus.Registry, logger *zap.Logger, enabled bool) *RateLimit {
	m := &RateLimit{
		Base: NewBase(registry, logger, enabled),
	}

	if !enabled {
		return m
	}

	m.decisionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "ratelimit",
			Name:      "decisions_total",
			Help:      "Rate limit decisions by scope and outcome",
		},
		[]string{"scope", "outcome"},
	)

	m.storeFailuresTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "ratelimit",
			Name:      "store_failures_total",
			Help:      "Rate limit store failures by the failure mode that was applied",
		},
		[]string{"failure_mode"},
	)

	m.mustRegister(m.decisionsTotal, m.storeFailuresTotal)

	return m
}

func (m *RateLimit) RecordDecision(scope string, allowed bool) {
	m.ifEnabled(func() {
		outcome := RateLimitOutcomeAllowed
		if !allowed {
			outcome = RateLimitOutcomeLimited
		}
		m.decisionsTotal.WithLabelValues(scope, outcome).Inc()
	})
}

func (m *RateLimit) RecordStoreFailure(failureMode string) {
	m.ifEnabled(func() {
		m.storeFailuresTotal.WithLabelValues(failureMode).Inc()
	})
}
