package runtimetools

import (
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/argcheck"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolcall"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolsummary"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

/*
A proposal the agent itself replaced, taken back before anyone sees it.

A turn's proposals are filed together when it ends, and two or more become one
plan the person approves whole. An agent that proposed shipment_create, saw it
had sent a field nobody asked for and proposed the corrected one told the
person to reject the first; the approval box offered both as one plan, the
single approval ran both, and the load was entered twice. Withdrawing the
first while the turn is still open records it Superseded, so it is never
pending, never a plan step and never in the approval box.

Only a proposal filed earlier in the same turn can be withdrawn: one from an
earlier turn is already in front of the person, who rejects it there.
*/

const WithdrawProposalName = "withdraw_proposal"

const withdrawProposalDescription = "Withdraw a proposal you filed earlier in this turn, " +
	"before the person sees it, when a later proposal corrects or replaces it: otherwise " +
	"they are asked to approve both, and approving both makes both changes. Pass the id " +
	"the earlier proposal's result gave. Only a proposal from this turn; one from an " +
	"earlier turn is already in front of the person, so tell them to reject it. " +
	"Withdrawing changes no record."

const (
	paramWithdrawReason   = "reason"
	maxWithdrawReasonRune = 200
	UnofferedWithdrawal   = "withdraw_proposal is only for a proposal you filed earlier in " +
		"this turn, and this turn has filed none. Answer without it."
	delegatedWithdrawal = "A task handed to you cannot withdraw a proposal: it goes to the " +
		"person as filed. Say in your answer which of your proposals replaces which."
)

func WithdrawProposalSpec() kernelports.ToolSpec {
	return kernelports.ToolSpec{
		Name:        WithdrawProposalName,
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

func hasFiledProposal(t Turn) bool {
	return slices.ContainsFunc(
		t.Result().Actions,
		func(action agentruntimeports.PendingAction) bool {
			return !action.Executed && !action.Simulated
		},
	)
}

func withdrawProposal(t Turn, arguments map[string]any) toolcall.Result {
	raw := strings.TrimSpace(argcheck.String(arguments, paramProposalID))
	id, err := pulid.Parse(raw)
	if raw == "" || err != nil {
		return toolcall.Failed(
			"Tool %q was not run: proposalId must be the id of a proposal "+
				"you filed earlier in this turn, as its result gave it.",
			WithdrawProposalName,
		)
	}

	action := filedAction(t, id)
	switch {
	case action == nil:
		if outcome, found := proposalOutcome(t.Request().Proposals, id); found {
			return toolcall.Failed("Proposal %s (%s) was filed in an earlier turn and is "+
				"already in front of the person, so it cannot be withdrawn here. If it is "+
				"still waiting and a later proposal replaces it, tell the person to reject "+
				"it in the approval box.", id, outcome.ToolName)
		}

		return toolcall.Failed("There is no proposal %s filed in this turn. Use the id the "+
			"earlier proposal's result gave.", id)
	case action.Withdrawn:
		return toolcall.Result{Content: fmt.Sprintf("Proposal %s (%s) is already withdrawn.",
			id, action.ToolName)}
	case action.Executed || action.Simulated:
		return toolcall.Failed(
			"Proposal %s (%s) already ran when it was filed, so there is "+
				"nothing to withdraw. If it should be undone, say so and propose the change that "+
				"undoes it.",
			id,
			action.ToolName,
		)
	}

	action.Withdrawn = true
	delete(t.Decisions(), id)

	content := fmt.Sprintf("Withdrew proposal %s (%s). It will not be offered to the person "+
		"and nothing has changed. Do not ask them to reject it.", id, action.ToolName)
	if waiting := waitingCount(t); waiting > 0 {
		content += fmt.Sprintf(" %s waiting for their decision from this turn.",
			countOfProposals(waiting))
	}

	summary := "Withdrew " + action.ToolName
	if reason := argcheck.String(arguments, paramWithdrawReason); strings.TrimSpace(
		reason,
	) != "" {
		summary += ": " + reason
	}

	return toolcall.Result{Content: content, Summary: toolsummary.Line(summary)}
}

func filedAction(t Turn, id pulid.ID) *agentruntimeports.PendingAction {
	for idx := range t.Result().Actions {
		if t.Result().Actions[idx].ProposalID == id {
			return &t.Result().Actions[idx]
		}
	}

	return nil
}

func waitingCount(t Turn) int {
	count := 0
	for idx := range t.Result().Actions {
		if t.Result().Actions[idx].Waiting() {
			count++
		}
	}

	return count
}

// Unwithdrawn is a turn's actions without the ones the agent withdrew: what
// is still headed for a person's decision, and what already ran.
func Unwithdrawn(actions []agentruntimeports.PendingAction) []agentruntimeports.PendingAction {
	kept := make([]agentruntimeports.PendingAction, 0, len(actions))
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
