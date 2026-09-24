package agent

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubjectTaint_OnlyContentWrittenOutsideTheOrganization(t *testing.T) {
	t.Parallel()

	outside := map[SubjectType]TaintSource{
		SubjectInboundMessage: TaintSourceInboundMessage,
		SubjectDocument:       TaintSourceDocument,
		SubjectEDIInboundFile: TaintSourceEDI,
		SubjectBankReceipt:    TaintSourceBankReceipt,
	}
	for _, subject := range AllSubjectTypes() {
		mark, ok := SubjectTaint(subject, "rec_1", 100)
		source, taints := outside[subject]
		assert.Equal(t, taints, ok, subject)
		if !taints {
			continue
		}
		assert.Equal(t, source, mark.Source, subject)
		require.NotNil(t, mark.Ref, subject)
		assert.Equal(t, "rec_1", mark.Ref.ID, subject)
		assert.Equal(t, int64(100), mark.At, subject)
	}

	_, ok := SubjectTaint(SubjectDocument, "", 100)
	assert.False(t, ok, "a subject with no id names nothing")
}

func TestRunTaint_AbsorbReportsOnlyNewMarks(t *testing.T) {
	t.Parallel()

	first := TaintMark{Source: TaintSourceDocument, ToolName: "get_document", CallID: "call_1"}
	second := TaintMark{Source: TaintSourceEDI, ToolName: "get_edi_file", CallID: "call_2"}
	taint := &RunTaint{}

	assert.Equal(t, []TaintMark{first}, taint.Absorb([]TaintMark{first}))
	assert.Equal(t, []TaintMark{second}, taint.Absorb([]TaintMark{first, second}))
	assert.Empty(t, taint.Absorb([]TaintMark{second}))
	assert.Equal(t, []TaintSource{TaintSourceDocument, TaintSourceEDI}, taint.Sources())

	var unknown *RunTaint
	assert.Nil(t, unknown.Absorb([]TaintMark{first}), "an unknown taint stays unknown")
	assert.True(t, unknown.Unknown())
	assert.Nil(t, unknown.Clone())
}

func TestRunTaint_CloneIsIndependent(t *testing.T) {
	t.Parallel()

	taint := &RunTaint{}
	taint.Add(TaintMark{Source: TaintSourceDocument, CallID: "call_1"})
	clone := taint.Clone()
	clone.Add(TaintMark{Source: TaintSourceEDI, CallID: "call_2"})

	assert.Len(t, taint.Marks, 1)
	assert.Len(t, clone.Marks, 2)
}

func TestAgentRun_RecordTaintMergesAndKeepsTheFirstTime(t *testing.T) {
	t.Parallel()

	run := &AgentRun{ID: pulid.MustNew("ar_")}
	assert.False(t, run.RecordTaint(&RunTaint{}, 10), "a clean run is not marked")
	assert.False(t, run.RecordTaint(nil, 10))
	assert.False(t, run.Tainted)
	assert.Nil(t, run.TaintedAt)

	first := &RunTaint{}
	first.Add(TaintMark{Source: TaintSourceDocument, CallID: "call_1"})
	assert.True(t, run.RecordTaint(first, 20))
	assert.True(t, run.Tainted)
	require.NotNil(t, run.TaintedAt)
	assert.Equal(t, int64(20), *run.TaintedAt)

	second := &RunTaint{}
	second.Add(TaintMark{Source: TaintSourceEDI, CallID: "call_2"})
	assert.True(t, run.RecordTaint(second, 30))
	assert.Equal(t, int64(20), *run.TaintedAt, "tainted_at is when it first happened")
	assert.Len(t, run.Taint.Marks, 2)
	assert.False(t, run.RecordTaint(second, 40), "nothing new")

	assert.Equal(t, []RecordRef{{EntityType: TaintEntityAgentRun, ID: run.ID.String()}},
		run.TaintedRecords())
	assert.Empty(t, (&AgentRun{}).TaintedRecords())
}

func TestAgentProposal_RequiresAPersonOnlyWhenTaintedAndLeaving(t *testing.T) {
	t.Parallel()

	for _, class := range EgressClasses() {
		tainted := &AgentProposal{Tainted: true}
		clean := &AgentProposal{}
		assert.Equal(t, class.Leaves(), tainted.RequiresPerson(class), class)
		assert.False(t, clean.RequiresPerson(class), class)
	}
}
