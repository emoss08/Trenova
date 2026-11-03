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
	case "request_decision":
		outcome = toolcall.Refused(aitrace.OutcomeDenied,
			"the turn is not offered the approval box", "%s", UnofferedDecisionRefusal)
		if offers("request_decision") {
			outcome = requestDecision(t, call.Arguments)
		}

		return outcome, true
	case "withdraw_proposal":
		outcome = toolcall.Refused(aitrace.OutcomeDenied,
			"the turn has filed no proposal to withdraw", "%s", UnofferedWithdrawal)
		if t.Request().Delegation != nil {
			outcome = toolcall.Refused(aitrace.OutcomeDenied,
				"a delegated task cannot withdraw a proposal", "%s", delegatedWithdrawal)
		}
		if offers("withdraw_proposal") && hasFiledProposal(t) {
			outcome = withdrawProposal(t, call.Arguments)
		}

		return outcome, true
	case "name_plan":
		outcome = toolcall.Refused(aitrace.OutcomeDenied,
			"the turn is not offered plan naming", "%s", UnofferedNaming)
		if t.Request().Delegation != nil {
			outcome = toolcall.Refused(aitrace.OutcomeDenied,
				"a delegated task cannot name the plan", "%s", DelegatedNaming)
		}
		if offers("name_plan") {
			outcome = namePlan(t, call.Arguments)
		}

		return outcome, true
	case "publish_artifact":
		outcome = PublishOutcome(call.Arguments)
		if !offers("publish_artifact") {
			outcome = toolcall.Refused(aitrace.OutcomeDenied,
				"the turn is not offered documents", "%s", UnpublishableRefusal)
		}

		return outcome, true
	case "show_board":
		outcome = toolcall.Refused(aitrace.OutcomeDenied,
			"the turn is not offered boards", "%s", UnboardableRefusal)
		if offers("show_board") {
			outcome = showBoard(t, call.Arguments)
		}

		return outcome, true
	default:
		return toolcall.Result{}, false
	}
}
