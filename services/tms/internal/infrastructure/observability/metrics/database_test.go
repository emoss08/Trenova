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
