//revive:disable-next-line:var-naming
package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewPermission_Disabled(t *testing.T) {
	t.Parallel()

	m := NewPermission(nil, zap.NewNop(), false)

	require.NotNil(t, m)
	assert.False(t, m.IsEnabled())
}

func TestPermission_Disabled_RecordsAreNoOps(t *testing.T) {
	t.Parallel()

	m := NewPermission(nil, zap.NewNop(), false)

	assert.NotPanics(t, func() {
		m.RecordCacheLookup(PermissionCacheHit, time.Millisecond)
		m.RecordCacheLookup(PermissionCacheMiss, 0)
		m.RecordCompute(PermissionPathCached, time.Millisecond)
		m.RecordClosureSize(3)
		m.RecordClosureSize(0)
	})
}

func TestNewPermission_Enabled(t *testing.T) {
	t.Parallel()

	m := NewPermission(prometheus.NewRegistry(), zap.NewNop(), true)

	require.NotNil(t, m)
	assert.True(t, m.IsEnabled())
	assert.NotNil(t, m.cacheLookupsTotal)
	assert.NotNil(t, m.computeTotal)
	assert.NotNil(t, m.computeDuration)
	assert.NotNil(t, m.cacheLookupDuration)
	assert.NotNil(t, m.closureRoles)
}

func TestPermission_RecordCacheLookup_CountsEachOutcomeSeparately(t *testing.T) {
	t.Parallel()

	m := NewPermission(prometheus.NewRegistry(), zap.NewNop(), true)

	m.RecordCacheLookup(PermissionCacheHit, 2*time.Millisecond)
	m.RecordCacheLookup(PermissionCacheHit, 2*time.Millisecond)
	m.RecordCacheLookup(PermissionCacheMiss, 3*time.Millisecond)
	m.RecordCacheLookup(PermissionCacheError, time.Millisecond)

	assert.InDelta(t, 2.0, counterValue(t, m.cacheLookupsTotal, PermissionCacheHit), 0.0001)
	assert.InDelta(t, 1.0, counterValue(t, m.cacheLookupsTotal, PermissionCacheMiss), 0.0001)
	assert.InDelta(t, 1.0, counterValue(t, m.cacheLookupsTotal, PermissionCacheError), 0.0001)
}

func TestPermission_RecordCompute_AttributesToThePath(t *testing.T) {
	t.Parallel()

	m := NewPermission(prometheus.NewRegistry(), zap.NewNop(), true)

	m.RecordCompute(PermissionPathCached, 5*time.Millisecond)
	m.RecordCompute(PermissionPathEffective, 40*time.Millisecond)
	m.RecordCompute(PermissionPathEffective, 40*time.Millisecond)
	m.RecordCompute(PermissionPathSimulation, 50*time.Millisecond)

	assert.InDelta(t, 1.0, counterValue(t, m.computeTotal, PermissionPathCached), 0.0001)
	assert.InDelta(t, 2.0, counterValue(t, m.computeTotal, PermissionPathEffective), 0.0001)
	assert.InDelta(t, 1.0, counterValue(t, m.computeTotal, PermissionPathSimulation), 0.0001)
}

func TestNewPermission_ExposesMetricsUnderTheExpectedNames(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := NewPermission(registry, zap.NewNop(), true)

	m.RecordCacheLookup(PermissionCacheHit, time.Millisecond)
	m.RecordCompute(PermissionPathCached, time.Millisecond)
	m.RecordClosureSize(2)

	for _, name := range []string{
		"trenova_permission_cache_lookups_total",
		"trenova_permission_computes_total",
		"trenova_permission_compute_duration_seconds",
		"trenova_permission_cache_lookup_duration_seconds",
		"trenova_permission_closure_roles",
	} {
		assert.Equal(t, 1, testutil.CollectAndCount(registry, name), "metric %s", name)
	}
}

func counterValue(t *testing.T, vec *prometheus.CounterVec, labels ...string) float64 {
	t.Helper()

	counter, err := vec.GetMetricWithLabelValues(labels...)
	require.NoError(t, err)

	return testutil.ToFloat64(counter)
}
