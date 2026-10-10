package agentruntime

import (
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

const requestDecisionName = "request_decision"

const requestDecisionDescription = "Open a proposal that is waiting on the person in the " +
	"approval box under the conversation, so they can approve or reject it there. Call it " +
	"when the person types an approval or asks you to go ahead with a change listed under " +
	"Proposals awaiting a decision: a typed \"yes\" does not decide a proposal, only the " +
	"approval box does. Pass that proposal's id from the list. For several waiting proposals " +
	"of one tool, pass them all in proposalIds so the person decides them together; for a " +
	"plan's steps, pass its planId. Calling it changes nothing and decides nothing."

const (
	undecidableRefusal = "There is no approval box here. Tell the person the proposal is " +
		"still waiting on their decision in this conversation, and that typing does not " +
		"approve it."
	unofferedDecisionRefusal = "request_decision is only for a conversation with a proposal " +
		"waiting on the person, and this one has none. Answer without it."
	maxRequestedProposals = 50
	paramProposalID       = "proposalId"
	paramProposalIDs      = "proposalIds"
	paramPlanID           = "planId"
)

func requestDecisionSpec() serviceports.ToolSpec {
	return serviceports.ToolSpec{
		Name:        requestDecisionName,
		Description: requestDecisionDescription,
		Parameters: map[string]any{
			toolschema.KeyType: toolschema.TypeObject,
			toolschema.KeyProperties: map[string]any{
				paramProposalID: map[string]any{
					toolschema.KeyType: toolschema.TypeString,
					toolschema.KeyDescription: "The id of the waiting proposal, exactly as " +
						"the Proposals awaiting a decision list gives it.",
				},
				paramProposalIDs: map[string]any{
					toolschema.KeyType:     toolschema.TypeArray,
					toolschema.KeyMinItems: 2,
					toolschema.KeyMaxItems: maxRequestedProposals,
					toolschema.KeyItems: map[string]any{
						toolschema.KeyType: toolschema.TypeString,
					},
					toolschema.KeyDescription: "Several waiting proposals of the same tool, " +
						"decided together; instead of proposalId.",
				},
				paramPlanID: map[string]any{
					toolschema.KeyType: toolschema.TypeString,
					toolschema.KeyDescription: "A waiting plan, decided whole; instead " +
						"of proposalId.",
				},
			},
			toolschema.KeyAdditionalProperties: false,
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

type requestedDecision struct {
	proposalIDs []pulid.ID
	planID      pulid.ID
}

var requestedNothing = fmt.Sprintf("Tool %q was not run: proposalId must be the id of a "+
	"proposal listed under Proposals awaiting a decision, proposalIds several of them, or "+
	"planId the id of a waiting plan.", requestDecisionName)

func requestedIDs(arguments map[string]any) (request requestedDecision, refusal string) {
	given := 0

	if raw := strings.TrimSpace(stringArg(arguments, paramProposalID)); raw != "" {
		given++
		id, err := pulid.Parse(raw)
		if err != nil {
			return requestedDecision{}, requestedNothing
		}
		request.proposalIDs = []pulid.ID{id}
	}
	if raw, ok := arguments[paramProposalIDs].([]any); ok && len(raw) > 0 {
		given++
		if len(raw) > maxRequestedProposals {
			return requestedDecision{}, fmt.Sprintf("Tool %q was not run: proposalIds holds "+
				"at most %d proposals.", requestDecisionName, maxRequestedProposals)
		}
		ids := make([]pulid.ID, 0, len(raw))
		for _, value := range raw {
			text, isText := value.(string)
			id, err := pulid.Parse(strings.TrimSpace(text))
			if !isText || err != nil {
				return requestedDecision{}, requestedNothing
			}
			if !slices.Contains(ids, id) {
				ids = append(ids, id)
			}
		}
		request.proposalIDs = ids
	}
	if raw := strings.TrimSpace(stringArg(arguments, paramPlanID)); raw != "" {
		given++
		id, err := pulid.Parse(raw)
		if err != nil {
			return requestedDecision{}, requestedNothing
		}
		request.planID = id
	}

	switch given {
	case 0:
		return requestedDecision{}, requestedNothing
	case 1:
		return request, ""
	default:
		return requestedDecision{}, fmt.Sprintf("Tool %q was not run: give only one of "+
			"proposalId, proposalIds and planId.", requestDecisionName)
	}
}

func (t *Turn) requestDecision(arguments map[string]any) toolOutcome {
	request, refusal := requestedIDs(arguments)
	if refusal != "" {
		return failedOutcome("%s", refusal)
	}
	if request.planID.IsNotNil() {
		return t.requestPlanDecision(request.planID)
	}
	if len(request.proposalIDs) == 1 {
		return t.requestProposalDecision(request.proposalIDs[0])
	}

	return t.requestBunchDecision(request.proposalIDs)
}

func (t *Turn) requestProposalDecision(id pulid.ID) toolOutcome {
	if filed, ok := t.filedThisTurn(id); ok {
		return filed
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
		return toolOutcome{content: fmt.Sprintf("Proposal %s (%s) is already in the "+
			"approval box from your earlier call. End your turn with one short line pointing "+
			"the person to it.", id, outcome.ToolName)}
	}
	t.decisions[id] = struct{}{}

	return decisionOutcome(serviceports.DecisionRequest{ProposalID: id, ToolName: outcome.ToolName})
}

func (t *Turn) filedThisTurn(id pulid.ID) (toolOutcome, bool) {
	for idx := range t.result.Actions {
		action := &t.result.Actions[idx]
		if action.ProposalID == id && action.Withdrawn {
			return failedOutcome("Proposal %s (%s) was withdrawn in this turn and is not "+
				"waiting on anyone.", id, action.ToolName), true
		}
		if action.ProposalID == id && !action.Executed {
			return toolOutcome{content: fmt.Sprintf("Proposal %s (%s) was filed in this turn, "+
				"and it is already in the approval box in front of the person. End your turn "+
				"with one short line pointing them to it.", id, action.ToolName)}, true
		}
	}

	return toolOutcome{}, false
}

func (t *Turn) requestPlanDecision(planID pulid.ID) toolOutcome {
	steps := make([]pulid.ID, 0, len(t.req.Proposals))
	for idx := range t.req.Proposals {
		proposal := &t.req.Proposals[idx]
		if proposal.PlanID == planID && proposal.Pending() && proposal.ProposalID.IsNotNil() {
			steps = append(steps, proposal.ProposalID)
		}
	}
	if len(steps) == 0 {
		return failedOutcome("There is no plan %s waiting on the person in this "+
			"conversation. Use a planId from Proposals awaiting a decision, or tell them what "+
			"became of it.", planID)
	}
	if _, requested := t.decisions[planID]; requested {
		return toolOutcome{content: fmt.Sprintf("Plan %s is already in the approval box from "+
			"your earlier call. End your turn with one short line pointing the person to it.",
			planID)}
	}
	t.decisions[planID] = struct{}{}

	return decisionOutcome(serviceports.DecisionRequest{
		ProposalID:  steps[0],
		ProposalIDs: steps,
		PlanID:      planID,
		ToolName:    agent.PlanToolName,
	})
}

func (t *Turn) requestBunchDecision(ids []pulid.ID) toolOutcome {
	tool := ""
	for _, id := range ids {
		if filed, ok := t.filedThisTurn(id); ok {
			return filed
		}
		outcome, found := proposalOutcome(t.req.Proposals, id)
		switch {
		case !found:
			return failedOutcome("There is no proposal %s in this conversation. Use ids "+
				"listed under Proposals awaiting a decision.", id)
		case !outcome.Pending():
			return failedOutcome("Proposal %s (%s) is no longer waiting on the person: it is "+
				"%s. Leave it out and tell them what became of it.",
				id, outcome.ToolName, strings.ToLower(string(outcome.Status)))
		case outcome.PlanID.IsNotNil():
			return failedOutcome("Proposal %s is a step of plan %s, which is decided whole; "+
				"pass that planId instead.", id, outcome.PlanID)
		case tool != "" && outcome.ToolName != tool:
			return failedOutcome("proposalIds decides one tool at a time, and "+
				"these are %s and %s; call request_decision once for each tool.",
				tool, outcome.ToolName)
		}
		tool = outcome.ToolName
	}

	fresh := false
	for _, id := range ids {
		if _, requested := t.decisions[id]; !requested {
			fresh = true
		}
	}
	if !fresh {
		return toolOutcome{content: fmt.Sprintf("These %d proposals are already in the "+
			"approval box from your earlier call. End your turn with one short line pointing "+
			"the person to it.", len(ids))}
	}
	for _, id := range ids {
		t.decisions[id] = struct{}{}
	}

	return decisionOutcome(serviceports.DecisionRequest{
		ProposalID:  ids[0],
		ProposalIDs: ids,
		ToolName:    tool,
	})
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
	request, refusal := requestedIDs(arguments)
	if refusal != "" {
		return failedOutcome("%s", refusal)
	}
	if request.planID.IsNotNil() {
		return decisionOutcome(serviceports.DecisionRequest{PlanID: request.planID})
	}
	if len(request.proposalIDs) == 1 {
		return decisionOutcome(serviceports.DecisionRequest{ProposalID: request.proposalIDs[0]})
	}

	return decisionOutcome(serviceports.DecisionRequest{
		ProposalID:  request.proposalIDs[0],
		ProposalIDs: request.proposalIDs,
	})
}

func decisionRequested(
	observe serviceports.ToolObserver,
	call *serviceports.ToolCall,
	request serviceports.DecisionRequest,
) toolOutcome {
	if observe == nil {
		return failedOutcome("%s", undecidableRefusal)
	}

	shown, err := observe(serviceports.ToolObservation{Call: *call, Data: request})
	switch {
	case err != nil:
		return failedOutcome("Proposal %s could not be opened in the approval box: %s. Tell "+
			"the person the proposal is still waiting on their decision in this "+
			"conversation, and that typing does not approve it.", request.ProposalID, err.Error())
	case shown == nil:
		return failedOutcome("%s", undecidableRefusal)
	}

	subject := fmt.Sprintf("proposal %s (%s)", request.ProposalID, shown.Title)
	switch {
	case request.PlanID.IsNotNil():
		subject = fmt.Sprintf("plan %s (%s)", request.PlanID, shown.Title)
	case len(request.ProposalIDs) > 1:
		subject = fmt.Sprintf("%d proposals (%s)", len(request.ProposalIDs), shown.Title)
	}

	return toolOutcome{
		content: fmt.Sprintf("The approval box under the conversation now shows %s. "+
			"The person approves or rejects it there; a typed \"yes\" does not decide it, and "+
			"nothing has changed yet. End your turn with one short line pointing them to the "+
			"approval box, and do not propose the change again.", subject),
		summary: summaryLine(shown.Title),
	}
}

func UnkeptOutcome(name string) ToolOutcome {
	if name == requestDecisionName {
		return ToolOutcome{
			Content: "The approval box could not be opened just now. Tell the person the " +
				"proposal is still waiting on their decision in this conversation, and that " +
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
