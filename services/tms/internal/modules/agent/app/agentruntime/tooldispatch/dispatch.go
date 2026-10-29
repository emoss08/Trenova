package tooldispatch

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/argcheck"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/proposalledger"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolaccess"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolcall"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolsummary"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/turnhistory"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/turntaint"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolsimulation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentbudgetports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/graphqlreadports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/toolcoerce"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// DispatchParams groups one tool call and everything deciding how it runs.
type DispatchParams struct {
	req            *agentruntimeports.RunRequest
	call           llmports.ToolCall
	completionText string
	// argumentNote is what the runtime changed about the call's arguments
	// before the tool read them, told back to the model with the result.
	argumentNote  string
	proposedSoFar []agentruntimeports.PendingAction
	moneySpent    agenttooltrust.MoneyUse
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
	beyond        agent.AutonomyTier
	stepKey       string
	// scriptStepKey is set for a write a script prepared: the ledger step of
	// the script. The call is decided and previewed as any write is, and
	// handed back as an intent; it never runs before a person approves the
	// script's batch.
	scriptStepKey string
	gate          ScriptGate
	// caller is who made the call: the model, or a script's broker.
	caller string
}

func (s *Dispatcher) dispatch(ctx context.Context, p *DispatchParams) toolcall.Result {
	req := p.req
	call := p.call
	// The loop refuses a tool the agent does not hold before it gets here, with
	// the nearest tools it does hold; this is the backstop for any other caller.
	if refusal, refused := s.access.ExtensionRefusal(ctx, req, call.Name); refused {
		refusal.Verdict = aitrace.OutcomeDenied
		refusal.Reason = "its extension is turned off or not set up"

		return refusal
	}
	if !s.access.HoldsFor(ctx, req, call.Name) && !s.access.Reachable(req, p.beyond) {
		return toolcall.Refused(aitrace.OutcomeDenied, aianalytics.ReasonNotHeld,
			"Tool %q is not available to this agent.", call.Name)
	}

	selfScoped := agenttoolports.IsSelfScoped(s.access.ToolNamed(call.Name))
	if selfScoped && (req.Unattended || req.Actor == nil ||
		req.Actor.PrincipalType != permissionports.PrincipalTypeUser) {
		return toolcall.Refused(
			aitrace.OutcomeDenied,
			"it acts only on the records of a person in the conversation, and none was",
			"Tool %q works only on the records of the person in the conversation, "+
				"and nobody is in this one.",
			call.Name,
		)
	}

	if tool, ok := s.queryTools.Get(call.Name); ok {
		return s.dispatchQuery(ctx, p, tool, selfScoped)
	}

	tool, ok := s.actionTools.Get(call.Name)
	if !ok {
		return toolcall.Refused(aitrace.OutcomeInvalid, aianalytics.ReasonUnregistered,
			"Tool %q does not exist.", call.Name)
	}

	return s.dispatchAction(ctx, p, tool, selfScoped)
}

func (s *Dispatcher) dispatchQuery(
	ctx context.Context,
	p *DispatchParams,
	tool agenttoolports.AgentQueryTool,
	selfScoped bool,
) toolcall.Result {
	req := p.req
	call := p.call
	if !selfScoped {
		if outcome, denied := s.authorize(
			ctx, req.Actor, call.Name, tool.Policy().Resource, permission.OpRead,
		); denied {
			return outcome
		}
	}

	runsScript := tool.Policy().Script == agenttoolports.ScriptRunner
	if runsScript {
		if refusal, refused := continuationRefusal(&call, p.gate); refused {
			return refusal
		}
		call.Arguments = inheritScriptName(call.Arguments, p.gate.Continues)
	}
	contract, err := argcheck.ContractCall(
		s.arguments, call.Name, tool.ParamSchema(), call.Arguments, organizationToday(req),
	)
	if err != nil {
		return argcheck.Refusal(call.Name, tool.ParamSchema(), err)
	}
	call.Arguments = contract.Args

	if runsScript {
		return s.runScriptTool(ctx, p, tool, &call)
	}

	outcome := s.runQueryToolWith(ctx, req, tool, &call, &queryExtras{
		fitPage: p.caller != aitrace.CallerScript,
	})
	outcome.RanWith = call.Arguments
	if note := contract.Note(); note != "" && !outcome.Failed {
		outcome.Content += "\n\n[" + note + "]"
	}

	return outcome
}

type callHold struct {
	tier     agent.AutonomyTier
	egress   agent.EgressClass
	external bool
	hours    bool
}

func (s *Dispatcher) dispatchAction(
	ctx context.Context,
	p *DispatchParams,
	tool agenttoolports.AgentTool,
	selfScoped bool,
) toolcall.Result {
	req := p.req
	policy := s.Ruled(ctx, req, new(tool.Policy()))
	if outcome, refused := s.prepareActionCall(ctx, p, tool, &policy, selfScoped); refused {
		return outcome
	}
	decision := s.DecideCall(ctx, agenttoolpolicy.DecideInput{
		Policy:         policy,
		Params:         RunToolParams(req, p.call.ID, p.call.Arguments),
		Definition:     req.Definition,
		HolderCeiling:  p.beyond,
		Unattended:     req.Unattended,
		Taint:          toolaccess.DecisionTaint(p.taint, p.afterExternal),
		MoneyAllowance: s.MoneyAllowance(req, p.moneySpent),
		FromScript:     p.scriptStepKey != "",
	})
	hold := callHold{tier: decision.Tier, egress: decision.Egress}
	if p.scriptStepKey != "" || !agenttoolpolicy.Escalates(req.Definition, &policy) {
		hold.tier, hold.external = toolaccess.AfterExternalContent(hold.tier, p.afterExternal)
		hold.tier, hold.hours = s.heldForBusinessHours(ctx, req, hold.tier)
	}
	if hold.tier != agent.TierAutoExecute {
		if outcome, refused := pinProposalSelection(ctx, p, tool); refused {
			return outcome
		}
	}
	if p.scriptStepKey != "" && hold.tier == agent.TierAutoExecute {
		return toolcall.Refused(
			aitrace.OutcomeInvalid,
			"a prepared change would run on its own",
			"Tool %q was not prepared: a change from a script never runs on its own.",
			p.call.Name,
		)
	}
	action := pendingAction(ctx, p, &decision, &hold)

	if outcome, refused := validateCall(ctx, p, tool, hold.tier); refused {
		return outcome
	}

	if hold.tier != agent.TierAutoExecute {
		return s.fileProposal(ctx, p, tool, action, &hold)
	}

	// An automatic write in simulation is previewed where it would have run,
	// and recorded as such, so a person can read what the agent would have
	// done at full reach before it is given any.
	if req.Definition.SimulationMode {
		return s.simulateAction(ctx, &actionParams{DispatchParams: *p, tool: tool, action: action})
	}

	if outcome, refused := s.withinBudget(ctx, req, p.call.Name, p.proposedSoFar); refused {
		return outcome
	}

	return s.executeAction(ctx, &actionParams{DispatchParams: *p, tool: tool, action: action})
}

func (s *Dispatcher) prepareActionCall(
	ctx context.Context,
	p *DispatchParams,
	tool agenttoolports.AgentTool,
	policy *agenttoolports.ToolPolicy,
	selfScoped bool,
) (toolcall.Result, bool) {
	req := p.req
	call := p.call
	if !selfScoped {
		if outcome, denied := s.authorize(
			ctx, req.Actor, call.Name, policy.Resource, policy.Operation,
		); denied {
			return outcome, true
		}
	}

	contract, err := argcheck.ContractCall(
		s.arguments, call.Name, tool.ParamSchema(), call.Arguments, organizationToday(req),
	)
	if err != nil {
		return argcheck.Refusal(call.Name, tool.ParamSchema(), err), true
	}
	call.Arguments = contract.Args
	p.argumentNote = contract.Note()
	if selfScoped {
		// Whose records these are is the runtime's to say, not the model's:
		// anything the model sent under this key is overwritten. A copy, so
		// the model's own call as recorded in the thread is left as it sent it.
		owned := make(map[string]any, len(call.Arguments)+1)
		maps.Copy(owned, call.Arguments)
		owned[agenttoolports.SelfScopeOwnerParam] = req.Actor.UserID.String()
		call.Arguments = owned
	}
	if resolver, resolves := tool.(agenttoolports.ToolSelectionResolver); resolves {
		resolved, rErr := resolver.ResolveSelection(
			ctx, RunToolParams(req, p.idempotencyKey, call.Arguments),
		)
		if rErr != nil {
			return unresolvedSelection(call.Name, "proposed or run", rErr), true
		}
		call.Arguments = resolved
	}
	p.call = call

	return oversizedChange(req, call.Name, tool.ParamSchema(), call.Arguments)
}

func pinProposalSelection(
	ctx context.Context,
	p *DispatchParams,
	tool agenttoolports.AgentTool,
) (toolcall.Result, bool) {
	pinner, pins := tool.(agenttoolports.ToolProposalSelectionResolver)
	if !pins {
		return toolcall.Result{}, false
	}
	pinned, err := pinner.ResolveProposalSelection(
		ctx, RunToolParams(p.req, p.idempotencyKey, p.call.Arguments),
	)
	if err != nil {
		return unresolvedSelection(p.call.Name, "proposed", err), true
	}
	p.call.Arguments = pinned

	return toolcall.Result{}, false
}

func pendingAction(
	ctx context.Context,
	p *DispatchParams,
	decision *agenttoolpolicy.Decision,
	hold *callHold,
) *agentruntimeports.PendingAction {
	call := p.call
	action := &agentruntimeports.PendingAction{
		ToolName:  call.Name,
		Arguments: call.Arguments,
		Rationale: turnhistory.ProposalRationale(turnhistory.RationaleInput{
			Narration: p.completionText,
			ToolName:  call.Name,
			Arguments: call.Arguments,
			Input:     p.req.Input,
		}),
		Tier:          hold.tier,
		ToolCallID:    call.ID,
		Egress:        decision.Egress,
		HeldBy:        decision.HeldBy,
		MoneyMinor:    decision.MoneyMinor,
		MoneyCurrency: decision.MoneyCurrency,
		Tainted:       p.taint.Tainted() || p.afterExternal,
		ScriptStepKey: p.scriptStepKey,
	}
	if hold.external && !slices.Contains(action.HeldBy, agenttoolpolicy.HeldByTainted) {
		action.HeldBy = append(slices.Clone(action.HeldBy), agenttoolpolicy.HeldByTainted)
	}
	if hold.hours {
		action.HeldBy = append(slices.Clone(action.HeldBy), agenttoolpolicy.HeldByBusinessHours)
	}
	if hold.tier != agent.TierAutoExecute {
		action.MoneyMinor = nil
		action.MoneyCurrency = ""
	}
	source := decision.Source
	if hold.external || hold.hours {
		source = agenttooltrust.TierSourcePolicyDefault
	}
	stampAction(ctx, action, source, p.stepKey)

	return action
}

// validateCall lets a tool that can check its own arguments do so now, at
// every tier, while the model can still fix the call: before a person is asked
// to approve a proposal that was never going to run, before a simulation
// records what it would have done, and before an automatic write runs.
func validateCall(
	ctx context.Context,
	p *DispatchParams,
	tool agenttoolports.AgentTool,
	tier agent.AutonomyTier,
) (toolcall.Result, bool) {
	validator, validates := tool.(agenttoolports.ToolValidator)
	if !validates {
		return toolcall.Result{}, false
	}
	vErr := validator.Validate(ctx, RunToolParams(p.req, p.idempotencyKey, p.call.Arguments))
	if vErr == nil {
		return toolcall.Result{}, false
	}
	verb := "run"
	if tier != agent.TierAutoExecute {
		verb = "proposed"
	}

	return toolcall.Refused(
		aitrace.OutcomeInvalid,
		vErr.Error(),
		"Tool %q was not %s, because it would fail as called: %s\n"+
			"Fix the call and try again.",
		p.call.Name, verb, vErr.Error(),
	), true
}

func (s *Dispatcher) fileProposal(
	ctx context.Context,
	p *DispatchParams,
	tool agenttoolports.AgentTool,
	action *agentruntimeports.PendingAction,
	hold *callHold,
) toolcall.Result {
	req := p.req
	call := p.call
	// The same write proposed twice is one decision asked for twice. A
	// model that never learned its earlier proposal was still waiting
	// used to raise it again on every "yes", and the person got a stack
	// of identical cards.
	if proposalledger.PendingDuplicate(&call, req.Proposals, p.proposedSoFar) {
		return toolcall.Result{
			Content: proposalledger.DuplicateText(call.Name),
			Verdict: aitrace.OutcomeDuplicate,
			Reason:  "the same change is already waiting for a decision",
		}
	}

	// The write is previewed as it is filed. One that its own rules would
	// refuse is not filed: a person could only reject it, so the model is
	// told why and asked for what is missing instead. A write on a record
	// an earlier step of this turn changes is the exception, since that
	// step may be what makes it valid.
	dependsOnStep := dependsOnEarlierStep(tool, &call, p.proposedSoFar)
	target, baseline := s.fileBaseline(ctx, &baselineCall{
		req:         req,
		tool:        tool,
		call:        call,
		proposalID:  action.ProposalID,
		persist:     req.AttributedPurpose() != aiusageports.AIUsagePurposeEvaluation,
		fileRefused: dependsOnStep,
	})
	if baseline != nil && !dependsOnStep {
		if refusal := baseline.Preview.Refusal(); refusal != nil {
			return refusedBeforeFiling(call.Name, refusal)
		}
	}
	action.Target = target

	content := s.filedText(ctx, &filedTextParams{
		req:      req,
		toolName: call.Name,
		action:   action,
		tier:     hold.tier,
		prepared: p.scriptStepKey != "",
	}) +
		heldReason(
			action.HeldBy,
			hold.egress,
		) + toolsummary.FilingEcho(baseline, call.Arguments)
	if note := p.argumentNote; note != "" {
		content += "\n" + note
	}
	if hold.external {
		content += toolaccess.ExternalContentNote
	}
	if hold.hours {
		content += businessHoursNote
	}
	if req.Definition.SimulationMode {
		content += " This agent is in simulation: an approval will preview the change, not make it."
	}

	return toolcall.Result{Content: content, Action: action, Verdict: aitrace.OutcomeProposed}
}

type filedTextParams struct {
	req      *agentruntimeports.RunRequest
	toolName string
	action   *agentruntimeports.PendingAction
	tier     agent.AutonomyTier
	prepared bool
}

func (s *Dispatcher) filedText(ctx context.Context, p *filedTextParams) string {
	if p.prepared {
		return fmt.Sprintf(
			"Prepared a change to run %q. Nothing has run: when the script ends it is filed "+
				"with the script's other changes as one batch, which a person approves or rejects "+
				"as a whole.",
			p.toolName,
		)
	}

	return proposedText(p.toolName, p.action.ProposalID, p.tier, s.proposalHold(ctx, p.req))
}

func unresolvedSelection(toolName, verb string, err error) toolcall.Result {
	return toolcall.Refused(
		aitrace.OutcomeInvalid,
		err.Error(),
		"Tool %q was not %s: its selection could not be resolved to records: %s\n"+
			"Fix the call and try again.",
		toolName, verb, err.Error(),
	)
}

func RunToolParams(
	req *agentruntimeports.RunRequest,
	idempotencyKey string,
	arguments map[string]any,
) agenttoolports.ToolExecuteParams {
	return agenttoolports.ToolExecuteParams{
		OrganizationID: req.Actor.OrganizationID,
		BusinessUnitID: req.Actor.BusinessUnitID,
		Actor:          req.Actor,
		IdempotencyKey: idempotencyKey,
		RunID:          req.RunID,
		Params:         arguments,
		Timezone:       req.Context.Timezone,
		DefinitionID:   definitionIDOf(req),
		ThreadID:       req.ThreadID,
		Delegated:      req.Delegation != nil,
	}
}

func definitionIDOf(req *agentruntimeports.RunRequest) pulid.ID {
	if req.Definition == nil {
		return pulid.Nil
	}

	return req.Definition.ID
}

// withinBudget refuses an automatic write past its tool's daily cap. The
// model is told which cap and to say so, rather than left to try again.
func (s *Dispatcher) withinBudget(
	ctx context.Context,
	req *agentruntimeports.RunRequest,
	toolName string,
	soFar []agentruntimeports.PendingAction,
) (toolcall.Result, bool) {
	if s.budgets == nil {
		return toolcall.Result{}, false
	}

	refusal, err := s.budgets.CheckTool(ctx, agentbudgetports.CheckToolBudgetRequest{
		Definition: req.Definition,
		ToolName:   toolName,
		Unrecorded: executedCount(soFar, toolName),
	})
	if err != nil {
		s.logger.Error("agent tool budget check failed",
			zap.String("tool", toolName), zap.Error(err))

		return toolcall.Refused(
			aitrace.OutcomeFailed,
			"its daily cap could not be checked",
			"Tool %q was not run: its budget could not be checked. Try again later.",
			toolName,
		), true
	}
	if !refusal.Refused() {
		return toolcall.Result{}, false
	}

	message := refusal.Message(req.Definition.Name)

	return toolcall.Refused(aitrace.OutcomeOverBudget, message,
		"Tool %q was not run. %s Tell the person, and do not retry it.",
		toolName, message), true
}

func (s *Dispatcher) MoneyAllowance(
	req *agentruntimeports.RunRequest,
	spent agenttooltrust.MoneyUse,
) agenttoolpolicy.MoneyAllowanceReader {
	if s.budgets == nil {
		return nil
	}

	return func(ctx context.Context) (agenttooltrust.MoneyAllowance, error) {
		allowance, err := s.budgets.MoneyAllowance(ctx, agentbudgetports.MoneyAllowanceRequest{
			Definition: req.Definition,
			RunID:      req.RunID,
			Unrecorded: spent,
		})
		if err != nil {
			s.logger.Warn("agent money limits could not be read; the write waits for a person",
				zap.Error(err))
		}

		return allowance, err
	}
}

func UnrecordedMoney(actions []agentruntimeports.PendingAction) agenttooltrust.MoneyUse {
	use := agenttooltrust.MoneyUse{}
	for i := range actions {
		action := &actions[i]
		if movedMoney(action) {
			use.Calls++
			use.AmountMinor += intutils.Abs(*action.MoneyMinor)
		}
	}

	return use
}

func movedMoney(action *agentruntimeports.PendingAction) bool {
	return action.MoneyMinor != nil && !action.Withdrawn && action.ExecutionError == "" &&
		(action.Executed || action.Simulated)
}

// executedCount is how many times this turn already ran a tool for real. Those
// writes are recorded as proposals only when the turn ends.
func executedCount(actions []agentruntimeports.PendingAction, toolName string) int {
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
	DispatchParams

	tool   agenttoolports.AgentTool
	action *agentruntimeports.PendingAction
}

// executeParams is what the tool is handed, built once so the simulated and
// the executed path cannot drift apart in what they pass.
func (a *actionParams) executeParams() agenttoolports.ToolExecuteParams {
	params := RunToolParams(a.req, a.idempotencyKey, a.call.Arguments)
	if a.tool.Policy().CarriesTaint {
		params.Taint = a.taint.Clone()
	}

	return params
}

func (s *Dispatcher) simulateAction(ctx context.Context, a *actionParams) toolcall.Result {
	call := a.call
	action := a.action

	action.Simulated = true
	target, baseline := s.fileBaseline(ctx, &baselineCall{
		req:        a.req,
		tool:       a.tool,
		call:       call,
		proposalID: action.ProposalID,
	})
	action.Target = target
	writeCtx, write := s.startWrite(ctx, a, true)
	if s.previews != nil {
		action.Simulation = toolsimulation.FromBaseline(call.Name, call.Arguments, baseline)
	} else {
		params := a.executeParams()
		action.Simulation = toolsimulation.Simulate(writeCtx, a.tool, &params)
	}
	write.End()

	return toolcall.Result{
		Verdict: aitrace.OutcomeSimulated,
		Content: fmt.Sprintf(
			"Simulated %q: nothing was changed, because this agent is in simulation. "+
				"What it would have done:\n%s\nCarry on as if it had run, and say in your "+
				"reply that the change was simulated.",
			call.Name, action.Simulation.Describe(),
		),
		Action: action,
	}
}

func (s *Dispatcher) authorize(
	ctx context.Context,
	actor *permissionports.RequestActor,
	toolName string,
	resource permission.Resource,
	operation permission.Operation,
) (toolcall.Result, bool) {
	if s.permissions == nil {
		return toolcall.Refused(
			aitrace.OutcomeFailed,
			"no permission engine could authorize it",
			"Tool %q could not be authorized.",
			toolName,
		), true
	}

	result, err := s.permissions.Check(ctx, &permissionports.PermissionCheckRequest{
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

		return toolcall.Refused(aitrace.OutcomeFailed, "its permission could not be checked",
			"Tool %q could not be authorized. Try again later.", toolName), true
	}

	if !result.Allowed {
		s.logger.Info("agent tool call denied",
			zap.String("tool", toolName),
			zap.String("principal", actor.PrincipalID.String()),
			zap.String("resource", resource.String()),
			zap.String("operation", string(operation)),
			zap.String("reason", result.Reason),
		)

		return toolcall.Refused(
			aitrace.OutcomeDenied,
			fmt.Sprintf("lacks %s access to %s", operation, resource.String()),
			"Tool %q is not permitted: %s. %s",
			toolName,
			deniedAccess(actor, operation, resource),
			deniedAdvice(actor, operation, resource),
		), true
	}

	return toolcall.Result{}, false
}

func deniedAccess(
	actor *permissionports.RequestActor,
	operation permission.Operation,
	resource permission.Resource,
) string {
	if actor.IsAgent() {
		return fmt.Sprintf(
			"this agent's unattended access does not include %s on %s",
			operation, resource.String(),
		)
	}

	return fmt.Sprintf(
		"the person you are working for does not have %s access to %s",
		operation, resource.String(),
	)
}

// organizationToday is the reading of "today" for a call's day parameters:
// the calendar day it is now in the organization's timezone, which is the
// day the runtime context told the model.
func organizationToday(req *agentruntimeports.RunRequest) toolcoerce.Option {
	return toolcoerce.WithToday(
		time.Unix(timeutils.NowUnix(), 0).In(timeutils.LoadLocation(req.Context.Timezone)),
	)
}

// deniedAdvice tells the model what to do with a denial. Before it, a model
// refused for a permission tried the same call another way, or told the
// person the system could not do it; the person needed to hear whose
// permission it was and who grants it.
func deniedAdvice(
	actor *permissionports.RequestActor,
	operation permission.Operation,
	resource permission.Resource,
) string {
	if actor.IsAgent() {
		return "Do not try it another way. Finish what the run can do without it and say in " +
			"your report that this step needs a person, or someone to widen what this " +
			"agent may do unattended in AI Control."
	}

	return fmt.Sprintf(
		"Do not try it another way or propose it instead. Tell the person plainly that "+
			"their account does not have %s access to %s, that an administrator grants it "+
			"under their role, and what you would do once it is granted. Do everything else "+
			"they asked that does not need it.",
		operation, humanResource(resource),
	)
}

func humanResource(resource permission.Resource) string {
	return strings.ReplaceAll(resource.String(), "_", " ")
}

// queryExtras is what only the tool that runs a script is handed: the broker
// its reads go through and the step it is claimed under; and, for a call the
// model reads, that a list page too long for it is cut to the rows that fit.
type queryExtras struct {
	broker    codemode.Broker
	stepKey   string
	continues map[string]any
	fitPage   bool
}

func (s *Dispatcher) runQueryToolWith(
	ctx context.Context,
	req *agentruntimeports.RunRequest,
	tool agenttoolports.AgentQueryTool,
	call *llmports.ToolCall,
	extras *queryExtras,
) toolcall.Result {
	data, err := tool.Query(ctx, &agenttoolports.QueryToolParams{
		OrganizationID:    req.Actor.OrganizationID,
		BusinessUnitID:    req.Actor.BusinessUnitID,
		Actor:             req.Actor,
		Timezone:          req.Context.Timezone,
		Params:            call.Arguments,
		AgentDefinitionID: req.Definition.ID,
		DataAccessCeiling: req.Definition.DataAccessSensitivity(),
		ThreadID:          req.ThreadID,
		RunID:             req.RunID,
		Broker:            extras.broker,
		StepKey:           extras.stepKey,
		CallID:            call.ID,
		Continues:         extras.continues,
		IncludeTotal:      extras.fitPage && firstPage(call.Arguments),
	})
	if refused, ok := errors.AsType[*graphqlreadports.ReadRefusedError](err); ok {
		return toolcall.Refused(
			aitrace.OutcomeDenied,
			fmt.Sprintf("lacks %s access to %s", permission.OpRead, refused.Resource.String()),
			"Tool %q is not permitted: %s. %s",
			call.Name,
			deniedAccess(req.Actor, permission.OpRead, refused.Resource),
			deniedAdvice(req.Actor, permission.OpRead, refused.Resource),
		)
	}
	if err != nil {
		return toolcall.Refused(aitrace.OutcomeFailed, err.Error(),
			"Tool %q failed: %s", call.Name, err.Error())
	}

	now := timeutils.NowUnix()
	var fields []string
	if extras.fitPage {
		fields = toolresult.ListFields(call.Arguments)
	}
	encoded, document, err := toolresult.EncodeModelDocument(
		data,
		now,
		req.Context.Timezone,
		fields,
	)
	if err != nil {
		return toolcall.Refused(aitrace.OutcomeFailed, "its result could not be encoded",
			"Tool %q returned data that could not be encoded.", call.Name)
	}
	content := ""
	if extras.fitPage {
		if fitted, cut := toolresult.FitPage(
			data,
			encoded,
			now,
			req.Context.Timezone,
			fields,
		); cut {
			data, encoded, document = fitted.Data, fitted.Encoded, fitted.Document
			content = "\n\n" + fitted.Note()
		}
	}

	policy := s.Ruled(ctx, req, new(tool.Policy()))
	outcome := toolcall.Result{
		Content: toolresult.Fence(call.Name, encoded) + content,
		Data:    data,
		Summary: toolsummary.Result(call.Name, document),
		Taint:   turntaint.Call(&policy, call, data, now),
		Verdict: aitrace.OutcomeRan,
	}
	if recall, ok := data.(agentmemoryports.MemoryRecall); ok {
		outcome.Memories = recall.RecalledMemoryIDs()
	}

	return outcome
}

// firstPage reports whether a list call reads its first page, which is where
// a model learns how many records the whole list holds: counted exactly up to
// pagination.ExactTotalUpTo and estimated past it, so "how many inactive
// drivers" is answered from the first page rather than by paging to the end.
func firstPage(arguments map[string]any) bool {
	after, _ := arguments[listAfterParam].(string)

	return after == ""
}

const listAfterParam = "after"

func (s *Dispatcher) executeAction(ctx context.Context, a *actionParams) toolcall.Result {
	call := a.call
	action := a.action

	action.Executed = true
	action.Target = s.snapshotTarget(ctx, a.req, a.tool, &call)

	writeCtx, write := s.startWrite(ctx, a, false)
	action.ExecutedAt = timeutils.NowUnix()
	var (
		result    *agentrecord.ToolExecutionResult
		saved     *agentmemoryports.SavedMemory
		refreshed *agentmemory.Memory
		err       error
	)
	if recorder, records := a.tool.(agentmemoryports.MemoryRecordingTool); records {
		var memory *agentmemory.Memory
		memory, err = recorder.Record(writeCtx, a.executeParams())
		if err == nil && memory != nil {
			saved, refreshed = savedOrRefreshed(memory, call.ID)
		}
	} else {
		result, err = agenttoolports.ExecuteTool(writeCtx, a.tool, a.executeParams())
	}
	if err != nil {
		aitrace.MarkFailed(write, aitrace.OutcomeFailed)
		write.End()
		action.ExecutionError = err.Error()

		return toolcall.Result{
			Content: fmt.Sprintf("Tool %q failed: %s", call.Name, err.Error()),
			Failed:  true,
			Action:  action,
			Verdict: aitrace.OutcomeFailed,
			Reason:  err.Error(),
		}
	}
	action.ExecutionResult = result
	action.ExecutedVersion = s.versionAfter(writeCtx, a.req, action.Target)
	if action.ExecutedVersion != nil {
		write.SetAttributes(aitrace.AIVersionAfter.Int64(*action.ExecutedVersion))
	}
	write.End()

	policy := s.Ruled(ctx, a.req, new(a.tool.Policy()))
	outcome := toolcall.Result{
		Content: ranContent(call.Name, result),
		Action:  action,
		Taint:   turntaint.Call(&policy, &call, nil, timeutils.NowUnix()),
		Verdict: aitrace.OutcomeRan,
		Saved:   saved,
	}
	switch {
	case refreshed != nil:
		outcome.Content = refreshedContent(call.Name, refreshed)
		outcome.Memories = []pulid.ID{refreshed.ID}
	case saved != nil:
		outcome.Content = savedMemoryContent(call.Name, saved)
	}
	if note := a.argumentNote; note != "" {
		outcome.Content += " " + note
	}

	return outcome
}

// savedMemoryContent tells the model whether what it asked to remember is
// kept or only offered. Both used to read "ran successfully", and a model
// told the person "I'll remember that" about a memory they had asked to be
// asked about first, which then sat on a card they had not decided.
func savedMemoryContent(name string, saved *agentmemoryports.SavedMemory) string {
	if saved.Pending {
		return fmt.Sprintf(
			"Tool %q ran: the memory is offered to the person on a card under your reply, "+
				"because they asked to be asked before anything is kept for them. Nothing is "+
				"kept until they choose Keep there. Tell them in a few words that it is there "+
				"to keep or not; do not say it is remembered.",
			name,
		)
	}

	return fmt.Sprintf(
		"Tool %q ran: the memory is kept and will be read in later conversations. Tell the "+
			"person in a few words that you will keep it in mind; they can undo it on the card "+
			"under your reply.",
		name,
	)
}

// heldReason says in one clause why a call became a proposal rather than
// running, so the model can tell the person plainly instead of guessing at
// the tier. The keys are the policy decision's HeldBy.
func heldReason(heldBy []string, egress agent.EgressClass) string {
	var reasons []string
	seen := make(map[string]struct{}, len(heldBy))
	for _, key := range heldBy {
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		switch key {
		case agenttoolpolicy.HeldByToolTier:
			reasons = append(reasons, "this tool is set to propose rather than run")
		case agenttoolpolicy.HeldByAgentCeiling:
			reasons = append(
				reasons,
				"this agent's autonomy ceiling does not let it run writes on its own",
			)
		case agenttoolpolicy.HeldByToolMax:
			reasons = append(reasons, "this change is always a person's decision")
		case agenttoolpolicy.HeldByEgressClass:
			reasons = append(reasons, egressReason(egress))
		case agenttoolpolicy.HeldByCondition:
			reasons = append(reasons, "the change is past the size or value this tool runs unasked")
		case agenttoolpolicy.HeldByTainted:
			reasons = append(reasons, "this turn read text written outside the organization")
		case agenttoolpolicy.HeldByBusinessHours:
			reasons = append(reasons, "it is outside this agent's business hours")
		case agenttoolpolicy.HeldByScriptIntent:
			reasons = append(
				reasons,
				"a script prepared it, and a person approves the script's batch",
			)
		default:
			if reason, ok := moneyHeldReasons[key]; ok {
				reasons = append(reasons, reason)
			}
		}
	}
	if len(reasons) == 0 {
		return ""
	}

	return " It waits because " + strings.Join(reasons, ", and because ") + "."
}

var moneyHeldReasons = map[string]string{
	agenttooltrust.HeldByMoneyUnbounded: "this tool moves money and cannot show that a call is " +
		"within the limits it may run in on its own",
	agenttooltrust.HeldByMoneyUnproven: "it could not be shown that this change is within the " +
		"limits it may run in on its own",
	agenttooltrust.HeldByMoneyExactMatch: "the money does not match its records exactly by " +
		"amount and reference",
	agenttooltrust.HeldByMoneyRecordState: "the record is past the point where an agent may " +
		"change its money on its own",
	agenttooltrust.HeldByMoneyAgentLine:    "a person added that line, so a person removes it",
	agenttooltrust.HeldByMoneyLimitsUnread: "the organization's money limits for agents could not be read",
	agenttooltrust.HeldByMoneyCurrency: "the amount is not in the organization's functional currency, " +
		"which its money limits for agents are set in",
	agenttooltrust.HeldByMoneyCallCeiling: "the amount is more than one automatic change may " +
		"move",
	agenttooltrust.HeldByMoneyRunCalls: "this run has already made as many automatic money " +
		"changes as it may",
	agenttooltrust.HeldByMoneyRunAmount: "it would take this run past the money it may move on " +
		"its own",
	agenttooltrust.HeldByMoneyDayCalls: "the organization's agents have made as many automatic " +
		"money changes today as they may",
	agenttooltrust.HeldByMoneyDayAmount: "it would take the organization's agents past the money " +
		"they may move on their own today",
}

func egressReason(egress agent.EgressClass) string {
	switch egress {
	case agent.EgressCustomerVisible:
		return "the customer would see this change"
	case agent.EgressDriverVisible:
		return "a driver would see this change"
	case agent.EgressExternalRecipient:
		return "it would reach someone outside the organization"
	case agent.EgressMoney:
		return "it moves money"
	case agent.EgressNone, agent.EgressPersonal, agent.EgressInternal:
		return "of what the change would reach"
	default:
		return "of what the change would reach"
	}
}

// savedOrRefreshed sorts what a remember returned. A memory it kept or
// offered is shown to the person with an Undo. One it only refreshed is not:
// nothing new was kept, and undoing would retire the memory they already had,
// so the turn names it as a memory it used instead.
func savedOrRefreshed(
	memory *agentmemory.Memory,
	callID string,
) (*agentmemoryports.SavedMemory, *agentmemory.Memory) {
	if memory.Refreshed {
		return nil, memory
	}

	return &agentmemoryports.SavedMemory{
		ID:      memory.ID,
		CallID:  callID,
		Pending: memory.Status == agentmemory.MemoryStatusSuggested,
	}, nil
}

// refreshedContent tells the model that what it asked to save was already
// remembered, so it says so to the person rather than announcing a new memory
// and does not try again with other words.
func refreshedContent(name string, memory *agentmemory.Memory) string {
	return fmt.Sprintf(
		"Tool %q ran successfully. This was already remembered as memory %s, so that "+
			"memory was refreshed instead of saving a duplicate; nothing new was recorded.",
		name, memory.ID,
	)
}

func (s *Dispatcher) startWrite(
	ctx context.Context,
	a *actionParams,
	simulated bool,
) (context.Context, trace.Span) {
	spec := &aitrace.WriteSpec{
		EntityType: string(a.tool.Policy().Resource),
		ProposalID: a.action.ProposalID,
		Simulated:  simulated,
	}
	if target := a.action.Target; target != nil {
		spec.EntityType = string(target.Resource)
		spec.EntityID = target.ID.String()
		before := target.Version
		spec.VersionBefore = &before
	}

	return aitrace.StartWrite(ctx, spec)
}

func (s *Dispatcher) versionAfter(
	ctx context.Context,
	req *agentruntimeports.RunRequest,
	target *agentruntimeports.ProposalTarget,
) *int64 {
	if target == nil || s.versions == nil {
		return nil
	}

	version, err := s.versions.Version(ctx, req.Actor.TenantInfo(), agenttoolports.ToolTarget{
		Resource: target.Resource,
		ID:       target.ID,
	})
	if err != nil {
		s.logger.Warn("could not read the version a write left its record at",
			zap.String("resource", string(target.Resource)),
			zap.String("id", target.ID.String()),
			zap.Error(err),
		)

		return nil
	}

	return &version
}

// ranContent tells the model a write ran and, when the tool says what it
// made, which record that is and the id to refer to it by, so the next call
// does not have to go looking for it.
func ranContent(name string, result *agentrecord.ToolExecutionResult) string {
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
func (s *Dispatcher) snapshotTarget(
	ctx context.Context,
	req *agentruntimeports.RunRequest,
	tool agenttoolports.AgentTool,
	call *llmports.ToolCall,
) *agentruntimeports.ProposalTarget {
	targeted, ok := tool.(agenttoolports.TargetedTool)
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

	return &agentruntimeports.ProposalTarget{
		Resource: target.Resource,
		ID:       target.ID,
		Version:  version,
	}
}

// Ruled is a tool's policy as the organization holds it: never looser than the
// rule declared beside the tool, and lower where the organization says so.
func (s *Dispatcher) Ruled(
	ctx context.Context,
	req *agentruntimeports.RunRequest,
	policy *agenttoolports.ToolPolicy,
) agenttoolports.ToolPolicy {
	if s.rules == nil || req == nil || req.Actor == nil {
		return *policy
	}

	rules, err := s.rules.For(ctx, req.Actor.TenantInfo())
	if err != nil {
		s.logger.Error(
			"the organization's tool rules could not be read; holding the tool to a proposal",
			zap.String("tool", policy.Name),
			zap.Error(err),
		)
		return agenttoolpolicy.ApplyOverride(policy, &agenttooltrust.ToolRuleOverride{
			ToolName: policy.Name,
			MaxTier:  agent.TierPropose,
		})
	}

	return agenttoolpolicy.ApplyOverrides(policy, rules)
}
