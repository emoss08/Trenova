package agentruntime

import (
	"fmt"
	"strings"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

/*
A proposal the agent itself replaced, taken back before anyone sees it.

A turn's proposals are filed together when it ends, and two or more become one
plan the person approves whole. An agent that proposed create_shipment, saw it
had sent a field nobody asked for and proposed the corrected one told the
person to reject the first; the approval box offered both as one plan, the
single approval ran both, and the load was entered twice. Withdrawing the
first while the turn is still open records it Superseded, so it is never
pending, never a plan step and never in the approval box.

Only a proposal filed earlier in the same turn can be withdrawn: one from an
earlier turn is already in front of the person, who rejects it there.
*/

const withdrawProposalName = "withdraw_proposal"

const withdrawProposalDescription = "Withdraw a proposal you filed earlier in this turn, " +
	"before the person sees it, when a later proposal corrects or replaces it: otherwise " +
	"they are asked to approve both, and approving both makes both changes. Pass the id " +
	"the earlier proposal's result gave. Only a proposal from this turn; one from an " +
	"earlier turn is already in front of the person, so tell them to reject it. " +
	"Withdrawing changes no record."

const (
	paramWithdrawReason   = "reason"
	maxWithdrawReasonRune = 200
	unofferedWithdrawal   = "withdraw_proposal is only for a proposal you filed earlier in " +
		"this turn, and this turn has filed none. Answer without it."
	delegatedWithdrawal = "A task handed to you cannot withdraw a proposal: it goes to the " +
		"person as filed. Say in your answer which of your proposals replaces which."
)

func withdrawProposalSpec() serviceports.ToolSpec {
	return serviceports.ToolSpec{
		Name:        withdrawProposalName,
		Description: withdrawProposalDescription,
		Parameters: map[string]any{
			toolschema.KeyType: toolschema.TypeObject,
			toolschema.KeyProperties: map[string]any{
				paramProposalID: map[string]any{
					toolschema.KeyType: toolschema.TypeString,
					toolschema.KeyDescription: "The id of the proposal to withdraw, exactly as " +
						"its result in this turn gave it.",
				},
				paramWithdrawReason: map[string]any{
					toolschema.KeyType:      toolschema.TypeString,
					toolschema.KeyMaxLength: maxWithdrawReasonRune,
					toolschema.KeyDescription: "Optional. What replaced it, in a short phrase, " +
						"e.g. \"resent without the tractor type\".",
				},
			},
			toolschema.KeyRequired:             []string{paramProposalID},
			toolschema.KeyAdditionalProperties: false,
		},
	}
}

// offerWithdrawal adds withdraw_proposal to the turn once it has filed a
// proposal a person will decide. A delegate's proposals are its own turn's
// to answer for, so a delegated turn is never offered it.
func (t *Turn) offerWithdrawal(action *serviceports.PendingAction) {
	if action == nil || !action.Waiting() || t.req.Delegation != nil || t.tools == nil {
		return
	}

	t.tools.add(withdrawProposalSpec())
}

func (t *Turn) withdrawProposal(arguments map[string]any) toolOutcome {
	raw := strings.TrimSpace(stringArg(arguments, paramProposalID))
	id, err := pulid.Parse(raw)
	if raw == "" || err != nil {
		return failedOutcome("Tool %q was not run: proposalId must be the id of a proposal "+
			"you filed earlier in this turn, as its result gave it.", withdrawProposalName)
	}

	action := t.filedAction(id)
	switch {
	case action == nil:
		if outcome, found := proposalOutcome(t.req.Proposals, id); found {
			return failedOutcome("Proposal %s (%s) was filed in an earlier turn and is "+
				"already in front of the person, so it cannot be withdrawn here. If it is "+
				"still waiting and a later proposal replaces it, tell the person to reject "+
				"it in the approval box.", id, outcome.ToolName)
		}

		return failedOutcome("There is no proposal %s filed in this turn. Use the id the "+
			"earlier proposal's result gave.", id)
	case action.Withdrawn:
		return toolOutcome{content: fmt.Sprintf("Proposal %s (%s) is already withdrawn.",
			id, action.ToolName)}
	case action.Executed || action.Simulated:
		return failedOutcome("Proposal %s (%s) already ran when it was filed, so there is "+
			"nothing to withdraw. If it should be undone, say so and propose the change that "+
			"undoes it.", id, action.ToolName)
	}

	action.Withdrawn = true
	if _, requested := t.decisions[id]; requested {
		delete(t.decisions, id)
	}

	content := fmt.Sprintf("Withdrew proposal %s (%s). It will not be offered to the person "+
		"and nothing has changed. Do not ask them to reject it.", id, action.ToolName)
	if waiting := t.waitingCount(); waiting > 0 {
		content += fmt.Sprintf(" %s waiting for their decision from this turn.",
			countOfProposals(waiting))
	}

	summary := "Withdrew " + action.ToolName
	if reason := stringArg(arguments, paramWithdrawReason); strings.TrimSpace(reason) != "" {
		summary += ": " + reason
	}

	return toolOutcome{content: content, summary: summaryLine(summary)}
}

func (t *Turn) filedAction(id pulid.ID) *serviceports.PendingAction {
	for idx := range t.result.Actions {
		if t.result.Actions[idx].ProposalID == id {
			return &t.result.Actions[idx]
		}
	}

	return nil
}

func (t *Turn) waitingCount() int {
	count := 0
	for idx := range t.result.Actions {
		if t.result.Actions[idx].Waiting() {
			count++
		}
	}

	return count
}

// unwithdrawn is a turn's actions without the ones the agent withdrew: what
// is still headed for a person's decision, and what already ran.
func unwithdrawn(actions []serviceports.PendingAction) []serviceports.PendingAction {
	kept := make([]serviceports.PendingAction, 0, len(actions))
	for idx := range actions {
		if !actions[idx].Withdrawn {
			kept = append(kept, actions[idx])
		}
	}

	return kept
}

func countOfProposals(n int) string {
	if n == 1 {
		return "1 proposal is still"
	}

	return fmt.Sprintf("%d proposals are still", n)
}
