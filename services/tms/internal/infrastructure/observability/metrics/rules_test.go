//revive:disable-next-line:var-naming
package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestRules_DisabledAndNilRecordsAreNoOps(t *testing.T) {
	t.Parallel()

	for _, m := range []*Rules{nil, NewRules(nil, zap.NewNop(), false)} {
		assert.NotPanics(t, func() {
			m.InstanceAcquired()
			m.CallFinished("rules.version", "ok", time.Millisecond)
			m.InstanceDiscarded("trap")
			m.InstanceReleased()
		})
	}
}

func TestRules_CountsCallsInstancesAndDiscards(t *testing.T) {
	t.Parallel()

	m := NewRules(prometheus.NewRegistry(), zap.NewNop(), true)

	m.InstanceAcquired()
	m.InstanceAcquired()
	m.CallFinished("formula.check", "ok", 2*time.Millisecond)
	m.CallFinished("formula.check", "ok", 3*time.Millisecond)
	m.CallFinished("formula.evaluate", "trap", time.Millisecond)
	m.InstanceDiscarded("trap")
	m.InstanceReleased()

	assert.InDelta(t, 2, testutil.ToFloat64(m.calls.WithLabelValues("formula.check", "ok")), 0)
	assert.InDelta(t, 1, testutil.ToFloat64(m.calls.WithLabelValues("formula.evaluate", "trap")), 0)
	assert.Equal(t, 2, testutil.CollectAndCount(m.duration))
	assert.InDelta(t, 1, testutil.ToFloat64(m.inUse), 0)
	assert.InDelta(t, 1, testutil.ToFloat64(m.discarded.WithLabelValues("trap")), 0)

	m.InstanceReleased()
	assert.InDelta(t, 0, testutil.ToFloat64(m.inUse), 0)
}
