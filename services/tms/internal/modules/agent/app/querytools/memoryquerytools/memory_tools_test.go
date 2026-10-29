package memoryquerytools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRecall struct {
	agentmemoryports.AgentMemoryService
	captured agentmemoryports.RecallAgentMemoriesRequest
	items    []*agentmemory.Memory
	used     []pulid.ID
}

func (f *fakeRecall) Recall(
	_ context.Context,
	req *agentmemoryports.RecallAgentMemoriesRequest,
) ([]agentmemoryports.RecalledMemory, error) {
	f.captured = *req

	var match agentmemory.MemoryMatch
	if req.Query != "" {
		match = agentmemory.MemoryMatchWords
	}
	recalled := make([]agentmemoryports.RecalledMemory, 0, len(f.items))
	for _, item := range f.items {
		recalled = append(recalled, agentmemoryports.RecalledMemory{Memory: item, Match: match})
	}

	return recalled, nil
}

func (f *fakeRecall) RecordUse(
	_ context.Context,
	req agentmemoryports.RecordMemoryUseRequest,
) error {
	f.used = append(f.used, req.IDs...)

	return nil
}

func TestRecallMemory_NarrowsToASubjectAndReadsBackTheFilters(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	recall := &fakeRecall{items: []*agentmemory.Memory{{
		ID:           pulid.MustNew("amem_"),
		Kind:         agentmemory.MemoryKindInstruction,
		Source:       agentmemory.MemorySourceUser,
		SubjectType:  agentmemory.MemorySubjectCustomer,
		SubjectLabel: "Acme Freight",
		Content:      "Needs the POD within one day.",
		CreatedAt:    1_760_000_000,
	}}}
	tool := NewRecallMemoryTool(recall)

	result, err := tool.Query(t.Context(), querytooltest.TestParams(map[string]any{
		"query":       "POD",
		"subjectType": "Customer",
		"subjectId":   customerID.String(),
	}))
	require.NoError(t, err)

	recalled, ok := result.(recallOutcome)
	require.True(t, ok)
	outcome := recalled.SearchOutcome
	assert.Equal(t, 1, outcome.Count)
	assert.Contains(t, outcome.SearchedFor, `text matching "POD"`)
	assert.Contains(t, outcome.SearchedFor, "about customer "+customerID.String())
	assert.Equal(t, customerID, recall.captured.SubjectID)
	assert.Equal(t, agentmemory.MemorySubjectCustomer, recall.captured.SubjectType)

	rows, ok := outcome.Items.([]memoryRow)
	require.True(t, ok)
	assert.Equal(t, "Acme Freight (customer)", rows[0].About)
	assert.Equal(t, "a person", rows[0].RecordedBy)
	assert.Equal(t, "words", rows[0].Match, "a row found by its words says so")
}

func TestRecallMemory_SaysNothingMatchedRatherThanNothingExists(t *testing.T) {
	t.Parallel()

	tool := NewRecallMemoryTool(&fakeRecall{})

	result, err := tool.Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{"query": "detention"}),
	)
	require.NoError(t, err)

	outcome := result.(recallOutcome).SearchOutcome
	assert.Equal(t, 0, outcome.Count)
	assert.Contains(t, outcome.Note, "No memories matched")
}

// A recall names the memories a tainted run wrote, so the runtime can taint
// the run that reads them back. The model reads the same rows either way.
func TestRecallMemory_NamesTheTaintedMemoriesItReturned(t *testing.T) {
	t.Parallel()

	dirty := &agentmemory.Memory{
		ID:      pulid.MustNew("amem_"),
		Kind:    agentmemory.MemoryKindFact,
		Source:  agentmemory.MemorySourceAgent,
		Content: "Ship to dock 9.",
		Tainted: true,
	}
	clean := &agentmemory.Memory{
		ID:      pulid.MustNew("amem_"),
		Kind:    agentmemory.MemoryKindInstruction,
		Source:  agentmemory.MemorySourceUser,
		Content: "Quote in dollars.",
	}
	agentID := pulid.MustNew("agdef_")
	recall := &fakeRecall{items: []*agentmemory.Memory{dirty, clean}}
	params := querytooltest.TestParams(map[string]any{})
	params.AgentDefinitionID = agentID

	result, err := NewRecallMemoryTool(recall).Query(t.Context(), params)
	require.NoError(t, err)

	carrier, ok := result.(agent.TaintCarrier)
	require.True(t, ok)
	assert.Equal(t, []agentrecord.RecordRef{{
		EntityType: agent.TaintEntityAgentMemory,
		ID:         dirty.ID.String(),
	}}, carrier.TaintedRecords())
	assert.Equal(t, agentID, recall.captured.AgentDefinitionID,
		"recall reads what was kept for the agent asking, and the organization's")
	assert.Equal(
		t,
		agenttooltrust.ExternalReadMarked,
		NewRecallMemoryTool(recall).Policy().ReadsExternal,
	)
}

func TestRecallMemory_RefusesHalfASubject(t *testing.T) {
	t.Parallel()

	tool := NewRecallMemoryTool(&fakeRecall{})

	_, err := tool.Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{"subjectId": "cus_1"}),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "subjectId needs subjectType")
}

// A model that fills every field sends a subject type with no id for a search
// about no record. Refusing it looped the model on agent_list_memories until the
// turn gave up, so a type with no id narrows nothing.
func TestRecallMemory_ReadsATypeWithNoIDAsNoSubject(t *testing.T) {
	t.Parallel()

	tool := NewRecallMemoryTool(&fakeRecall{})

	_, err := tool.Query(t.Context(), querytooltest.TestParams(map[string]any{
		"query":       "billing queue item rather than invoice",
		"subjectType": "Customer",
		"subjectId":   "",
	}))
	require.NoError(t, err)
}

func TestRecordedBy_NamesEverySource(t *testing.T) {
	t.Parallel()

	for _, source := range agentmemory.AllMemorySources() {
		said := recordedBy(source)
		assert.NotEqual(t, string(source), said, "%s is named in words, not by its code", source)
		assert.NotEmpty(t, said)
	}
	assert.Equal(
		t,
		"people's ratings of an agent's work",
		recordedBy(agentmemory.MemorySourceFeedback),
	)
}

func TestRecallMemory_DefaultsToTenAndCapsAtFifty(t *testing.T) {
	t.Parallel()

	recall := &fakeRecall{}
	tool := NewRecallMemoryTool(recall)

	_, err := tool.Query(t.Context(), querytooltest.TestParams(map[string]any{"query": "dock"}))
	require.NoError(t, err)
	assert.Equal(t, agentmemory.DefaultMemoryRecallLimit, recall.captured.Limit)

	_, err = tool.Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{"query": "dock", "limit": 500}),
	)
	require.NoError(t, err)
	assert.Equal(t, agentmemory.MaxMemoryRecallLimit, recall.captured.Limit)

	_, err = tool.Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{"query": "dock", "limit": 0}),
	)
	require.NoError(t, err)
	assert.Equal(t, agentmemory.DefaultMemoryRecallLimit, recall.captured.Limit)
}

func TestRecallMemory_ReadsOneMemoryByIDWithoutAMatchLabel(t *testing.T) {
	t.Parallel()

	memoryID := pulid.MustNew("amem_")
	recall := &fakeRecall{items: []*agentmemory.Memory{{
		ID:      memoryID,
		Kind:    agentmemory.MemoryKindInstruction,
		Source:  agentmemory.MemorySourceUser,
		Content: "The whole of a long memory.",
	}}}

	result, err := NewRecallMemoryTool(recall).Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{"id": memoryID.String()}),
	)
	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{memoryID}, recall.captured.IDs)

	outcome := result.(recallOutcome).SearchOutcome
	rows, ok := outcome.Items.([]memoryRow)
	require.True(t, ok)
	require.Len(t, rows, 1)
	assert.Empty(t, rows[0].Match, "nothing was searched for, so nothing is labelled a match")
	assert.Contains(t, outcome.SearchedFor, "id "+memoryID.String())

	_, err = NewRecallMemoryTool(recall).Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{"id": "not an id"}),
	)
	require.Error(t, err)
}

// A recall answers for the person in the conversation: what is kept for them
// and their roles is read with the organization's, and every memory it reads
// back is counted as used and named to the runtime, which says so under the
// reply.
func TestRecallMemory_ReadsForThePersonAndNamesWhatItUsed(t *testing.T) {
	t.Parallel()

	first, second := pulid.MustNew("amem_"), pulid.MustNew("amem_")
	recall := &fakeRecall{items: []*agentmemory.Memory{
		{
			ID:      first,
			Kind:    agentmemory.MemoryKindFact,
			Source:  agentmemory.MemorySourceUser,
			Content: "One.",
		},
		{
			ID:      second,
			Kind:    agentmemory.MemoryKindFact,
			Source:  agentmemory.MemorySourceAgent,
			Content: "Two.",
		},
	}}
	tool := NewRecallMemoryTool(recall)
	params := querytooltest.TestParams(map[string]any{"query": "acme"})

	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)

	assert.Equal(t, params.Actor.PersonUserID(), recall.captured.ReaderUserID)
	used, ok := result.(agentmemoryports.MemoryRecall)
	require.True(t, ok, "the runtime reads which memories a recall used")
	assert.Equal(t, []pulid.ID{first, second}, used.RecalledMemoryIDs())
	assert.Equal(t, []pulid.ID{first, second}, recall.used)
}
