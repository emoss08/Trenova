//revive:disable-next-line:var-naming
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

const toolLabel = "tool"

// ProposalPreview measures the previews a person approves against: how many
// each tool produces at each coverage, how long they take, and how often an
// approval is refused because what was shown no longer matches. A rising
// conflict rate is how an unstable digest shows itself.
type ProposalPreview struct {
	Base
	previews  *prometheus.CounterVec
	duration  *prometheus.HistogramVec
	conflicts *prometheus.CounterVec
}

func NewProposalPreview(
	registry *prometheus.Registry,
	logger *zap.Logger,
	enabled bool,
) *ProposalPreview {
	m := &ProposalPreview{Base: NewBase(registry, logger, enabled)}
	if !enabled {
		return m
	}

	m.previews = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: "ai",
		Name:      "proposal_preview_total",
		Help:      "Proposal previews computed, by tool and coverage",
	}, []string{toolLabel, "coverage"})

	m.duration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: Namespace,
		Subsystem: "ai",
		Name:      "proposal_preview_duration_seconds",
		Help:      "Time to compute one proposal preview, by tool",
		Buckets:   []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 3, 5},
	}, []string{toolLabel})

	m.conflicts = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: "ai",
		Name:      "proposal_preview_conflicts_total",
		Help:      "Approvals refused because the preview digest no longer matched, by tool",
	}, []string{toolLabel})

	m.mustRegister(m.previews, m.duration, m.conflicts)

	return m
}

func (m *ProposalPreview) RecordPreview(tool, coverage string, seconds float64) {
	if m == nil || !m.IsEnabled() {
		return
	}
	m.previews.WithLabelValues(tool, coverage).Inc()
	m.duration.WithLabelValues(tool).Observe(seconds)
}

func (m *ProposalPreview) RecordConflict(tool string) {
	if m == nil || !m.IsEnabled() {
		return
	}
	m.conflicts.WithLabelValues(tool).Inc()
}
