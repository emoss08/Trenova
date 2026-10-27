//revive:disable-next-line:var-naming
package metrics

import (
	"testing"

	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewDatabase_Disabled(t *testing.T) {
	t.Parallel()

	m := NewDatabase(nil, zap.NewNop(), false)

	require.NotNil(t, m)
	assert.False(t, m.IsEnabled())
	assert.Nil(t, m.concurrencyTotal)
}

func TestDatabase_RecordConcurrencyEvent(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := NewDatabase(registry, zap.NewNop(), true)

	m.RecordConcurrencyEvent(dberror.ConcurrencyEvent{
		Kind:   "version_mismatch",
		Entity: "Shipment",
		Code:   "",
	})

	families, err := registry.Gather()
	require.NoError(t, err)

	var found bool
	for _, family := range families {
		if family.GetName() != "trenova_db_concurrency_total" {
			continue
		}

		found = true
		require.Len(t, family.Metric, 1)

		labels := map[string]string{}
		for _, label := range family.Metric[0].Label {
			labels[label.GetName()] = label.GetValue()
		}

		assert.Equal(t, "version_mismatch", labels["kind"])
		assert.Equal(t, "shipment", labels["entity"])
		assert.Equal(t, "unknown", labels["code"])
		assert.Equal(t, float64(1), family.Metric[0].GetCounter().GetValue())
	}

	assert.True(t, found)
}

func TestDatabase_RegisterRuntimeStats(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := NewDatabase(registry, zap.NewNop(), true)
	m.RegisterRuntimeStats(func() RuntimeStats {
		return RuntimeStats{
			Cache: ResultCacheStats{
				Hits:      40,
				Misses:    9,
				Bypasses:  3,
				Errors:    1,
				Bytes:     4096,
				Versions:  12,
				Listening: true,
			},
			TenantWaits:        7,
			ReadCommitFailures: 2,
		}
	})

	families, err := registry.Gather()
	require.NoError(t, err)

	values := map[string]float64{}
	for _, family := range families {
		require.Len(t, family.Metric, 1)
		metric := family.Metric[0]
		switch {
		case metric.GetCounter() != nil:
			values[family.GetName()] = metric.GetCounter().GetValue()
		case metric.GetGauge() != nil:
			values[family.GetName()] = metric.GetGauge().GetValue()
		}
	}

	assert.Equal(t, map[string]float64{
		"trenova_db_cache_hits_total":                40,
		"trenova_db_cache_misses_total":              9,
		"trenova_db_cache_bypasses_total":            3,
		"trenova_db_cache_errors_total":              1,
		"trenova_db_cache_bytes":                     4096,
		"trenova_db_cache_versions":                  12,
		"trenova_db_cache_listening":                 1,
		"trenova_db_pool_tenant_waits_total":         7,
		"trenova_db_pool_read_commit_failures_total": 2,
	}, values)
}

func TestDatabase_RegisterRuntimeStatsDisabled(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := NewDatabase(registry, zap.NewNop(), false)
	m.RegisterRuntimeStats(func() RuntimeStats { return RuntimeStats{} })

	families, err := registry.Gather()
	require.NoError(t, err)
	assert.Empty(t, families)
}
