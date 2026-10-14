//revive:disable-next-line:var-naming
package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestFormula_DisabledAndNilRecordsAreNoOps(t *testing.T) {
	t.Parallel()

	for _, m := range []*Formula{nil, NewFormula(nil, zap.NewNop(), false)} {
		assert.NotPanics(t, func() {
			m.EvaluationStarted()
			m.EvaluationAbandoned()
			m.RecordTimeout(FormulaTimeoutStageRunning)
			m.AbandonedEvaluationFinished()
			m.EvaluationFinished()
		})
	}
}

func TestFormula_TracksSlotsAbandonmentAndTimeouts(t *testing.T) {
	t.Parallel()

	m := NewFormula(prometheus.NewRegistry(), zap.NewNop(), true)

	m.EvaluationStarted()
	m.EvaluationStarted()
	m.EvaluationAbandoned()
	m.RecordTimeout(FormulaTimeoutStageRunning)
	m.RecordTimeout(FormulaTimeoutStageQueued)
	m.EvaluationFinished()

	assert.InDelta(t, 1, testutil.ToFloat64(m.inFlight), 0)
	assert.InDelta(t, 1, testutil.ToFloat64(m.abandoned), 0)
	assert.InDelta(t, 1,
		testutil.ToFloat64(m.timeouts.WithLabelValues(FormulaTimeoutStageRunning)), 0)
	assert.InDelta(t, 1,
		testutil.ToFloat64(m.timeouts.WithLabelValues(FormulaTimeoutStageQueued)), 0)

	m.AbandonedEvaluationFinished()
	m.EvaluationFinished()

	assert.InDelta(t, 0, testutil.ToFloat64(m.inFlight), 0)
	assert.InDelta(t, 0, testutil.ToFloat64(m.abandoned), 0)
}
