package agentruntime

import (
	"fmt"
	"strings"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
)

const requestDecisionName = "request_decision"

const requestDecisionDescription = "Put a proposal that is waiting on the person back in " +
	"front of them as its card, so they can approve or reject it there. Call it when the " +
	"person types an approval or asks you to go ahead with a change listed under Proposals " +
	"awaiting a decision: a typed \"yes\" does not decide a proposal, only the card does. " +
	"Pass that proposal's id from the list. Calling it changes nothing and decides nothing."

const (
	undecidableRefusal = "There is nowhere to show the card here. Tell the person the " +
		"proposal is still waiting on its card earlier in this conversation, and that typing " +
		"does not approve it."
	unofferedDecisionRefusal = "request_decision is only for a conversation with a proposal " +
		"waiting on the person, and this one has none. Answer without it."
)

func requestDecisionSpec() serviceports.ToolSpec {
	return serviceports.ToolSpec{
		Name:        requestDecisionName,
		Description: requestDecisionDescription,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"proposalId": map[string]any{
					"type": "string",
					"description": "The id of the waiting proposal, exactly as the " +
						"Proposals awaiting a decision list gives it.",
				},
			},
			"required":             []string{"proposalId"},
			"additionalProperties": false,
		},
	}
}

func pendingDecisions(proposals []serviceports.ProposalOutcome) bool {
	for idx := range proposals {
		if proposals[idx].Pending() && proposals[idx].ProposalID.IsNotNil() {
			return true
		}
	}

	return false
}

func requestedProposalID(arguments map[string]any) (pulid.ID, bool) {
	raw := strings.TrimSpace(stringArg(arguments, "proposalId"))
	if raw == "" {
		return pulid.Nil, false
	}
	id, err := pulid.Parse(raw)
	if err != nil {
		return pulid.Nil, false
	}

	return id, true
}

func (t *Turn) requestDecision(arguments map[string]any) toolOutcome {
	id, ok := requestedProposalID(arguments)
	if !ok {
		return failedOutcome("Tool %q was not run: proposalId must be the id of a proposal "+
			"listed under Proposals awaiting a decision.", requestDecisionName)
	}

	for idx := range t.result.Actions {
		action := &t.result.Actions[idx]
		if action.ProposalID == id && !action.Executed {
			return toolOutcome{content: fmt.Sprintf("Proposal %s (%s) was filed in this turn, "+
				"and its card is already in front of the person. End your turn with one "+
				"short line pointing them to it.", id, action.ToolName)}
		}
	}

	outcome, found := proposalOutcome(t.req.Proposals, id)
	switch {
	case !found:
		return failedOutcome("There is no proposal %s in this conversation. Use an id "+
			"listed under Proposals awaiting a decision.", id)
	case !outcome.Pending():
		return failedOutcome("Proposal %s (%s) is no longer waiting on the person: it is %s. "+
			"Tell them what became of it rather than asking them to decide it.",
			id, outcome.ToolName, strings.ToLower(string(outcome.Status)))
	}

	if _, requested := t.decisions[id]; requested {
		return toolOutcome{content: fmt.Sprintf("The card for proposal %s (%s) is already "+
			"in front of the person from your earlier call. End your turn with one short "+
			"line pointing them to it.", id, outcome.ToolName)}
	}
	t.decisions[id] = struct{}{}

	return decisionOutcome(serviceports.DecisionRequest{ProposalID: id, ToolName: outcome.ToolName})
}

func proposalOutcome(
	proposals []serviceports.ProposalOutcome,
	id pulid.ID,
) (serviceports.ProposalOutcome, bool) {
	for idx := range proposals {
		if proposals[idx].ProposalID == id {
			return proposals[idx], true
		}
	}

	return serviceports.ProposalOutcome{}, false
}

func decisionOutcome(request serviceports.DecisionRequest) toolOutcome {
	return toolOutcome{publishes: true, data: request}
}

func decisionStepOutcome(arguments map[string]any) toolOutcome {
	id, ok := requestedProposalID(arguments)
	if !ok {
		return failedOutcome("Tool %q was not run: proposalId must be the id of a proposal "+
			"listed under Proposals awaiting a decision.", requestDecisionName)
	}

	return decisionOutcome(serviceports.DecisionRequest{ProposalID: id})
}

func decisionRequested(
	observe serviceports.ToolObserver,
	call serviceports.ToolCall,
	request serviceports.DecisionRequest,
) toolOutcome {
	if observe == nil {
		return failedOutcome("%s", undecidableRefusal)
	}

	shown, err := observe(serviceports.ToolObservation{Call: call, Data: request})
	switch {
	case err != nil:
		return failedOutcome("The card for proposal %s could not be shown again: %s. Tell "+
			"the person the proposal is still waiting on its card earlier in this "+
			"conversation, and that typing does not approve it.", request.ProposalID, err.Error())
	case shown == nil:
		return failedOutcome("%s", undecidableRefusal)
	}

	return toolOutcome{
		content: fmt.Sprintf("The card for proposal %s (%s) is in front of the person again. "+
			"They approve or reject it there; a typed \"yes\" does not decide it, and nothing "+
			"has changed yet. End your turn with one short line pointing them to the card, "+
			"and do not propose the change again.", request.ProposalID, shown.Title),
		summary: summaryLine(shown.Title),
	}
}

func UnkeptOutcome(name string) ToolOutcome {
	if name == requestDecisionName {
		return ToolOutcome{
			Content: "The card could not be shown again just now. Tell the person the " +
				"proposal is still waiting on its card earlier in this conversation, and that " +
				"typing does not approve it.",
			Failed: true,
		}
	}

	return ToolOutcome{
		Content: fmt.Sprintf("Tool %q could not keep the document just now. "+
			"Put the text in your reply instead.", name),
		Failed: true,
	}
}
