package ai

import (
	"bytes"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/aitraining"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func scoreReport(model, baseline float64) *aitraining.ScoreReport {
	return &aitraining.ScoreReport{
		Examples:      10,
		Model:         aitraining.ScoreSide{Accuracy: model},
		Baseline:      aitraining.ScoreSide{Accuracy: baseline},
		AccuracyDelta: model - baseline,
	}
}

func TestScoreGate(t *testing.T) {
	t.Parallel()

	require.NoError(t, scoreGate(scoreReport(0.91, 0.90), 0, 0))
	require.NoError(t, scoreGate(scoreReport(0.895, 0.90), 0, 0.01))
	require.ErrorIs(t, scoreGate(scoreReport(0.88, 0.90), 0, 0.01), errScoreGate)
	require.ErrorIs(t, scoreGate(scoreReport(0.91, 0.90), 0.95, 0), errScoreGate)
	require.ErrorIs(t, scoreGate(&aitraining.ScoreReport{}, 0, 0), errScoreGate)
}

func TestPrintScoreListsEveryFieldOnBothSides(t *testing.T) {
	t.Parallel()

	report := scoreReport(0.9, 0.8)
	report.Model.Fields = []aicorrection.FieldAccuracy{{Key: "rate", Accuracy: 0.9, Scored: 10}}
	report.Baseline.Fields = []aicorrection.FieldAccuracy{
		{Key: "rate", Accuracy: 0.8, Scored: 10},
		{Key: "shipper", Accuracy: 0.5, Scored: 4},
	}

	var out bytes.Buffer
	printScore(&out, report)
	assert.Contains(t, out.String(), "+10.00 points")
	assert.Contains(t, out.String(), "rate")
	assert.Contains(t, out.String(), "shipper")
}
