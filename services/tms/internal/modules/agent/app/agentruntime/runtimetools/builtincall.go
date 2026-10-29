package runtimetools

import (
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolcall"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
)

// Answer answers a call of one of the runtime's own tools that needs
// nothing but the turn: request_decision, withdraw_proposal, name_plan,
// publish_artifact and show_board. offers says whether the turn was offered a
// tool. handled is false for any other tool.
func Answer(
	t Turn,
	offers func(name string) bool,
	call *llmports.ToolCall,
) (outcome toolcall.Result, handled bool) {
	switch call.Name {
	case RequestDecisionName:
		outcome = toolcall.Refused(aitrace.OutcomeDenied,
			"the turn is not offered the approval box", "%s", UnofferedDecisionRefusal)
		if offers(RequestDecisionName) {
			outcome = requestDecision(t, call.Arguments)
		}

		return outcome, true
	case WithdrawProposalName:
		outcome = toolcall.Refused(aitrace.OutcomeDenied,
			"the turn has filed no proposal to withdraw", "%s", UnofferedWithdrawal)
		if t.Request().Delegation != nil {
			outcome = toolcall.Refused(aitrace.OutcomeDenied,
				"a delegated task cannot withdraw a proposal", "%s", delegatedWithdrawal)
		}
		if offers(WithdrawProposalName) && hasFiledProposal(t) {
			outcome = withdrawProposal(t, call.Arguments)
		}

		return outcome, true
	case NamePlanName:
		outcome = toolcall.Refused(aitrace.OutcomeDenied,
			"the turn is not offered plan naming", "%s", UnofferedNaming)
		if t.Request().Delegation != nil {
			outcome = toolcall.Refused(aitrace.OutcomeDenied,
				"a delegated task cannot name the plan", "%s", DelegatedNaming)
		}
		if offers(NamePlanName) {
			outcome = namePlan(t, call.Arguments)
		}

		return outcome, true
	case PublishArtifactName:
		outcome = PublishOutcome(call.Arguments)
		if !offers(PublishArtifactName) {
			outcome = toolcall.Refused(aitrace.OutcomeDenied,
				"the turn is not offered documents", "%s", UnpublishableRefusal)
		}

		return outcome, true
	case ShowBoardName:
		outcome = toolcall.Refused(aitrace.OutcomeDenied,
			"the turn is not offered boards", "%s", UnboardableRefusal)
		if offers(ShowBoardName) {
			outcome = showBoard(t, call.Arguments)
		}

		return outcome, true
	default:
		return toolcall.Result{}, false
	}
}
