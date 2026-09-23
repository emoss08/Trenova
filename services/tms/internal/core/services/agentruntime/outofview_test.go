package agentruntime

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

/*
A delegate raised raise_exception, the person approved it, and the next reply
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

	note := outOfViewDecisions(history, []serviceports.ProposalOutcome{
		{
			SourceMessageID: pulid.MustNew("amsg_"),
			ToolName:        "raise_exception",
			Status:          agent.ProposalStatusExecuted,
			AutonomyTier:    agent.TierPropose,
		},
		{
			SourceMessageID: visible,
			ToolName:        "update_report",
			Status:          agent.ProposalStatusRejected,
		},
		{
			SourceMessageID: pulid.MustNew("amsg_"),
			ToolName:        "create_report",
			Status:          agent.ProposalStatusPending,
		},
		{
			SourceMessageID: pulid.MustNew("amsg_"),
			ToolName:        "save_table_view",
			Status:          agent.ProposalStatusExecuted,
			AutonomyTier:    agent.TierAutoExecute,
		},
		{ToolName: "add_home_widget", Status: agent.ProposalStatusRejected},
	})

	assert.Contains(t, note, `approved the proposal to run "raise_exception"`)
	assert.NotContains(t, note, "update_report", "the ledger already swaps a call in view")
	assert.NotContains(t, note, "create_report", "a waiting card is what the model was told")
	assert.NotContains(t, note, "save_table_view", "a write that ran on its own was not decided")
	assert.NotContains(t, note, "add_home_widget", "a proposal with no source is not placed")
	assert.True(t, strings.HasSuffix(note, "]\n\n"), "the note stands apart from the question")
}

func TestOutOfViewDecisions_SaysNothingWhenEverythingIsInView(t *testing.T) {
	t.Parallel()

	source := pulid.MustNew("amsg_")
	history := []conversation.Message{{ID: source, Role: conversation.RoleAssistant}}

	assert.Empty(t, outOfViewDecisions(history, []serviceports.ProposalOutcome{{
		SourceMessageID: source,
		ToolName:        "update_report",
		Status:          agent.ProposalStatusExecuted,
	}}))
	assert.Empty(t, outOfViewDecisions(history, nil))
}

// A thread's whole record of decisions is its Decision entries. The note
// carries the newest few, oldest first.
func TestOutOfViewDecisions_KeepsTheNewestFew(t *testing.T) {
	t.Parallel()

	outcomes := make([]serviceports.ProposalOutcome, 0, maxOutOfViewDecisions+2)
	names := make([]string, 0, cap(outcomes))
	for idx := range maxOutOfViewDecisions + 2 {
		name := "tool_" + string(rune('a'+idx))
		names = append(names, name)
		outcomes = append(outcomes, serviceports.ProposalOutcome{
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
