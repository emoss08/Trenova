package platformmetrics

import (
	"github.com/emoss08/trenova/internal/infrastructure/observability/metricbase"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

const (
	tenancySubsystem = "tenancy"

	LimitScopeOrganization = "organization"
	LimitScopeInstance     = "instance"
)

type Tenancy struct {
	metricbase.Base
	entitlementRefusals *prometheus.CounterVec
	limitRefusals       *prometheus.CounterVec
	slotWaits           *prometheus.CounterVec
	archiveRefusals     *prometheus.CounterVec
}

func NewTenancy(registry *prometheus.Registry, logger *zap.Logger, enabled bool) *Tenancy {
	m := &Tenancy{Base: metricbase.NewBase(registry, logger, enabled)}
	if !enabled {
		return m
	}

	m.entitlementRefusals = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricbase.Namespace,
			Subsystem: tenancySubsystem,
			Name:      "entitlement_refusals_total",
			Help:      "Activities refused because the tenant's plan lacks the feature their schedule belongs to, or the tenant is suspended; any is a filter that missed",
		},
		[]string{"activity_type", "feature"},
	)
	m.limitRefusals = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricbase.Namespace,
			Subsystem: tenancySubsystem,
			Name:      "limit_refusals_total",
			Help:      "Requests and jobs refused or held back by a per-organization limit or the instance cap behind it",
		},
		[]string{"limit", "scope"},
	)
	m.slotWaits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricbase.Namespace,
			Subsystem: tenancySubsystem,
			Name:      "slot_waits_total",
			Help:      "Heavy jobs that found no free slot and waited in their queue",
		},
		[]string{rollupLabelKind, "scope"},
	)
	m.archiveRefusals = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricbase.Namespace,
			Subsystem: tenancySubsystem,
			Name:      "archive_refusals_total",
			Help:      "Tenant archives refused, by purpose and refusal code; code tenant on a nightly archive is a row that refers to another tenant",
		},
		[]string{"purpose", "code"},
	)
	m.MustRegister(m.entitlementRefusals, m.limitRefusals, m.slotWaits, m.archiveRefusals)
	return m
}

func (m *Tenancy) RecordEntitlementRefusal(
	activityType string,
	feature platformcatalog.FeatureKey,
) {
	if m == nil {
		return
	}
	m.IfEnabled(func() {
		m.entitlementRefusals.WithLabelValues(activityType, string(feature)).Inc()
	})
}

func (m *Tenancy) RecordLimitRefusal(limit, scope string) {
	if m == nil {
		return
	}
	m.IfEnabled(func() { m.limitRefusals.WithLabelValues(limit, scope).Inc() })
}

func (m *Tenancy) RecordSlotWait(kind, scope string) {
	if m == nil {
		return
	}
	m.IfEnabled(func() { m.slotWaits.WithLabelValues(kind, scope).Inc() })
}

func (m *Tenancy) RecordArchiveRefusal(purpose, code string) {
	if m == nil {
		return
	}
	m.IfEnabled(func() { m.archiveRefusals.WithLabelValues(purpose, code).Inc() })
}
