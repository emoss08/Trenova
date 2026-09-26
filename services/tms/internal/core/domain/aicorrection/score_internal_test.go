package aicorrection

import (
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func expectedSnapshot(t *testing.T) *Snapshot {
	t.Helper()

	pickup := time.Date(2026, time.March, 16, 3, 0, 0, 0, time.UTC).Unix()
	snapshot := &Snapshot{
		Fields: map[string]string{
			FieldReference:    "LD-10442",
			FieldRate:         "2450.00",
			FieldPickupWindow: "2026-03-15",
		},
		Stops: []StopSnapshot{
			{
				Role: RolePickup, Name: "Northwind Foods", City: "Dallas", State: "TX",
				PostalCode: "75201", ScheduledWindowStart: pickup, Timezone: "America/Chicago",
			},
		},
	}

	encoded, err := sonic.Marshal(snapshot)
	require.NoError(t, err)
	var stored Snapshot
	require.NoError(t, sonic.Unmarshal(encoded, &stored))

	return &stored
}

func predictionFrom(fields map[string]any, stops ...any) *Prediction {
	return ReadPrediction(map[string]any{"fields": fields, "stops": stops})
}

func valueField(value string) map[string]any {
	return map[string]any{"value": value, "source": "ai", "confidence": 0.9}
}

func resultsOf(results []FieldResult) map[string]Outcome {
	out := make(map[string]Outcome, len(results))
	for i := range results {
		out[results[i].Key] = results[i].Outcome
	}
	return out
}

func TestScore_AgainstAStoredSnapshot(t *testing.T) {
	t.Parallel()

	prediction := predictionFrom(
		map[string]any{
			"loadNumber":   valueField("ld 10442"),
			"rate":         valueField("$2,450"),
			"pickupWindow": valueField("03/15/2026 22:00"),
		},
		map[string]any{"role": "shipper", "name": "Northwind", "city": "Dallas", "state": "tx"},
	)

	outcomes := resultsOf(Score(prediction, expectedSnapshot(t)))

	assert.Equal(t, OutcomeCorrect, outcomes[FieldReference])
	assert.Equal(t, OutcomeCorrect, outcomes[FieldRate])
	assert.Equal(t, OutcomeCorrect, outcomes[FieldPickupWindow], "the day is read in the stop's timezone")
	assert.Equal(t, OutcomeCorrect, outcomes["stops.pickup[0].name"])
	assert.Equal(t, OutcomeCorrect, outcomes["stops.pickup[0].state"])
	assert.Equal(t, OutcomeMissed, outcomes["stops.pickup[0].postalCode"])
}

func TestScore_AnEmptyPredictionMissesEverythingConfirmed(t *testing.T) {
	t.Parallel()

	results := Score(ReadPrediction(nil), expectedSnapshot(t))
	require.NotEmpty(t, results)
	for i := range results {
		assert.Equal(t, OutcomeMissed, results[i].Outcome, results[i].Key)
	}
}

func TestReadPrediction_CapsStopsPerRole(t *testing.T) {
	t.Parallel()

	stops := make([]any, 0, MaxStopsPerRole+5)
	for range MaxStopsPerRole + 5 {
		stops = append(stops, map[string]any{"role": "delivery", "name": "Drop"})
	}

	prediction := predictionFrom(map[string]any{}, stops...)
	assert.Len(t, prediction.Snapshot.Stops, MaxStopsPerRole)
	assert.Len(t, prediction.StopMeta, MaxStopsPerRole)
}

func TestAccuracyAggregator_GroupsStopsAndOrdersWorstFirst(t *testing.T) {
	t.Parallel()

	aggregator := NewAccuracyAggregator()
	aggregator.Add([]FieldResult{
		{Key: FieldRate, Outcome: OutcomeCorrect},
		{Key: "stops.pickup[0].city", Outcome: OutcomeCorrected},
		{Key: "stops.pickup[1].city", Outcome: OutcomeCorrect},
	})
	aggregator.Add([]FieldResult{
		{Key: FieldRate, Outcome: OutcomeCorrect},
		{Key: "stops.pickup[0].city", Outcome: OutcomeUnconfirmed},
	})

	fields := aggregator.Fields()
	require.Len(t, fields, 2)
	assert.Equal(t, "stops.pickup.city", fields[0].Key)
	assert.Equal(t, 2, fields[0].Scored)
	assert.Equal(t, 1, fields[0].Unconfirmed)
	assert.InDelta(t, 0.5, fields[0].Accuracy, 1e-9)
	assert.Equal(t, FieldRate, fields[1].Key)
	assert.InDelta(t, 1.0, fields[1].Accuracy, 1e-9)
	assert.Zero(t, Accuracy(0, 0))
}

func TestReadPrediction_ReadsStopsInMemoryAndFromStorage(t *testing.T) {
	t.Parallel()

	stop := map[string]any{"role": "delivery", "name": "Contoso"}
	inMemory := ReadPrediction(map[string]any{"stops": []map[string]any{stop}})
	stored := ReadPrediction(map[string]any{"stops": []any{stop, "not a stop"}})

	require.Len(t, inMemory.Snapshot.Stops, 1)
	require.Len(t, stored.Snapshot.Stops, 1)
	assert.Equal(t, RoleDelivery, inMemory.Snapshot.Stops[0].Role)
	assert.Equal(t, inMemory.Snapshot.Stops, stored.Snapshot.Stops)
}
