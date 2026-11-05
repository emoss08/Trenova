package turntaint_test

import (
	"strconv"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/turntaint"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type markedRows []agentrecord.RecordRef

func (r markedRows) TaintedRecords() []agentrecord.RecordRef { return r }

func rowsOf(entity string, count int) markedRows {
	rows := make(markedRows, 0, count)
	for idx := range count {
		rows = append(rows, agentrecord.RecordRef{EntityType: entity, ID: strconv.Itoa(idx)})
	}

	return rows
}

func listMarks(source agent.TaintSource, callID string, rows markedRows) []agent.TaintMark {
	policy := agenttoolports.ToolPolicy{
		ReadsExternal: agenttooltrust.ExternalReadMarked,
		Source:        source,
	}

	return turntaint.Call(&policy, &llmports.ToolCall{ID: callID, Name: "shipment_list"}, rows, 10)
}

func TestCall_OneListCannotFillTheTurnsMarks(t *testing.T) {
	t.Parallel()

	marks := listMarks(
		agent.TaintSourceEDI,
		"call-1",
		rowsOf(agent.TaintEntityShipment, agent.MaxTaintMarks+4),
	)

	require.Len(t, marks, 1)
	assert.Equal(t, agent.TaintSourceEDI, marks[0].Source)
	assert.Nil(t, marks[0].Ref, "the mark stands for the whole list")

	taint := &agent.RunTaint{}
	taint.Absorb(marks)
	assert.False(t, taint.Overflowed)
	assert.True(t, taint.Has(agent.TaintSourceEDI))
}

func TestCall_TwoListsKeepTheirOwnMarks(t *testing.T) {
	t.Parallel()

	rows := rowsOf(agent.TaintEntityShipment, agent.MaxMarksPerCall+1)
	taint := &agent.RunTaint{}
	taint.Absorb(listMarks(agent.TaintSourceEDI, "call-1", rows))
	taint.Absorb(listMarks(agent.TaintSourceEDI, "call-2", rows))

	assert.Len(t, taint.Marks, 2)
}

func TestCall_AShortListNamesEachRecord(t *testing.T) {
	t.Parallel()

	marks := listMarks(
		agent.TaintSourceEDI,
		"call-1",
		rowsOf(agent.TaintEntityShipment, agent.MaxMarksPerCall),
	)

	require.Len(t, marks, agent.MaxMarksPerCall)
	for idx := range marks {
		require.NotNil(t, marks[idx].Ref)
		assert.Equal(t, strconv.Itoa(idx), marks[idx].Ref.ID)
	}
}

func TestCall_ARereadableListNamesTheRecordsABoundVouchesFor(t *testing.T) {
	t.Parallel()

	marks := listMarks(
		agent.TaintSourceBankReceipt,
		"call-1",
		rowsOf(agent.TaintEntityBankReceipt, agent.MaxTaintMarks-1),
	)
	require.Len(t, marks, agent.MaxTaintMarks-1)
	assert.NotNil(t, marks[0].Ref)

	marks = listMarks(
		agent.TaintSourceBankReceipt,
		"call-1",
		rowsOf(agent.TaintEntityBankReceipt, agent.MaxTaintMarks),
	)
	require.Len(t, marks, 1)
	assert.Nil(t, marks[0].Ref, "a list that would fill the marks cannot be vouched for anyway")
}
