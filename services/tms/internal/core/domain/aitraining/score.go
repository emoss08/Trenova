package aitraining

import (
	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
)

type ScoreSide struct {
	Scored      int                          `json:"scored"`
	Correct     int                          `json:"correct"`
	Corrected   int                          `json:"corrected"`
	Missed      int                          `json:"missed"`
	Unconfirmed int                          `json:"unconfirmed"`
	Unscored    int                          `json:"unscored"`
	Accuracy    float64                      `json:"accuracy"`
	Fields      []aicorrection.FieldAccuracy `json:"fields"`
}

type ScoreReport struct {
	Examples              int       `json:"examples"`
	Predicted             int       `json:"predicted"`
	MissingPredictions    int       `json:"missingPredictions"`
	FailedPredictions     int       `json:"failedPredictions"`
	UnreadablePredictions int       `json:"unreadablePredictions"`
	UnknownPredictions    int       `json:"unknownPredictions"`
	Model                 ScoreSide `json:"model"`
	Baseline              ScoreSide `json:"baseline"`
	AccuracyDelta         float64   `json:"accuracyDelta"`
}

type ScoreTally struct {
	tally      aicorrection.Tally
	aggregator *aicorrection.AccuracyAggregator
}

func NewScoreTally() *ScoreTally {
	return &ScoreTally{aggregator: aicorrection.NewAccuracyAggregator()}
}

func (t *ScoreTally) Add(results []aicorrection.FieldResult) {
	add := aicorrection.TallyResults(results)
	t.tally.Scored += add.Scored
	t.tally.Correct += add.Correct
	t.tally.Corrected += add.Corrected
	t.tally.Missed += add.Missed
	t.tally.Unconfirmed += add.Unconfirmed
	t.tally.Unscored += add.Unscored
	t.aggregator.Add(results)
}

func (t *ScoreTally) Side() ScoreSide {
	return ScoreSide{
		Scored:      t.tally.Scored,
		Correct:     t.tally.Correct,
		Corrected:   t.tally.Corrected,
		Missed:      t.tally.Missed,
		Unconfirmed: t.tally.Unconfirmed,
		Unscored:    t.tally.Unscored,
		Accuracy:    aicorrection.Accuracy(t.tally.Correct, t.tally.Scored),
		Fields:      t.aggregator.Fields(),
	}
}
