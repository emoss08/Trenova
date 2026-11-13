package agentquality

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func recordedCase(fixtures ...ToolFixture) *EvalCase {
	return &EvalCase{ToolFixtures: fixtures}
}

func TestRecording_AnswersTheSameToolAndArgumentsWithWhatTheOriginalWasGiven(t *testing.T) {
	t.Parallel()

	recording := recordedCase(ToolFixture{
		Tool:   "shipment_get",
		Args:   map[string]any{"id": "shp_1", "fields": []any{"status", "proNumber"}},
		Result: map[string]any{"proNumber": "S-100", "status": "InTransit"},
	}).Recording()
	require.NotNil(t, recording)

	result, failed, ok := recording.Recorded("shipment_get", map[string]any{
		"fields": []any{"status", "proNumber"},
		"id":     "shp_1",
	})
	require.True(t, ok, "the order the arguments were written in does not matter")
	assert.False(t, failed)
	assert.Equal(t, map[string]any{"proNumber": "S-100", "status": "InTransit"}, result)
	assert.Equal(t, 1, recording.Served())
}

func TestRecording_HoldsNothingForAnotherToolOrOtherArguments(t *testing.T) {
	t.Parallel()

	recording := recordedCase(ToolFixture{
		Tool:   "shipment_get",
		Args:   map[string]any{"id": "shp_1"},
		Result: map[string]any{"proNumber": "S-100"},
	}).Recording()

	_, _, ok := recording.Recorded("shipment_get", map[string]any{"id": "shp_2"})
	assert.False(t, ok, "a read the original never made is made against the records")
	_, _, ok = recording.Recorded("customer_get", map[string]any{"id": "shp_1"})
	assert.False(t, ok)
	_, _, ok = recording.Recorded("shipment_get", nil)
	assert.False(t, ok)
	assert.Zero(t, recording.Served())
}

func TestRecording_ReadsWithNoArgumentsMatchEachOther(t *testing.T) {
	t.Parallel()

	recording := recordedCase(ToolFixture{
		Tool:   "billing_queue_summary",
		Result: map[string]any{"waiting": float64(4)},
	}).Recording()

	result, _, ok := recording.Recorded("billing_queue_summary", map[string]any{})
	require.True(t, ok)
	assert.Equal(t, map[string]any{"waiting": float64(4)}, result)
}

func TestRecording_RepeatedReadsComeBackInTheOrderTheyWereGiven(t *testing.T) {
	t.Parallel()

	args := map[string]any{"id": "shp_1"}
	recording := recordedCase(
		ToolFixture{Tool: "shipment_get", Args: args, Result: map[string]any{"status": "New"}},
		ToolFixture{Tool: "shipment_get", Args: args, Result: map[string]any{"status": "Assigned"}},
	).Recording()

	first, _, _ := recording.Recorded("shipment_get", args)
	second, _, _ := recording.Recorded("shipment_get", args)
	third, _, ok := recording.Recorded("shipment_get", args)

	assert.Equal(t, map[string]any{"status": "New"}, first)
	assert.Equal(t, map[string]any{"status": "Assigned"}, second)
	require.True(t, ok)
	assert.Equal(t, map[string]any{"status": "Assigned"}, third,
		"a read asked more often than the original keeps getting the last answer")
}

func TestRecording_AFailedReadFailsAgain(t *testing.T) {
	t.Parallel()

	recording := recordedCase(ToolFixture{
		Tool:   "shipment_get",
		Args:   map[string]any{"id": "shp_9"},
		Result: "Shipment not found",
		Failed: true,
	}).Recording()

	result, failed, ok := recording.Recorded("shipment_get", map[string]any{"id": "shp_9"})
	require.True(t, ok)
	assert.True(t, failed)
	assert.Equal(t, "Shipment not found", result)
}

func TestRecording_LeavesOutWhatWasRedactedSoItIsReadFromTheRecords(t *testing.T) {
	t.Parallel()

	recording := recordedCase(
		ToolFixture{
			Tool: "customer_get",
			Args: map[string]any{"id": "cus_1"},
			Result: map[string]any{
				"name":    "Acme Foods",
				"billing": map[string]any{"creditLimit": RestrictedPlaceholder("creditLimit")},
			},
		},
		ToolFixture{
			Tool:   "worker_get",
			Args:   map[string]any{"ssn": RestrictedPlaceholder("ssn")},
			Result: map[string]any{"name": "Dana"},
		},
		ToolFixture{
			Tool:   "shipment_get",
			Args:   map[string]any{"id": "shp_1"},
			Result: map[string]any{"proNumber": "S-100"},
		},
	).Recording()
	require.NotNil(t, recording)

	_, _, ok := recording.Recorded("customer_get", map[string]any{"id": "cus_1"})
	assert.False(t, ok, "a result with a field taken out is not what the original was given")
	_, _, ok = recording.Recorded("worker_get", map[string]any{"ssn": RestrictedPlaceholder("ssn")})
	assert.False(t, ok)
	_, _, ok = recording.Recorded("shipment_get", map[string]any{"id": "shp_1"})
	assert.True(t, ok)
}

func TestRecording_IsNothingForACaseWithNoUsableReads(t *testing.T) {
	t.Parallel()

	assert.Nil(t, recordedCase().Recording())
	assert.Nil(t, recordedCase(ToolFixture{Tool: "shipment_get"}).Recording(),
		"a call whose result was not kept has nothing to answer with")

	var none *Recording
	_, _, ok := none.Recorded("shipment_get", nil)
	assert.False(t, ok)
	assert.Zero(t, none.Served())
}

func TestRecording_AnAnswerCannotBeChangedByWhoeverReadIt(t *testing.T) {
	t.Parallel()

	args := map[string]any{"id": "shp_1"}
	recording := recordedCase(ToolFixture{
		Tool:   "shipment_get",
		Args:   args,
		Result: map[string]any{"stops": []any{map[string]any{"city": "Dallas"}}},
	}).Recording()

	first, _, _ := recording.Recorded("shipment_get", args)
	first.(map[string]any)["stops"].([]any)[0].(map[string]any)["city"] = "Changed"

	second, _, _ := recording.Recorded("shipment_get", args)
	assert.Equal(t, "Dallas", second.(map[string]any)["stops"].([]any)[0].(map[string]any)["city"],
		"the runtime rewrites dates inside a result in place, which must not reach the next replay")
}
