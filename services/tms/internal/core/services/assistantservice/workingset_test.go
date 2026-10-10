package assistantservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTouchedRecords_CollectsWhatTheTurnWasAbout(t *testing.T) {
	t.Parallel()

	subject := pulid.MustNew("shp_")
	mentioned := pulid.MustNew("wrk_")
	read := pulid.MustNew("inv_")
	failed := pulid.MustNew("trac_")
	delegated := pulid.MustNew("tr_")
	wrote := pulid.MustNew("cus_")

	thread := &conversation.Thread{
		SubjectType: agent.SubjectShipment,
		SubjectID:   subject,
	}
	saved := []conversation.Message{
		{Role: conversation.RoleUser, Content: "where is it"},
		{Role: conversation.RoleAssistant, ToolCalls: []conversation.ToolCallRecord{
			{ID: "c1", Name: "get_invoice", Arguments: map[string]any{"invoiceId": read.String()}},
			{ID: "c2", Name: "get_tractor", Arguments: map[string]any{"tractorId": failed.String()}},
			{ID: "c3", Name: "list_shipments", Arguments: map[string]any{"customerId": wrote.String()}},
		}},
		{Role: conversation.RoleTool, ToolCallID: "c1", ToolSummary: "INV-7"},
		{Role: conversation.RoleTool, ToolCallID: "c2", ToolFailed: true},
		{Role: conversation.RoleTool, ToolCallID: "c3"},
		{Role: conversation.RoleAssistant, Kind: conversation.MessageKindDelegated, ToolCalls: []conversation.ToolCallRecord{
			{ID: "d1", Name: "get_trailer", Arguments: map[string]any{"trailerId": delegated.String()}},
		}},
	}
	actions := []services.PendingAction{
		{Executed: true, Target: &services.ProposalTarget{Resource: permission.ResourceCustomer, ID: wrote}},
		{Executed: false, Target: &services.ProposalTarget{Resource: permission.ResourceCustomer, ID: pulid.MustNew("cus_")}},
	}

	touched := touchedRecords(&workingTurn{
		thread: thread,
		plan: &TurnPlan{Mentions: []agent.EntityRef{
			{Type: "worker", ID: mentioned.String(), Label: "Jane Doe"},
		}},
		saved:   saved,
		actions: actions,
		at:      100,
	})

	sources := make(map[string]conversation.WorkingSource, len(touched))
	for _, record := range touched {
		sources[record.ID] = record.Source
	}
	assert.Equal(t, conversation.WorkingSourceSubject, sources[subject.String()])
	assert.Equal(t, conversation.WorkingSourceMention, sources[mentioned.String()])
	assert.Equal(t, conversation.WorkingSourceRead, sources[read.String()])
	assert.Equal(t, conversation.WorkingSourceWrote, sources[wrote.String()])
	assert.NotContains(t, sources, failed.String(), "a read that failed read nothing")
	assert.NotContains(t, sources, delegated.String(), "another agent's steps are its own")
	require.Len(t, touched, 4)

	for _, record := range touched {
		if record.ID == read.String() {
			assert.Equal(t, "INV-7", record.Label)
		}
	}
}

func TestAnchorRefs_PutsThisTurnFirstAndLeavesOutTheSubject(t *testing.T) {
	t.Parallel()

	subject := pulid.MustNew("shp_")
	older, _ := conversation.NewWorkingRecord(pulid.MustNew("inv_").String(), "INV-1",
		conversation.WorkingSourceRead, 5)
	subjectRecord, _ := conversation.NewWorkingRecord(subject.String(), "",
		conversation.WorkingSourceSubject, 6)
	mentioned := pulid.MustNew("wrk_").String()

	refs := anchorRefs(&anchorScope{
		thread: &conversation.Thread{
			SubjectType: agent.SubjectShipment,
			SubjectID:   subject,
			WorkingSet:  []conversation.WorkingRecord{subjectRecord, older},
		},
		mentions: []agent.EntityRef{{Type: "worker", ID: mentioned, Label: "Jane Doe"}},
		page:     &agent.PageContext{EntityType: "invoice", EntityID: older.ID, Title: "INV-1"},
	})

	require.Len(t, refs, 2)
	assert.Equal(t, mentioned, refs[0].ID)
	assert.Equal(t, older.ID, refs[1].ID, "named by the page and the set, read once")
}

/*
A load the person approved into being is what the conversation is about next.
The Peak load was created by an approval, so no turn's own write held it, and
"who can run this one?" reached dispatch with nothing to say which one.
*/
func TestTouchedRecords_KeepsWhatAnApprovedOrAutomaticWriteMade(t *testing.T) {
	t.Parallel()

	approved := pulid.MustNew("shp_").String()
	older := pulid.MustNew("shp_").String()
	automatic := pulid.MustNew("loc_").String()
	earlier := int64(1_000)
	later := int64(2_000)

	turn := &workingTurn{
		thread: &conversation.Thread{WorkingSet: []conversation.WorkingRecord{
			{Kind: "customer", ID: pulid.MustNew("cus_").String(), TouchedAt: 1_500},
		}},
		plan: &TurnPlan{FollowUp: true, Proposals: []services.ProposalOutcome{
			{
				Status:          agent.ProposalStatusExecuted,
				ExecutedAt:      &later,
				ExecutionResult: &agent.ToolExecutionResult{Name: "S2610", Record: &agent.RecordRef{EntityType: "shipment", ID: approved}},
			},
			{
				Status:          agent.ProposalStatusExecuted,
				ExecutedAt:      &earlier,
				ExecutionResult: &agent.ToolExecutionResult{Record: &agent.RecordRef{EntityType: "shipment", ID: older}},
			},
		}},
		actions: []services.PendingAction{{
			Executed:        true,
			ExecutionResult: &agent.ToolExecutionResult{Record: &agent.RecordRef{EntityType: "location", ID: automatic}},
		}},
		at: 2_100,
	}

	ids := make([]string, 0, 3)
	for _, record := range touchedRecords(turn) {
		ids = append(ids, record.ID)
		assert.Equal(t, conversation.WorkingSourceWrote, record.Source)
	}
	assert.ElementsMatch(t, []string{approved, automatic}, ids,
		"a write already behind the conversation's records is not touched again")
}
