//revive:disable-next-line:var-naming
package metrics

import (
	"regexp"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

const (
	AnonymousOperation = "anonymous"
	OverflowOperation  = "other"
)

var (
	GraphQLDurationBuckets = []float64{
		.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10,
	}
	GraphQLResolverDurationBuckets = []float64{
		.0005, .001, .005, .01, .025, .05, .1, .25, .5, 1,
	}
	GraphQLPhaseDurationBuckets = []float64{
		.0001, .00025, .0005, .001, .0025, .005, .01, .025, .05,
	}
	GraphQLCostBuckets = prometheus.ExponentialBuckets(10, 4, 10)
)

var operationNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,63}$`)

type GraphQLOptions struct {
	ResolverMetrics bool
	MaxOperations   int
}

type RecordOperationParams struct {
	Operation string
	Type      string
	ErrorCode string
	Duration  float64
	Errors    int
}

type RecordResolverParams struct {
	Object   string
	Field    string
	Duration float64
	Failed   bool
}

type GraphQL struct {
	Base
	operationsTotal    *prometheus.CounterVec
	operationDuration  *prometheus.HistogramVec
	operationErrors    *prometheus.CounterVec
	operationCost      *prometheus.HistogramVec
	activeOperations   prometheus.Gauge
	rejectionsTotal    *prometheus.CounterVec
	resolverCalls      *prometheus.CounterVec
	resolverDuration   *prometheus.HistogramVec
	resolverErrors     *prometheus.CounterVec
	parseDuration      prometheus.Histogram
	validationDuration prometheus.Histogram

	resolverMetrics bool
	maxOperations   int
	mu              sync.RWMutex
	trackedOps      map[string]struct{}
}

func NewGraphQL(
	registry *prometheus.Registry,
	logger *zap.Logger,
	enabled bool,
	opts GraphQLOptions,
) *GraphQL {
	m := &GraphQL{
		Base:            NewBase(registry, logger, enabled),
		resolverMetrics: opts.ResolverMetrics,
		maxOperations:   opts.MaxOperations,
		trackedOps:      make(map[string]struct{}, 0),
	}

	if !enabled {
		return m
	}

	m.operationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "graphql",
			Name:      "operations_total",
			Help:      "Total number of GraphQL operations executed",
		},
		[]string{"operation", "type", "status"},
	)

	m.operationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: "graphql",
			Name:      "operation_duration_seconds",
			Help:      "GraphQL operation execution latencies in seconds",
			Buckets:   GraphQLDurationBuckets,
		},
		[]string{"operation", "type"},
	)

	m.operationErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "graphql",
			Name:      "operation_errors_total",
			Help:      "Total number of GraphQL operations that returned errors, by error code",
		},
		[]string{"operation", "type", "code"},
	)

	m.activeOperations = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: "graphql",
			Name:      "active_operations",
			Help:      "Number of GraphQL operations currently executing",
		},
	)

	m.operationCost = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: "graphql",
			Name:      "operation_cost",
			Help:      "Computed GraphQL query complexity per operation",
			Buckets:   GraphQLCostBuckets,
		},
		[]string{"operation", "type"},
	)

	m.rejectionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "graphql",
			Name:      "rejections_total",
			Help:      "Total number of GraphQL requests rejected before execution",
		},
		[]string{"reason"},
	)

	m.parseDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: "graphql",
			Name:      "parse_duration_seconds",
			Help:      "Time spent parsing GraphQL documents in seconds",
			Buckets:   GraphQLPhaseDurationBuckets,
		},
	)

	m.validationDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: "graphql",
			Name:      "validation_duration_seconds",
			Help:      "Time spent validating GraphQL documents in seconds",
			Buckets:   GraphQLPhaseDurationBuckets,
		},
	)

	m.mustRegister(
		m.operationsTotal,
		m.operationDuration,
		m.operationErrors,
		m.operationCost,
		m.activeOperations,
		m.rejectionsTotal,
		m.parseDuration,
		m.validationDuration,
	)

	if m.resolverMetrics {
		m.registerResolverCollectors()
	}

	return m
}

func (m *GraphQL) registerResolverCollectors() {
	m.resolverCalls = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "graphql",
			Name:      "resolver_calls_total",
			Help:      "Total number of GraphQL resolver invocations, for N+1 detection",
		},
		[]string{"object", "field"},
	)

	m.resolverDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: "graphql",
			Name:      "resolver_duration_seconds",
			Help:      "GraphQL resolver latencies in seconds",
			Buckets:   GraphQLResolverDurationBuckets,
		},
		[]string{"object", "field"},
	)

	m.resolverErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "graphql",
			Name:      "resolver_errors_total",
			Help:      "Total number of GraphQL resolver invocations that returned an error",
		},
		[]string{"object", "field"},
	)

	m.mustRegister(m.resolverCalls, m.resolverDuration, m.resolverErrors)
}

func (m *GraphQL) ResolverMetricsEnabled() bool {
	return m.IsEnabled() && m.resolverMetrics
}

func (m *GraphQL) BoundOperationName(name string) string {
	if name == "" {
		return AnonymousOperation
	}
	if !operationNamePattern.MatchString(name) {
		return UnknownValue
	}

	m.mu.RLock()
	_, tracked := m.trackedOps[name]
	m.mu.RUnlock()
	if tracked {
		return name
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.trackedOps[name]; ok {
		return name
	}
	if len(m.trackedOps) >= m.maxOperations {
		return OverflowOperation
	}
	m.trackedOps[name] = struct{}{}

	return name
}

func (m *GraphQL) RecordOperation(p RecordOperationParams) {
	if !m.IsEnabled() {
		return
	}

	status := metricStatusSuccess
	if p.Errors > 0 {
		status = metricStatusFailure
	}

	m.operationsTotal.WithLabelValues(p.Operation, p.Type, status).Inc()
	m.operationDuration.WithLabelValues(p.Operation, p.Type).Observe(p.Duration)

	if p.Errors > 0 {
		m.operationErrors.WithLabelValues(p.Operation, p.Type, p.ErrorCode).Inc()
	}
}

func (m *GraphQL) RecordResolver(p RecordResolverParams) {
	if !m.ResolverMetricsEnabled() {
		return
	}

	m.resolverCalls.WithLabelValues(p.Object, p.Field).Inc()
	m.resolverDuration.WithLabelValues(p.Object, p.Field).Observe(p.Duration)
	if p.Failed {
		m.resolverErrors.WithLabelValues(p.Object, p.Field).Inc()
	}
}

func (m *GraphQL) RecordParsePhases(parse, validation float64) {
	m.ifEnabled(func() {
		m.parseDuration.Observe(parse)
		m.validationDuration.Observe(validation)
	})
}

func (m *GraphQL) RecordOperationCost(operation, operationType string, cost float64) {
	m.ifEnabled(func() {
		m.operationCost.WithLabelValues(operation, operationType).Observe(cost)
	})
}

func (m *GraphQL) RecordRejection(reason string) {
	m.ifEnabled(func() { m.rejectionsTotal.WithLabelValues(reason).Inc() })
}

func (m *GraphQL) IncrementActiveOperations() {
	m.ifEnabled(func() { m.activeOperations.Inc() })
}

func (m *GraphQL) DecrementActiveOperations() {
	m.ifEnabled(func() { m.activeOperations.Dec() })
}
