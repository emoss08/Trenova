//revive:disable-next-line:var-naming
package metrics

import (
	"strconv"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestGraphQL(t *testing.T, enabled, resolverMetrics bool) (*GraphQL, *prometheus.Registry) {
	t.Helper()

	var registry *prometheus.Registry
	if enabled {
		registry = prometheus.NewRegistry()
	}

	return NewGraphQL(registry, zap.NewNop(), enabled, GraphQLOptions{
		ResolverMetrics: resolverMetrics,
		MaxOperations:   defaultTestMaxOperations,
	}), registry
}

const defaultTestMaxOperations = 3

func TestNewGraphQL_Disabled(t *testing.T) {
	t.Parallel()

	m, _ := newTestGraphQL(t, false, true)

	require.NotNil(t, m)
	assert.False(t, m.IsEnabled())
	assert.False(t, m.ResolverMetricsEnabled())
	assert.Nil(t, m.operationsTotal)
	assert.Nil(t, m.resolverCalls)

	assert.NotPanics(t, func() {
		m.RecordOperation(RecordOperationParams{Operation: "Q", Type: "query", Duration: 1})
		m.RecordResolver(RecordResolverParams{Object: "Query", Field: "f"})
		m.RecordParsePhases(0.1, 0.2)
		m.IncrementActiveOperations()
		m.DecrementActiveOperations()
	})
}

func TestNewGraphQL_ResolverMetricsOptional(t *testing.T) {
	t.Parallel()

	off, _ := newTestGraphQL(t, true, false)
	assert.True(t, off.IsEnabled())
	assert.False(t, off.ResolverMetricsEnabled())
	assert.Nil(t, off.resolverCalls)

	on, _ := newTestGraphQL(t, true, true)
	assert.True(t, on.ResolverMetricsEnabled())
	assert.NotNil(t, on.resolverCalls)
}

func TestGraphQL_RecordOperation(t *testing.T) {
	t.Parallel()

	m, registry := newTestGraphQL(t, true, false)

	m.RecordOperation(RecordOperationParams{
		Operation: "ListShipments",
		Type:      "query",
		Duration:  0.25,
	})
	m.RecordOperation(RecordOperationParams{
		Operation: "ListShipments",
		Type:      "query",
		ErrorCode: "Invalid",
		Duration:  0.1,
		Errors:    2,
	})

	gathered, err := registry.Gather()
	require.NoError(t, err)

	names := map[string]bool{}
	for _, mf := range gathered {
		names[mf.GetName()] = true
	}
	assert.True(t, names["trenova_graphql_operations_total"])
	assert.True(t, names["trenova_graphql_operation_duration_seconds"])
	assert.True(t, names["trenova_graphql_operation_errors_total"])
}

func TestGraphQL_RecordResolver_SkippedWhenDisabled(t *testing.T) {
	t.Parallel()

	m, registry := newTestGraphQL(t, true, false)

	assert.NotPanics(t, func() {
		m.RecordResolver(RecordResolverParams{Object: "Query", Field: "shipments"})
	})

	gathered, err := registry.Gather()
	require.NoError(t, err)
	for _, mf := range gathered {
		assert.NotEqual(t, "trenova_graphql_resolver_calls_total", mf.GetName())
	}
}

func TestGraphQL_RecordResolver(t *testing.T) {
	t.Parallel()

	m, registry := newTestGraphQL(t, true, true)

	m.RecordResolver(RecordResolverParams{
		Object:   "CustomerPayment",
		Field:    "customer",
		Duration: 0.01,
		Failed:   true,
	})

	gathered, err := registry.Gather()
	require.NoError(t, err)

	names := map[string]bool{}
	for _, mf := range gathered {
		names[mf.GetName()] = true
	}
	assert.True(t, names["trenova_graphql_resolver_calls_total"])
	assert.True(t, names["trenova_graphql_resolver_duration_seconds"])
	assert.True(t, names["trenova_graphql_resolver_errors_total"])
}

func TestGraphQL_BoundOperationName(t *testing.T) {
	t.Parallel()

	m, _ := newTestGraphQL(t, true, false)

	assert.Equal(t, AnonymousOperation, m.BoundOperationName(""))
	assert.Equal(t, UnknownValue, m.BoundOperationName("not a name"))
	assert.Equal(t, UnknownValue, m.BoundOperationName("9StartsWithDigit"))
	assert.Equal(t, "ListShipments", m.BoundOperationName("ListShipments"))
	assert.Equal(t, "ListShipments", m.BoundOperationName("ListShipments"))
}

func TestGraphQL_BoundOperationName_CapsCardinality(t *testing.T) {
	t.Parallel()

	m, _ := newTestGraphQL(t, true, false)

	for i := range defaultTestMaxOperations {
		name := "Op" + strconv.Itoa(i)
		assert.Equal(t, name, m.BoundOperationName(name))
	}

	assert.Equal(t, OverflowOperation, m.BoundOperationName("OneTooMany"))
	assert.Equal(t, "Op0", m.BoundOperationName("Op0"))
}

func TestGraphQL_BoundOperationName_Concurrent(t *testing.T) {
	t.Parallel()

	m := NewGraphQL(prometheus.NewRegistry(), zap.NewNop(), true, GraphQLOptions{
		MaxOperations: 50,
	})

	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.BoundOperationName("Op" + strconv.Itoa(i%60))
		}()
	}
	wg.Wait()

	m.mu.RLock()
	defer m.mu.RUnlock()
	assert.LessOrEqual(t, len(m.trackedOps), 50)
}
