package agentscoring

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestScoreSuite_WeighsThumbsUpLess(t *testing.T) {
	t.Parallel()

	suite := ScoreSuite([]CaseResult{
		{Source: agentquality.CaseSourceCurated, Score: 1, Passed: true},
		{Source: agentquality.CaseSourceDecidedProposal, Score: 1, Passed: true},
		{Source: agentquality.CaseSourceThumbsUp, Score: 0, HardFailure: true},
	})

	assert.InDelta(t, 2.0/2.6, suite.Score, 1e-9)
	assert.Equal(t, 3, suite.Cases)
	assert.Equal(t, 2, suite.Passed)
	assert.Equal(t, 1, suite.Failed)
	assert.Equal(t, 1, suite.HardFailures)
	assert.InDelta(t, 2.6, suite.Weight, 1e-9)
	assert.Zero(t, ScoreSuite(nil).Score)
}

func history(scores ...float64) []SuiteRun {
	runs := make([]SuiteRun, 0, len(scores))
	for i, score := range scores {
		runs = append(runs, SuiteRun{Score: score, Cases: 20, CompletedAt: int64(1000 + i)})
	}

	return runs
}

func TestDetectRegression_AgainstTheMedianOfTheLastSeven(t *testing.T) {
	t.Parallel()

	runs := history(0.10, 0.10, 0.90, 0.91, 0.92, 0.93, 0.90, 0.94, 0.90)
	policy := DefaultRegressionPolicy()

	dropped := DetectRegression(RegressionInput{
		Current: Suite{Score: 0.84, Cases: 20},
		History: runs,
	}, policy)
	assert.Equal(t, 7, dropped.Compared, "only the last seven completed runs count")
	assert.InDelta(t, 0.91, dropped.Median, 1e-9, "the two old outliers fell out of the window")
	assert.True(t, dropped.ScoreDropped)
	assert.True(t, dropped.Regressed)

	steady := DetectRegression(RegressionInput{
		Current: Suite{Score: 0.87, Cases: 20},
		History: runs,
	}, policy)
	assert.False(t, steady.ScoreDropped, "a fall within the threshold is noise")
	assert.False(t, steady.Regressed)
}

func TestDetectRegression_NeedsEnoughCases(t *testing.T) {
	t.Parallel()

	result := DetectRegression(RegressionInput{
		Current: Suite{Score: 0.2, Cases: 4},
		History: history(0.95, 0.95, 0.95),
	}, DefaultRegressionPolicy())

	assert.False(t, result.ScoreDropped)
	assert.False(t, result.Regressed)
}

func TestDetectRegression_NoHistoryNoScoreRegression(t *testing.T) {
	t.Parallel()

	result := DetectRegression(RegressionInput{
		Current: Suite{Score: 0.1, Cases: 50},
	}, DefaultRegressionPolicy())

	assert.Zero(t, result.Compared)
	assert.False(t, result.Regressed)
}

func TestDetectRegression_ACasePassedAtBaselineNowFailsHard(t *testing.T) {
	t.Parallel()

	steady := pulid.MustNew("aec_")
	broke := pulid.MustNew("aec_")
	alreadyBroken := pulid.MustNew("aec_")

	result := DetectRegression(RegressionInput{
		Current: Suite{Score: 0.95, Cases: 3},
		Results: []CaseResult{
			{CaseID: steady, Score: 1, Passed: true},
			{CaseID: broke, HardFailure: true},
			{CaseID: alreadyBroken, HardFailure: true},
		},
		History: history(0.95, 0.95),
		Baseline: []CaseResult{
			{CaseID: steady, Score: 1, Passed: true},
			{CaseID: broke, Score: 1, Passed: true},
			{CaseID: alreadyBroken, HardFailure: true},
		},
	}, DefaultRegressionPolicy())

	assert.True(t, result.Regressed, "a hard failure regresses whatever the score")
	assert.False(t, result.ScoreDropped)
	assert.Equal(t, []pulid.ID{broke}, result.NewHardFailures)
}
