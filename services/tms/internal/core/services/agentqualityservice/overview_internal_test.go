package agentqualityservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/stretchr/testify/assert"
)

func scored(score, baseline float64, regression bool) *agentquality.SuiteRun {
	return &agentquality.SuiteRun{
		QualityScore:  &score,
		BaselineScore: &baseline,
		Regression:    regression,
	}
}

func TestWorstRegressionIsTheOpenOneThatFellFurthest(t *testing.T) {
	t.Parallel()

	small := scored(0.80, 0.86, true)
	large := scored(0.60, 0.90, true)
	fellButNotRegressed := scored(0.10, 0.95, false)

	assert.Same(t, large, worstRegression([]*agentquality.SuiteRun{small, fellButNotRegressed, large}))
}

func TestWorstRegressionTakesOneWithoutABaselineWhenItIsTheOnlyOne(t *testing.T) {
	t.Parallel()

	score := 0.5
	unmeasured := &agentquality.SuiteRun{QualityScore: &score, Regression: true}

	assert.Same(t, unmeasured, worstRegression([]*agentquality.SuiteRun{nil, unmeasured}))
	assert.Nil(t, worstRegression([]*agentquality.SuiteRun{scored(0.9, 0.8, false)}))
	assert.Nil(t, worstRegression(nil))
}
