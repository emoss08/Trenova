package decisionnotes

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

/*
The agent filed a schedule while every agent in the organization was paused
and told the person no one could approve it yet. The person resumed agents and
approved it. Told only "approved, and it ran", the agent said it had had no
basis for the pause and apologised for the truth. The note now says the hold
was lifted in between, and that what it said then stands.
*/
func TestLiftedHold_TellsTheAgentAnApprovedProposalWasHeldWhenFiled(t *testing.T) {
	t.Parallel()

	proposalID := pulid.MustNew("ap_")
	history := []conversation.Message{
		{Role: conversation.RoleUser, Content: "Schedule this to run everyday at 10:15 AM EST"},
		{
			Role:     conversation.RoleTool,
			ToolName: "agent_create_schedule",
			Content: "Recorded a proposal to run \"agent_create_schedule\" (proposal " +
				proposalID.String() + ") at the Propose tier. It has not run, and no one can " +
				"approve it yet: every agent in this organization is paused in AI Control, so " +
				"a person must resume agents there before anyone can approve it. Tell the " +
				"person that, and do not say it waits on their approval.",
		},
	}
	executed := &agent.AgentProposal{
		ID:       proposalID,
		ToolName: "agent_create_schedule",
		Status:   agent.ProposalStatusExecuted,
	}

	note := liftedHold(history, executed)
	assert.Contains(t, note, "every agent in this organization was paused in AI Control")
	assert.Contains(t, note, "agents have been resumed since")
	assert.Contains(t, note, "do not take it back or apologise for it")

	rejected := *executed
	rejected.Status = agent.ProposalStatusRejected
	assert.Empty(t, liftedHold(history, &rejected), "a rejection lifted nothing")

	other := *executed
	other.ID = pulid.MustNew("ap_")
	assert.Empty(t, liftedHold(history, &other), "only the hold on this proposal is read")
}

/*
The agent proposed posting an invoice and repeated the card: a Pending
journal entry booked, no accounting connection. Told only "approved, and it
ran. Now: Posted, unpaid", it apologised for having "described a journal
entry that no tool result showed". The note now quotes the card the person
approved, as what ran.
*/
func TestCardNote_QuotesTheCardOfAProposalThatRan(t *testing.T) {
	t.Parallel()

	proposalID := pulid.MustNew("ap_")
	history := []conversation.Message{
		{
			Role:     conversation.RoleTool,
			ToolName: "invoice_post",
			Content: "Recorded a proposal to run \"invoice_post\" (proposal " + proposalID.String() +
				"). It is awaiting a person's review at the Propose tier and has not run.\n" +
				"The card shows the person what this proposal would do.\n" +
				"Summary: Would post Invoice INV2511000148 to Crown Holdings for 2582.22 USD; " +
				"a Pending journal entry booked.",
		},
	}
	executed := &agent.AgentProposal{
		ID:       proposalID,
		ToolName: "invoice_post",
		Status:   agent.ProposalStatusExecuted,
	}

	note := cardNote(history, executed)
	assert.Contains(t, note, "The card the person approved said:")
	assert.Contains(t, note, "a Pending journal entry booked.")
	assert.Contains(t, note, "what the card lists was done")

	failed := *executed
	failed.Status = agent.ProposalStatusExecutionFailed
	assert.Empty(t, cardNote(history, &failed), "a write that failed did not do what the card said")

	other := *executed
	other.ID = pulid.MustNew("ap_")
	assert.Empty(t, cardNote(history, &other), "only this proposal's card is quoted")
}
