package agentruntime

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/core/services/toolsimulation"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

type toolOutcome struct {
	content string
	failed  bool
	action  *serviceports.PendingAction
	// data is what a query tool returned, before it was encoded for the
	// model. The caller may turn it into something a person sees.
	data any
	// publishes marks a document the observer has to keep; its result is
	// written once the observer says what it kept.
	publishes bool
	summary   string
	// delegateReport is the bounded account of a delegate_task call, kept on
	// the call's result message.
	delegateReport *conversation.DelegateReport
	// taint is the outside content the call read.
	taint []agent.TaintMark
	found []string
}

func failedOutcome(format string, args ...any) toolOutcome {
	return toolOutcome{content: fmt.Sprintf(format, args...), failed: true}
}

// dispatchParams groups one tool call and everything deciding how it runs.
type dispatchParams struct {
	req            *serviceports.RunRequest
	call           serviceports.ToolCall
	completionText string
	proposedSoFar  []serviceports.PendingAction
	// idempotencyKey names this operation to the tool. It is the run's step
	// key where there is a ledger, and the provider's call id otherwise.
	idempotencyKey string
	// taint is the outside content the turn had read when it made the call;
	// nil for a turn opened before taint was kept.
	taint *agent.RunTaint
	// afterExternal holds every write to a proposal: the turn has read
	// content from outside the organization, which may have been written to
	// steer it.
	afterExternal bool
}

func (s *Service) dispatch(ctx context.Context, p dispatchParams) toolOutcome {
	req := p.req
	call := p.call
	completionText := p.completionText
	proposedSoFar := p.proposedSoFar
	// The loop refuses a tool the agent does not hold before it gets here, with
	// the nearest tools it does hold; this is the backstop for any other caller.
	if refusal, refused := s.extensionRefusal(ctx, req, call.Name); refused {
		return refusal
	}
	if !s.holdsFor(ctx, req, call.Name) {
		return failedOutcome("Tool %q is not available to this agent.", call.Name)
	}

	selfScoped := serviceports.IsSelfScoped(s.toolNamed(call.Name))
	if selfScoped && (req.Unattended || req.Actor == nil ||
		req.Actor.PrincipalType != serviceports.PrincipalTypeUser) {
		return failedOutcome(
			"Tool %q works only on the records of the person in the conversation, "+
				"and nobody is in this one.",
			call.Name,
		)
	}

	if tool, ok := s.queryTools.Get(call.Name); ok {
		if !selfScoped {
			if outcome, denied := s.authorize(
				ctx, req.Actor, call.Name, tool.Policy().Resource, permission.OpRead,
			); denied {
				return outcome
			}
		}

		call.Arguments = aliasedArguments(tool.ParamSchema(), call.Arguments)

		return s.runQueryTool(ctx, req, tool, call)
	}

	tool, ok := s.actionTools.Get(call.Name)
	if !ok {
		return failedOutcome("Tool %q does not exist.", call.Name)
	}

	policy := tool.Policy()
	if !selfScoped {
		if outcome, denied := s.authorize(
			ctx, req.Actor, call.Name, policy.Resource, policy.Operation,
		); denied {
			return outcome
		}
	}

	call.Arguments = declaredArguments(
		tool.ParamSchema(),
		aliasedArguments(tool.ParamSchema(), call.Arguments),
	)
	if selfScoped {
		// Whose records these are is the runtime's to say, not the model's:
		// anything the model sent under this key is overwritten. A copy, so
		// the model's own call as recorded in the thread is left as it sent it.
		owned := make(map[string]any, len(call.Arguments)+1)
		maps.Copy(owned, call.Arguments)
		owned[serviceports.SelfScopeOwnerParam] = req.Actor.UserID.String()
		call.Arguments = owned
	}
	p.call = call
	tierParams := serviceports.ToolExecuteParams{
		OrganizationID: req.Actor.OrganizationID,
		BusinessUnitID: req.Actor.BusinessUnitID,
		Actor:          req.Actor,
		IdempotencyKey: call.ID,
		RunID:          req.RunID,
		Params:         call.Arguments,
	}
	decision := s.decideCall(ctx, agenttoolpolicy.DecideInput{
		Policy:     policy,
		Params:     tierParams,
		Definition: req.Definition,
		Unattended: req.Unattended,
		Taint:      decisionTaint(p.taint, p.afterExternal),
	})
	tier, heldForExternal := afterExternalContent(decision.Tier, p.afterExternal)
	action := &serviceports.PendingAction{
		ToolName:  call.Name,
		Arguments: call.Arguments,
		Rationale: proposalRationale(rationaleInput{
			Narration: completionText,
			ToolName:  call.Name,
			Arguments: call.Arguments,
			Input:     req.Input,
		}),
		Tier:       tier,
		ToolCallID: call.ID,
		Egress:     decision.Egress,
		HeldBy:     decision.HeldBy,
		Tainted:    p.taint.Tainted() || p.afterExternal,
	}
	if heldForExternal && !slices.Contains(action.HeldBy, agenttoolpolicy.HeldByTainted) {
		action.HeldBy = append(slices.Clone(action.HeldBy), agenttoolpolicy.HeldByTainted)
	}

	if tier != agent.TierAutoExecute {
		// The same write proposed twice is one decision asked for twice. A
		// model that never learned its earlier proposal was still waiting
		// used to raise it again on every "yes", and the person got a stack
		// of identical cards.
		if pendingDuplicate(call, req.Proposals, proposedSoFar) {
			return toolOutcome{content: duplicateProposalText(call.Name)}
		}

		// A tool that can check its own arguments does so now, while the
		// model can still fix the call, rather than after a person has
		// approved a proposal that was never going to run.
		if validator, ok := tool.(serviceports.ToolValidator); ok {
			if vErr := validator.Validate(ctx, serviceports.ToolExecuteParams{
				OrganizationID: req.Actor.OrganizationID,
				BusinessUnitID: req.Actor.BusinessUnitID,
				Actor:          req.Actor,
				IdempotencyKey: p.idempotencyKey,
				RunID:          req.RunID,
				Params:         call.Arguments,
			}); vErr != nil {
				return failedOutcome(
					"Tool %q was not proposed, because it would fail as called: %s\n"+
						"Fix the call and try again.",
					call.Name, vErr.Error(),
				)
			}
		}

		action.Target = s.snapshotTarget(ctx, req, tool, call)

		content := fmt.Sprintf(
			"Recorded a proposal to run %q. It is awaiting a person's review at the %s tier and has not run.",
			call.Name,
			tier,
		)
		if heldForExternal {
			content += externalContentNote
		}
		if req.Definition.SimulationMode {
			content += " This agent is in simulation: an approval will preview the change, not make it."
		}

		return toolOutcome{content: content, action: action}
	}

	// An automatic write in simulation is previewed where it would have run,
	// and recorded as such, so a person can read what the agent would have
	// done at full reach before it is given any.
	if req.Definition.SimulationMode {
		return s.simulateAction(ctx, actionParams{dispatchParams: p, tool: tool, action: action})
	}

	if outcome, refused := s.withinBudget(ctx, req, call.Name, proposedSoFar); refused {
		return outcome
	}

	return s.executeAction(ctx, actionParams{dispatchParams: p, tool: tool, action: action})
}

// withinBudget refuses an automatic write past its tool's daily cap. The
// model is told which cap and to say so, rather than left to try again.
func (s *Service) withinBudget(
	ctx context.Context,
	req *serviceports.RunRequest,
	toolName string,
	soFar []serviceports.PendingAction,
) (toolOutcome, bool) {
	if s.budgets == nil {
		return toolOutcome{}, false
	}

	refusal, err := s.budgets.CheckTool(ctx, serviceports.CheckToolBudgetRequest{
		Definition: req.Definition,
		ToolName:   toolName,
		Unrecorded: executedCount(soFar, toolName),
	})
	if err != nil {
		s.logger.Error("agent tool budget check failed",
			zap.String("tool", toolName), zap.Error(err))

		return failedOutcome(
			"Tool %q was not run: its budget could not be checked. Try again later.",
			toolName,
		), true
	}
	if !refusal.Refused() {
		return toolOutcome{}, false
	}

	return failedOutcome("Tool %q was not run. %s Tell the person, and do not retry it.",
		toolName, refusal.Message(req.Definition.Name)), true
}

// executedCount is how many times this turn already ran a tool for real. Those
// writes are recorded as proposals only when the turn ends.
func executedCount(actions []serviceports.PendingAction, toolName string) int {
	n := 0
	for i := range actions {
		if actions[i].ToolName == toolName && actions[i].Executed && !actions[i].Simulated {
			n++
		}
	}

	return n
}

// actionParams groups one write and the tool that would make it.
type actionParams struct {
	dispatchParams

	tool   serviceports.AgentTool
	action *serviceports.PendingAction
}

// executeParams is what the tool is handed, built once so the simulated and
// the executed path cannot drift apart in what they pass.
func (a actionParams) executeParams() serviceports.ToolExecuteParams {
	params := serviceports.ToolExecuteParams{
		OrganizationID: a.req.Actor.OrganizationID,
		BusinessUnitID: a.req.Actor.BusinessUnitID,
		Actor:          a.req.Actor,
		IdempotencyKey: a.idempotencyKey,
		RunID:          a.req.RunID,
		Params:         a.call.Arguments,
	}
	if a.tool.Policy().CarriesTaint {
		params.Taint = a.taint.Clone()
	}

	return params
}

func (s *Service) simulateAction(ctx context.Context, a actionParams) toolOutcome {
	call := a.call
	action := a.action

	action.Simulated = true
	action.Simulation = toolsimulation.Simulate(ctx, a.tool, a.executeParams())

	return toolOutcome{
		content: fmt.Sprintf(
			"Simulated %q: nothing was changed, because this agent is in simulation. "+
				"What it would have done:\n%s\nCarry on as if it had run, and say in your "+
				"reply that the change was simulated.",
			call.Name, action.Simulation.Describe(),
		),
		action: action,
	}
}

func (s *Service) authorize(
	ctx context.Context,
	actor *serviceports.RequestActor,
	toolName string,
	resource permission.Resource,
	operation permission.Operation,
) (toolOutcome, bool) {
	if s.permissions == nil {
		return failedOutcome("Tool %q could not be authorized.", toolName), true
	}

	result, err := s.permissions.Check(ctx, &serviceports.PermissionCheckRequest{
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		UserID:         actor.UserID,
		APIKeyID:       actor.APIKeyID,
		BusinessUnitID: actor.BusinessUnitID,
		OrganizationID: actor.OrganizationID,
		Resource:       resource.String(),
		Operation:      operation,
	})
	if err != nil {
		s.logger.Error("agent tool authorization failed",
			zap.String("tool", toolName),
			zap.String("principal", actor.PrincipalID.String()),
			zap.Error(err),
		)

		return failedOutcome("Tool %q could not be authorized. Try again later.", toolName), true
	}

	if !result.Allowed {
		s.logger.Info("agent tool call denied",
			zap.String("tool", toolName),
			zap.String("principal", actor.PrincipalID.String()),
			zap.String("resource", resource.String()),
			zap.String("operation", string(operation)),
			zap.String("reason", result.Reason),
		)

		return failedOutcome(
			"Tool %q is not permitted: the person you are working for does not have %s access to %s.",
			toolName,
			operation,
			resource.String(),
		), true
	}

	return toolOutcome{}, false
}

func (s *Service) runQueryTool(
	ctx context.Context,
	req *serviceports.RunRequest,
	tool serviceports.AgentQueryTool,
	call serviceports.ToolCall,
) toolOutcome {
	data, err := tool.Query(ctx, &serviceports.QueryToolParams{
		OrganizationID:    req.Actor.OrganizationID,
		BusinessUnitID:    req.Actor.BusinessUnitID,
		Actor:             req.Actor,
		Timezone:          req.Context.Timezone,
		Params:            call.Arguments,
		AgentDefinitionID: req.Definition.ID,
	})
	if err != nil {
		return failedOutcome("Tool %q failed: %s", call.Name, err.Error())
	}

	encoded, document, err := encodeToolDocument(data, timeutils.NowUnix(), req.Context.Timezone)
	if err != nil {
		return failedOutcome("Tool %q returned data that could not be encoded.", call.Name)
	}

	return toolOutcome{
		content: FenceToolResult(call.Name, encoded),
		data:    data,
		summary: summarizeResult(call.Name, document),
		taint:   callTaint(tool.Policy(), call, data, timeutils.NowUnix()),
	}
}

func (s *Service) executeAction(ctx context.Context, a actionParams) toolOutcome {
	call := a.call
	action := a.action

	action.Executed = true

	result, err := serviceports.ExecuteTool(ctx, a.tool, a.executeParams())
	if err != nil {
		action.ExecutionError = err.Error()

		return toolOutcome{
			content: fmt.Sprintf("Tool %q failed: %s", call.Name, err.Error()),
			failed:  true,
			action:  action,
		}
	}
	action.ExecutionResult = result

	return toolOutcome{
		content: ranContent(call.Name, result),
		action:  action,
		taint:   callTaint(a.tool.Policy(), call, nil, timeutils.NowUnix()),
	}
}

// ranContent tells the model a write ran and, when the tool says what it
// made, which record that is and the id to refer to it by, so the next call
// does not have to go looking for it.
func ranContent(name string, result *agent.ToolExecutionResult) string {
	content := fmt.Sprintf("Tool %q ran successfully.", name)
	if describe := result.Describe(); describe != "" {
		content += " " + describe
	}
	if note := result.IDNote(); note != "" {
		content += " " + note
	}

	return content
}

// snapshotTarget pins the record a proposal would change, at the version it
// has now.
//
// A proposal is a promise to change something later, and "later" is the
// problem: the shipment a hold is proposed on today may be delivered by the
// time somebody approves. The version taken here is what the executor compares
// against before running. A tool with no single target, or a version that
// cannot be read, leaves the proposal unpinned rather than unmade — the person
// still gets to decide, they just decide without the staleness check.
func (s *Service) snapshotTarget(
	ctx context.Context,
	req *serviceports.RunRequest,
	tool serviceports.AgentTool,
	call serviceports.ToolCall,
) *serviceports.ProposalTarget {
	targeted, ok := tool.(serviceports.TargetedTool)
	if !ok || s.versions == nil {
		return nil
	}

	target, ok := targeted.Target(call.Arguments)
	if !ok {
		return nil
	}

	version, err := s.versions.Version(ctx, req.Actor.TenantInfo(), target)
	if err != nil {
		s.logger.Warn("could not pin the record a proposal would change",
			zap.String("tool", call.Name),
			zap.String("resource", string(target.Resource)),
			zap.String("id", target.ID.String()),
			zap.Error(err),
		)

		return nil
	}

	return &serviceports.ProposalTarget{Resource: target.Resource, ID: target.ID, Version: version}
}
