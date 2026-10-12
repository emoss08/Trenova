package agent

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEgressClass_LeavesAndCeiling(t *testing.T) {
	t.Parallel()

	cases := map[EgressClass]struct {
		leaves  bool
		ceiling AutonomyTier
	}{
		EgressNone:              {leaves: false, ceiling: TierAutoExecute},
		EgressPersonal:          {leaves: false, ceiling: TierAutoExecute},
		EgressInternal:          {leaves: false, ceiling: TierAutoExecute},
		EgressCustomerVisible:   {leaves: true, ceiling: TierActWithApproval},
		EgressDriverVisible:     {leaves: true, ceiling: TierActWithApproval},
		EgressExternalRecipient: {leaves: true, ceiling: TierActWithApproval},
		EgressMoney:             {leaves: true, ceiling: TierAutoExecute},
	}

	assert.Len(t, EgressClasses(), len(cases))
	for class, want := range cases {
		assert.True(t, class.IsValid(), class)
		assert.Equal(t, want.leaves, class.Leaves(), class)
		assert.Equal(t, want.ceiling, class.Ceiling(), class)
	}
	assert.False(t, EgressClass("broadcast").IsValid())
}

func TestRunTaint_NilIsUnknownAndEmptyIsClean(t *testing.T) {
	t.Parallel()

	var unknown *RunTaint
	assert.False(t, unknown.Tainted())
	assert.False(t, unknown.Add(TaintMark{Source: TaintSourceDocument}))
	assert.False(t, (&RunTaint{}).Tainted())
}

func TestRunTaint_AddDedupesAndBounds(t *testing.T) {
	t.Parallel()

	taint := &RunTaint{}
	mark := TaintMark{
		Source:   TaintSourceInboundMessage,
		ToolName: "get_inbound_message",
		CallID:   "call_1",
		Ref:      &RecordRef{EntityType: "inbound_message", ID: "imsg_1"},
	}
	assert.True(t, taint.Add(mark))
	assert.True(t, taint.Tainted())

	again := mark
	again.CallID = "call_2"
	assert.False(t, taint.Add(again), "the same record read twice is one mark")
	assert.False(t, taint.Add(TaintMark{Source: "gossip"}), "an unknown source is refused")

	for idx := range MaxTaintMarks + 4 {
		taint.Add(TaintMark{
			Source:   TaintSourceDocument,
			ToolName: "get_document_summary",
			CallID:   fmt.Sprintf("call_%d", idx+10),
		})
	}
	assert.Len(t, taint.Marks, MaxTaintMarks)
	assert.True(t, taint.Tainted())
}

func TestRunTaint_Merge(t *testing.T) {
	t.Parallel()

	first := &RunTaint{}
	first.Add(TaintMark{Source: TaintSourceWeather, ToolName: "list_weather_alerts", CallID: "a"})
	second := &RunTaint{}
	second.Add(TaintMark{Source: TaintSourceWeather, ToolName: "list_weather_alerts", CallID: "a"})
	second.Add(TaintMark{Source: TaintSourceMemory, ToolName: "recall_memory", CallID: "b"})

	first.Merge(second)
	first.Merge(nil)
	assert.Len(t, first.Marks, 2)

	var unknown *RunTaint
	unknown.Merge(second)
	assert.Nil(t, unknown)
}
