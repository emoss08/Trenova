package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRecall struct {
	serviceports.AgentMemoryService
	captured serviceports.RecallAgentMemoriesRequest
	items    []*agent.Memory
}

func (f *fakeRecall) Recall(
	_ context.Context,
	req serviceports.RecallAgentMemoriesRequest,
) ([]*agent.Memory, error) {
	f.captured = req

	return f.items, nil
}

func TestRecallMemory_NarrowsToASubjectAndReadsBackTheFilters(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	recall := &fakeRecall{items: []*agent.Memory{{
		ID:           pulid.MustNew("amem_"),
		Kind:         agent.MemoryKindInstruction,
		Source:       agent.MemorySourceUser,
		SubjectType:  agent.MemorySubjectCustomer,
		SubjectLabel: "Acme Freight",
		Content:      "Needs the POD within one day.",
		CreatedAt:    1_760_000_000,
	}}}
	tool := newRecallMemoryTool(recall)

	result, err := tool.Query(t.Context(), testParams(map[string]any{
		"query":       "POD",
		"subjectType": "Customer",
		"subjectId":   customerID.String(),
	}))
	require.NoError(t, err)

	recalled, ok := result.(recallOutcome)
	require.True(t, ok)
	outcome := recalled.searchOutcome
	assert.Equal(t, 1, outcome.Count)
	assert.Contains(t, outcome.SearchedFor, `text matching "POD"`)
	assert.Contains(t, outcome.SearchedFor, "about customer "+customerID.String())
	assert.Equal(t, customerID, recall.captured.SubjectID)
	assert.Equal(t, agent.MemorySubjectCustomer, recall.captured.SubjectType)

	rows, ok := outcome.Items.([]memoryRow)
	require.True(t, ok)
	assert.Equal(t, "Acme Freight (customer)", rows[0].About)
	assert.Equal(t, "a person", rows[0].RecordedBy)
}

func TestRecallMemory_SaysNothingMatchedRatherThanNothingExists(t *testing.T) {
	t.Parallel()

	tool := newRecallMemoryTool(&fakeRecall{})

	result, err := tool.Query(t.Context(), testParams(map[string]any{"query": "detention"}))
	require.NoError(t, err)

	outcome := result.(recallOutcome).searchOutcome
	assert.Equal(t, 0, outcome.Count)
	assert.Contains(t, outcome.Note, "No memories matched")
}

// A recall names the memories a tainted run wrote, so the runtime can taint
// the run that reads them back. The model reads the same rows either way.
func TestRecallMemory_NamesTheTaintedMemoriesItReturned(t *testing.T) {
	t.Parallel()

	dirty := &agent.Memory{
		ID:      pulid.MustNew("amem_"),
		Kind:    agent.MemoryKindFact,
		Source:  agent.MemorySourceAgent,
		Content: "Ship to dock 9.",
		Tainted: true,
	}
	clean := &agent.Memory{
		ID:      pulid.MustNew("amem_"),
		Kind:    agent.MemoryKindInstruction,
		Source:  agent.MemorySourceUser,
		Content: "Quote in dollars.",
	}
	agentID := pulid.MustNew("agdef_")
	recall := &fakeRecall{items: []*agent.Memory{dirty, clean}}
	params := testParams(map[string]any{})
	params.AgentDefinitionID = agentID

	result, err := newRecallMemoryTool(recall).Query(t.Context(), params)
	require.NoError(t, err)

	carrier, ok := result.(agent.TaintCarrier)
	require.True(t, ok)
	assert.Equal(t, []agent.RecordRef{{
		EntityType: agent.TaintEntityAgentMemory,
		ID:         dirty.ID.String(),
	}}, carrier.TaintedRecords())
	assert.Equal(t, agentID, recall.captured.AgentDefinitionID,
		"recall reads what was kept for the agent asking, and the organization's")
	assert.Equal(t, agent.ExternalReadMarked, newRecallMemoryTool(recall).Policy().ReadsExternal)
}

func TestRecallMemory_RefusesHalfASubject(t *testing.T) {
	t.Parallel()

	tool := newRecallMemoryTool(&fakeRecall{})

	_, err := tool.Query(t.Context(), testParams(map[string]any{"subjectId": "cus_1"}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "both or neither")
}
