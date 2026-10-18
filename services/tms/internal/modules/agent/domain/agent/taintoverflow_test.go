package agent

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunTaint_KeepsEverySourcePastTheMarkCap(t *testing.T) {
	t.Parallel()

	taint := &RunTaint{}
	for idx := range MaxTaintMarks {
		require.True(t, taint.Add(TaintMark{
			Source:   TaintSourceBankReceipt,
			ToolName: "get_bank_receipt",
			CallID:   fmt.Sprintf("call_%d", idx),
		}))
	}
	assert.False(t, taint.Overflowed)

	added := taint.Add(TaintMark{Source: TaintSourceInboundMessage, ToolName: "get_inbound_message", CallID: "call_email"})

	assert.False(t, added, "the marks are full")
	assert.Len(t, taint.Marks, MaxTaintMarks)
	assert.True(t, taint.Overflowed)
	assert.True(t, taint.Has(TaintSourceInboundMessage), "the email's source is not lost")
	assert.ElementsMatch(t, []TaintSource{TaintSourceBankReceipt, TaintSourceInboundMessage}, taint.Sources())

	clone := taint.Clone()
	assert.True(t, clone.Overflowed)
	assert.True(t, clone.Has(TaintSourceInboundMessage))
}

func TestRunTaint_ARepeatedReadIsNotAnOverflow(t *testing.T) {
	t.Parallel()

	taint := &RunTaint{}
	mark := TaintMark{Source: TaintSourceWeb, ToolName: "web_search", CallID: "call_1"}
	taint.Add(mark)
	taint.Add(mark)

	assert.Len(t, taint.Marks, 1)
	assert.False(t, taint.Overflowed)
}

func TestRunTaint_MarksCrossingADelegationAreFlagged(t *testing.T) {
	t.Parallel()

	own := TaintMark{Source: TaintSourceBankReceipt, ToolName: "get_bank_receipt", CallID: "call_own"}
	parent := &RunTaint{}
	parent.Add(own)

	handed := parent.HandedOn()
	require.Len(t, handed.Marks, 1)
	assert.True(t, handed.Marks[0].Delegated)
	assert.False(t, parent.Marks[0].Delegated, "handing on leaves the turn's own marks alone")

	delegate := handed.Clone()
	delegate.Add(TaintMark{Source: TaintSourceWeb, ToolName: "web_search", CallID: "call_web"})

	added := parent.Inherit(delegate)
	require.Len(t, added, 1, "the turn's own mark is not taken again")
	assert.True(t, added[0].Delegated)
	assert.False(t, parent.Marks[0].Delegated)
	assert.Nil(t, (*RunTaint)(nil).HandedOn())
}
