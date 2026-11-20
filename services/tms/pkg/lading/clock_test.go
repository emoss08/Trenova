package lading_test

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShiftFlowStateMovesEveryDeadlineTheFlowCompares(t *testing.T) {
	t.Parallel()

	state := jsonutils.RawJSON(`{
		"key": "PO-1",
		"documents": {"tender": {"weight": 12345678901234567}},
		"load": {"exists": true, "state": "open"},
		"awaitingResponse": 1000,
		"responseDue": 8200,
		"answerDue": 6400,
		"escalated": false,
		"pending": [{"id": "s1", "set": "990", "control": "0001", "due": 5000, "resends": 2}],
		"batches": [{"clause": 3, "documents": [], "due": 9000}],
		"finishedSends": [],
		"sends": 1
	}`)

	shifted, err := lading.ShiftFlowState(state, 7200)
	require.NoError(t, err)

	var got struct {
		Documents        map[string]jsonutils.RawJSON `json:"documents"`
		AwaitingResponse int64                        `json:"awaitingResponse"`
		ResponseDue      int64                        `json:"responseDue"`
		AnswerDue        int64                        `json:"answerDue"`
		Pending          []lading.Pending             `json:"pending"`
		Batches          []struct {
			Clause int   `json:"clause"`
			Due    int64 `json:"due"`
		} `json:"batches"`
		Sends int `json:"sends"`
	}
	require.NoError(t, sonic.Unmarshal(shifted, &got))

	assert.Equal(t, int64(1000-7200), got.AwaitingResponse)
	assert.Equal(t, int64(8200-7200), got.ResponseDue)
	assert.Equal(t, int64(6400-7200), got.AnswerDue)
	require.Len(t, got.Pending, 1)
	assert.Equal(t, int64(5000-7200), got.Pending[0].Due)
	assert.Equal(t, 2, got.Pending[0].Resends)
	assert.Equal(t, "0001", got.Pending[0].Control)
	require.Len(t, got.Batches, 1)
	assert.Equal(t, int64(9000-7200), got.Batches[0].Due)
	assert.Equal(t, 3, got.Batches[0].Clause)
	assert.Equal(t, 1, got.Sends)
	assert.JSONEq(t, `{"weight": 12345678901234567}`, string(got.Documents["tender"]))
}

func TestShiftFlowStateLeavesAbsentDeadlinesAbsent(t *testing.T) {
	t.Parallel()

	shifted, err := lading.ShiftFlowState(
		jsonutils.RawJSON(`{"key": "PO-2", "awaitingResponse": null, "pending": []}`),
		3600,
	)
	require.NoError(t, err)
	assert.JSONEq(t, `{"key": "PO-2", "awaitingResponse": null, "pending": []}`, string(shifted))

	empty, err := lading.ShiftFlowState(nil, 3600)
	require.NoError(t, err)
	assert.Empty(t, empty)
}
