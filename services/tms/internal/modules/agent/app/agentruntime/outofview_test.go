package agentruntime

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

/*
A delegate raised agent_raise_exception, the person approved it, and the next reply
said the case was still waiting on its card: the delegate's steps are never
replayed, so the ledger had no call to swap, and the delegate_task result the
model did read still said the card was waiting. The decided state now reaches
the model beside the question.
*/
func TestOutOfViewDecisions_TellsTheModelWhatBecameOfADelegatesProposal(t *testing.T) {
	t.Parallel()

	visible := pulid.MustNew("amsg_")
	history := []conversation.Message{
		{ID: pulid.MustNew("amsg_"), Role: conversation.RoleUser, Content: "Build it"},
		{ID: visible, Role: conversation.RoleAssistant, Content: "Handed it on."},
	}

	note := outOfViewDecisions(history, []agentports.ProposalOutcome{
		{
			SourceMessageID: pulid.MustNew("amsg_"),
			ToolName:        "agent_raise_exception",
			Status:          agent.ProposalStatusExecuted,
			AutonomyTier:    agent.TierPropose,
		},
		{
			SourceMessageID: visible,
			ToolName:        "report_update",
			Status:          agent.ProposalStatusRejected,
		},
		{
			SourceMessageID: pulid.MustNew("amsg_"),
			ToolName:        "report_create",
			Status:          agent.ProposalStatusPending,
		},
		{
			SourceMessageID: pulid.MustNew("amsg_"),
			ToolName:        "report_save_table_view",
			Status:          agent.ProposalStatusExecuted,
			AutonomyTier:    agent.TierAutoExecute,
		},
		{ToolName: "home_add_widget", Status: agent.ProposalStatusRejected},
	})

	assert.Contains(t, note, `approved the proposal to run "agent_raise_exception"`)
	assert.NotContains(t, note, "report_update", "the ledger already swaps a call in view")
	assert.NotContains(t, note, "report_create", "a waiting card is what the model was told")
	assert.NotContains(t, note, "report_save_table_view", "a write that ran on its own was not decided")
	assert.NotContains(t, note, "home_add_widget", "a proposal with no source is not placed")
	assert.True(t, strings.HasSuffix(note, "]\n\n"), "the note stands apart from the question")
}

func TestOutOfViewDecisions_SaysNothingWhenEverythingIsInView(t *testing.T) {
	t.Parallel()

	source := pulid.MustNew("amsg_")
	history := []conversation.Message{{ID: source, Role: conversation.RoleAssistant}}

	assert.Empty(t, outOfViewDecisions(history, []agentports.ProposalOutcome{{
		SourceMessageID: source,
		ToolName:        "report_update",
		Status:          agent.ProposalStatusExecuted,
	}}))
	assert.Empty(t, outOfViewDecisions(history, nil))
}

// A thread's whole record of decisions is its Decision entries. The note
// carries the newest few, oldest first.
func TestOutOfViewDecisions_KeepsTheNewestFew(t *testing.T) {
	t.Parallel()

	outcomes := make([]agentports.ProposalOutcome, 0, maxOutOfViewDecisions+2)
	names := make([]string, 0, cap(outcomes))
	for idx := range maxOutOfViewDecisions + 2 {
		name := "tool_" + string(rune('a'+idx))
		names = append(names, name)
		outcomes = append(outcomes, agentports.ProposalOutcome{
			SourceMessageID: pulid.MustNew("amsg_"),
			ToolName:        name,
			Status:          agent.ProposalStatusRejected,
		})
	}

	note := outOfViewDecisions(nil, outcomes)

	assert.NotContains(t, note, `"`+names[0]+`"`)
	assert.NotContains(t, note, `"`+names[1]+`"`)
	last := names[len(names)-1]
	first := names[2]
	assert.Contains(t, note, `"`+first+`"`)
	assert.Less(t, strings.Index(note, `"`+first+`"`), strings.Index(note, `"`+last+`"`))
}
