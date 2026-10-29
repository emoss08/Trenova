package runtimetools

import (
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/argcheck"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolcall"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolsummary"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

const RequestDecisionName = "request_decision"

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
	UnofferedDecisionRefusal = "request_decision is only for a conversation with a proposal " +
		"waiting on the person, and this one has none. Answer without it."
	maxRequestedProposals = 50
	paramProposalID       = "proposalId"
	paramProposalIDs      = "proposalIds"
	paramPlanID           = "planId"
)

func RequestDecisionSpec() kernelports.ToolSpec {
	return kernelports.ToolSpec{
		Name:        RequestDecisionName,
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

func PendingDecisions(proposals []agentruntimeports.ProposalOutcome) bool {
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
	"planId the id of a waiting plan.", RequestDecisionName)

func requestedIDs(arguments map[string]any) (request requestedDecision, refusal string) {
	given := 0

	if raw := strings.TrimSpace(argcheck.String(arguments, paramProposalID)); raw != "" {
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
				"at most %d proposals.", RequestDecisionName, maxRequestedProposals)
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
	if raw := strings.TrimSpace(argcheck.String(arguments, paramPlanID)); raw != "" {
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
			"proposalId, proposalIds and planId.", RequestDecisionName)
	}
}

func requestDecision(t Turn, arguments map[string]any) toolcall.Result {
	request, refusal := requestedIDs(arguments)
	if refusal != "" {
		return toolcall.Failed("%s", refusal)
	}
	if request.planID.IsNotNil() {
		return requestPlanDecision(t, request.planID)
	}
	if len(request.proposalIDs) == 1 {
		return requestProposalDecision(t, request.proposalIDs[0])
	}

	return requestBunchDecision(t, request.proposalIDs)
}

func requestProposalDecision(t Turn, id pulid.ID) toolcall.Result {
	if filed, ok := filedThisTurn(t, id); ok {
		return filed
	}

	outcome, found := proposalOutcome(t.Request().Proposals, id)
	switch {
	case !found:
		return toolcall.Failed("There is no proposal %s in this conversation. Use an id "+
			"listed under Proposals awaiting a decision.", id)
	case !outcome.Pending():
		return toolcall.Failed(
			"Proposal %s (%s) is no longer waiting on the person: it is %s. "+
				"Tell them what became of it rather than asking them to decide it.",
			id,
			outcome.ToolName,
			strings.ToLower(string(outcome.Status)),
		)
	}

	if _, requested := t.Decisions()[id]; requested {
		return toolcall.Result{Content: fmt.Sprintf("Proposal %s (%s) is already in the "+
			"approval box from your earlier call. End your turn with one short line pointing "+
			"the person to it.", id, outcome.ToolName)}
	}
	t.Decisions()[id] = struct{}{}

	return decisionOutcome(
		agentruntimeports.DecisionRequest{ProposalID: id, ToolName: outcome.ToolName},
	)
}

func filedThisTurn(t Turn, id pulid.ID) (toolcall.Result, bool) {
	for idx := range t.Result().Actions {
		action := &t.Result().Actions[idx]
		if action.ProposalID == id && action.Withdrawn {
			return toolcall.Failed("Proposal %s (%s) was withdrawn in this turn and is not "+
				"waiting on anyone.", id, action.ToolName), true
		}
		if action.ProposalID == id && !action.Executed {
			return toolcall.Result{Content: fmt.Sprintf("Proposal %s (%s) was filed in this turn, "+
				"and it is already in the approval box in front of the person. End your turn "+
				"with one short line pointing them to it.", id, action.ToolName)}, true
		}
	}

	return toolcall.Result{}, false
}

func requestPlanDecision(t Turn, planID pulid.ID) toolcall.Result {
	steps := make([]pulid.ID, 0, len(t.Request().Proposals))
	for idx := range t.Request().Proposals {
		proposal := &t.Request().Proposals[idx]
		if proposal.PlanID == planID && proposal.Pending() && proposal.ProposalID.IsNotNil() {
			steps = append(steps, proposal.ProposalID)
		}
	}
	if len(steps) == 0 {
		return toolcall.Failed("There is no plan %s waiting on the person in this "+
			"conversation. Use a planId from Proposals awaiting a decision, or tell them what "+
			"became of it.", planID)
	}
	if _, requested := t.Decisions()[planID]; requested {
		return toolcall.Result{Content: fmt.Sprintf("Plan %s is already in the approval box from "+
			"your earlier call. End your turn with one short line pointing the person to it.",
			planID)}
	}
	t.Decisions()[planID] = struct{}{}

	return decisionOutcome(agentruntimeports.DecisionRequest{
		ProposalID:  steps[0],
		ProposalIDs: steps,
		PlanID:      planID,
		ToolName:    agentplan.PlanToolName,
	})
}

func requestBunchDecision(t Turn, ids []pulid.ID) toolcall.Result {
	tool := ""
	for _, id := range ids {
		if filed, ok := filedThisTurn(t, id); ok {
			return filed
		}
		outcome, found := proposalOutcome(t.Request().Proposals, id)
		switch {
		case !found:
			return toolcall.Failed("There is no proposal %s in this conversation. Use ids "+
				"listed under Proposals awaiting a decision.", id)
		case !outcome.Pending():
			return toolcall.Failed(
				"Proposal %s (%s) is no longer waiting on the person: it is "+
					"%s. Leave it out and tell them what became of it.",
				id,
				outcome.ToolName,
				strings.ToLower(string(outcome.Status)),
			)
		case outcome.PlanID.IsNotNil():
			return toolcall.Failed(
				"Proposal %s is a step of plan %s, which is decided whole; "+
					"pass that planId instead.",
				id,
				outcome.PlanID,
			)
		case tool != "" && outcome.ToolName != tool:
			return toolcall.Failed("proposalIds decides one tool at a time, and "+
				"these are %s and %s; call request_decision once for each tool.",
				tool, outcome.ToolName)
		}
		tool = outcome.ToolName
	}

	fresh := false
	for _, id := range ids {
		if _, requested := t.Decisions()[id]; !requested {
			fresh = true
		}
	}
	if !fresh {
		return toolcall.Result{Content: fmt.Sprintf("These %d proposals are already in the "+
			"approval box from your earlier call. End your turn with one short line pointing "+
			"the person to it.", len(ids))}
	}
	for _, id := range ids {
		t.Decisions()[id] = struct{}{}
	}

	return decisionOutcome(agentruntimeports.DecisionRequest{
		ProposalID:  ids[0],
		ProposalIDs: ids,
		ToolName:    tool,
	})
}

func proposalOutcome(
	proposals []agentruntimeports.ProposalOutcome,
	id pulid.ID,
) (agentruntimeports.ProposalOutcome, bool) {
	for idx := range proposals {
		if proposals[idx].ProposalID == id {
			return proposals[idx], true
		}
	}

	return agentruntimeports.ProposalOutcome{}, false
}

func decisionOutcome(request agentruntimeports.DecisionRequest) toolcall.Result {
	return toolcall.Result{Publishes: true, Data: request}
}

func DecisionStepOutcome(arguments map[string]any) toolcall.Result {
	request, refusal := requestedIDs(arguments)
	if refusal != "" {
		return toolcall.Failed("%s", refusal)
	}
	if request.planID.IsNotNil() {
		return decisionOutcome(agentruntimeports.DecisionRequest{PlanID: request.planID})
	}
	if len(request.proposalIDs) == 1 {
		return decisionOutcome(
			agentruntimeports.DecisionRequest{ProposalID: request.proposalIDs[0]},
		)
	}

	return decisionOutcome(agentruntimeports.DecisionRequest{
		ProposalID:  request.proposalIDs[0],
		ProposalIDs: request.proposalIDs,
	})
}

func DecisionRequested(
	observe agentruntimeports.ToolObserver,
	call *llmports.ToolCall,
	request agentruntimeports.DecisionRequest,
) toolcall.Result {
	if observe == nil {
		return toolcall.Failed("%s", undecidableRefusal)
	}

	shown, err := observe(agentruntimeports.ToolObservation{Call: *call, Data: request})
	switch {
	case err != nil:
		return toolcall.Failed(
			"Proposal %s could not be opened in the approval box: %s. Tell "+
				"the person the proposal is still waiting on their decision in this "+
				"conversation, and that typing does not approve it.",
			request.ProposalID,
			err.Error(),
		)
	case shown == nil:
		return toolcall.Failed("%s", undecidableRefusal)
	}

	subject := fmt.Sprintf("proposal %s (%s)", request.ProposalID, shown.Title)
	switch {
	case request.PlanID.IsNotNil():
		subject = fmt.Sprintf("plan %s (%s)", request.PlanID, shown.Title)
	case len(request.ProposalIDs) > 1:
		subject = fmt.Sprintf("%d proposals (%s)", len(request.ProposalIDs), shown.Title)
	}

	return toolcall.Result{
		Content: fmt.Sprintf("The approval box under the conversation now shows %s. "+
			"The person approves or rejects it there; a typed \"yes\" does not decide it, and "+
			"nothing has changed yet. End your turn with one short line pointing them to the "+
			"approval box, and do not propose the change again.", subject),
		Summary: toolsummary.Line(shown.Title),
	}
}

func UnkeptOutcome(name string) toolcall.Outcome {
	if name == RequestDecisionName {
		return toolcall.Outcome{
			Content: "The approval box could not be opened just now. Tell the person the " +
				"proposal is still waiting on their decision in this conversation, and that " +
				"typing does not approve it.",
			Failed: true,
		}
	}
	if name == ShowBoardName {
		return toolcall.Outcome{
			Content: fmt.Sprintf("Tool %q could not show the board just now. Answer in your "+
				"reply instead.", name),
			Failed: true,
		}
	}

	return toolcall.Outcome{
		Content: fmt.Sprintf("Tool %q could not keep the document just now. "+
			"Put the text in your reply instead.", name),
		Failed: true,
	}
}
