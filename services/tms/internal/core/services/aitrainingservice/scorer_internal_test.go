package aitrainingservice

import (
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/aitraining"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/aidocumentservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/extractionevaljobs"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func expectedSnapshot() *aitraining.ExampleSnapshot {
	return &aitraining.ExampleSnapshot{
		Fields: map[string]string{
			aicorrection.FieldReference:    "OUG-8393964",
			aicorrection.FieldRate:         "2563.12",
			aicorrection.FieldShipper:      "Juniper Manufacturing",
			aicorrection.FieldPickupWindow: "2026-03-14",
		},
		Stops: []aitraining.ExampleStop{{
			Role: aicorrection.RolePickup, Name: "Juniper Manufacturing", City: "Hartwell",
			State: "CA", PostalCode: "85055", Date: "2026-03-14",
		}},
	}
}

func jsonLines(t *testing.T, records ...any) string {
	t.Helper()
	var b strings.Builder
	for _, record := range records {
		encoded, err := sonic.Marshal(record)
		require.NoError(t, err)
		b.Write(encoded)
		b.WriteByte('\n')
	}
	return b.String()
}

func replyFor(t *testing.T, snapshot *aitraining.ExampleSnapshot, dateText string) string {
	t.Helper()
	fields := make([]map[string]any, 0, len(snapshot.Fields))
	for key, value := range snapshot.Fields {
		if key == aicorrection.FieldPickupWindow && dateText != "" {
			value = dateText
		}
		fields = append(fields, map[string]any{
			"key": key, "label": key, "value": value, "confidence": 0.9, "evidenceExcerpt": "",
			"pageNumber": 1, "reviewRequired": false, "conflict": false, "source": "ai",
			"alternativeValues": []string{},
		})
	}
	stops := make([]map[string]any, 0, len(snapshot.Stops))
	for i, stop := range snapshot.Stops {
		date := stop.Date
		if dateText != "" {
			date = dateText
		}
		stops = append(stops, map[string]any{
			"sequence": i + 1, "role": stop.Role, "name": stop.Name, "addressLine1": stop.AddressLine1,
			"addressLine2": stop.AddressLine2, "city": stop.City, "state": stop.State,
			"postalCode": stop.PostalCode, "date": date, "timeWindow": stop.TimeWindow,
			"appointmentRequired": stop.AppointmentRequired, "pageNumber": 1, "evidenceExcerpt": "",
			"confidence": 0.9, "reviewRequired": false, "source": "ai",
		})
	}
	encoded, err := sonic.Marshal(map[string]any{
		"documentKind": "RateConfirmation", "overallConfidence": 0.9, "reviewStatus": "Ready",
		"missingFields": []string{}, "signals": []string{}, "conflicts": []any{},
		"fields": fields, "stops": stops,
	})
	require.NoError(t, err)
	return string(encoded)
}

func newTestScorer() *Scorer {
	return NewScorer(extractionevaljobs.NewReplyReader(aidocumentservice.Contract{}))
}

func TestScorerComparesTheModelWithTheBaseline(t *testing.T) {
	t.Parallel()

	expected := expectedSnapshot()
	baseline := expectedSnapshot()
	baseline.Fields[aicorrection.FieldShipper] = "Granite Brands"
	baseline.Fields[aicorrection.FieldRate] = "2600.00"

	evaluation := jsonLines(t,
		&aitraining.EvaluationRecord{ID: "a", Expected: expected, Baseline: baseline},
		&aitraining.EvaluationRecord{ID: "b", Expected: expected, Baseline: baseline},
		&aitraining.EvaluationRecord{ID: "c", Expected: expected, Baseline: baseline},
		&aitraining.EvaluationRecord{ID: "d", Expected: expected, Baseline: baseline},
	)
	predictions := jsonLines(t,
		&aitraining.PredictionRecord{ID: "a", Reply: replyFor(t, expected, "03/14/2026")},
		&aitraining.PredictionRecord{ID: "b", Reply: "not json"},
		&aitraining.PredictionRecord{ID: "c", Error: "context length exceeded"},
		&aitraining.PredictionRecord{ID: "zzz", Reply: replyFor(t, expected, "")},
	)

	report, err := newTestScorer().Score(t.Context(), &services.ScoreTrainingPredictionsRequest{
		Evaluation:  strings.NewReader(evaluation),
		Predictions: strings.NewReader(predictions),
	})
	require.NoError(t, err)

	assert.Equal(t, 4, report.Examples)
	assert.Equal(t, 1, report.Predicted)
	assert.Equal(t, 1, report.UnreadablePredictions)
	assert.Equal(t, 1, report.FailedPredictions)
	assert.Equal(t, 1, report.MissingPredictions)
	assert.Equal(t, 1, report.UnknownPredictions)
	assert.Equal(t, report.Model.Scored, report.Baseline.Scored)
	assert.Positive(t, report.Model.Correct)
	assert.Equal(t, report.Model.Scored-report.Model.Correct, report.Model.Missed)
	assert.Zero(t, report.Baseline.Missed)
	assert.Positive(t, report.Baseline.Corrected)
	assert.InDelta(t, report.Model.Accuracy-report.Baseline.Accuracy, report.AccuracyDelta, 1e-9)
	assert.NotEmpty(t, report.Model.Fields)
}

func TestScorerGivesAPerfectReplyFullMarks(t *testing.T) {
	t.Parallel()

	expected := expectedSnapshot()
	report, err := newTestScorer().Score(t.Context(), &services.ScoreTrainingPredictionsRequest{
		Evaluation: strings.NewReader(jsonLines(t,
			&aitraining.EvaluationRecord{ID: "a", Expected: expected, Baseline: expected},
		)),
		Predictions: strings.NewReader(jsonLines(t,
			&aitraining.PredictionRecord{ID: "a", Reply: replyFor(t, expected, "")},
		)),
	})
	require.NoError(t, err)

	assert.Equal(t, report.Model.Scored, report.Model.Correct)
	assert.InDelta(t, 1.0, report.Model.Accuracy, 1e-9)
	assert.InDelta(t, 0.0, report.AccuracyDelta, 1e-9)
}

func TestScorerRejectsDuplicateAndAnonymousPredictions(t *testing.T) {
	t.Parallel()

	for _, predictions := range []string{
		jsonLines(t, &aitraining.PredictionRecord{ID: "a"}, &aitraining.PredictionRecord{ID: "a"}),
		jsonLines(t, &aitraining.PredictionRecord{Reply: "{}"}),
	} {
		_, err := newTestScorer().Score(t.Context(), &services.ScoreTrainingPredictionsRequest{
			Evaluation:  strings.NewReader(""),
			Predictions: strings.NewReader(predictions),
		})
		require.Error(t, err)
		assert.True(t, errortypes.IsBusinessError(err))
	}

	_, err := newTestScorer().Score(t.Context(), &services.ScoreTrainingPredictionsRequest{})
	require.Error(t, err)
}
