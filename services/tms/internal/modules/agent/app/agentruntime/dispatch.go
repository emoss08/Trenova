package agentruntime

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolsimulation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type toolOutcome struct {
	content string
	failed  bool
	action  *agentports.PendingAction
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
	// handOff names the person's other agents a find_tools answer named as
	// holding what it matched, for the Desk's hand-off menu.
	handOff []pulid.ID
	verdict string
	reason  string
	// dispatched marks an outcome dispatch produced, which wrote its own
	// step; any other refusal is the loop's own (RunResult.LoopRefusals).
	dispatched bool
	// memories are the memories a recall read back, and saved the one a
	// remember kept or offered.
	memories []pulid.ID
	saved    *agentports.SavedMemory
	// shown is what the observer put in front of the person for the call.
	shown *agentports.ShownArtifact
}

func failedOutcome(format string, args ...any) toolOutcome {
	return toolOutcome{content: fmt.Sprintf(format, args...), failed: true}
}

func refusedOutcome(verdict, reason, format string, args ...any) toolOutcome {
	outcome := failedOutcome(format, args...)
	outcome.verdict = verdict
	outcome.reason = reason

	return outcome
}

// dispatchParams groups one tool call and everything deciding how it runs.
type dispatchParams struct {
	req            *agentports.RunRequest
	call           agentports.ToolCall
	completionText string
	// argumentNote is what the runtime changed about the call's arguments
	// before the tool read them, told back to the model with the result.
	argumentNote  string
	proposedSoFar []agentports.PendingAction
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
	stepKey       string
}

func (s *Service) dispatch(ctx context.Context, p dispatchParams) toolOutcome {
	req := p.req
	call := p.call
	completionText := p.completionText
	proposedSoFar := p.proposedSoFar
	// The loop refuses a tool the agent does not hold before it gets here, with
	// the nearest tools it does hold; this is the backstop for any other caller.
	if refusal, refused := s.extensionRefusal(ctx, req, call.Name); refused {
		refusal.verdict = aitrace.OutcomeDenied
		refusal.reason = "its extension is turned off or not set up"

		return refusal
	}
	if !s.holdsFor(ctx, req, call.Name) {
		return refusedOutcome(aitrace.OutcomeDenied, aianalytics.ReasonNotHeld,
			"Tool %q is not available to this agent.", call.Name)
	}

	selfScoped := agentports.IsSelfScoped(s.toolNamed(call.Name))
	if selfScoped && (req.Unattended || req.Actor == nil ||
		req.Actor.PrincipalType != permissionports.PrincipalTypeUser) {
		return refusedOutcome(
			aitrace.OutcomeDenied,
			"it acts only on the records of a person in the conversation, and none was",
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

		contract, err := contractCall(
			s.arguments, call.Name, tool.ParamSchema(), call.Arguments, organizationToday(req),
		)
		if err != nil {
			return argumentOutcome(call.Name, tool.ParamSchema(), err)
		}
		call.Arguments = contract.args

		outcome := s.runQueryTool(ctx, req, tool, call)
		if note := contract.note(); note != "" && !outcome.failed {
			outcome.content += "\n\n[" + note + "]"
		}

		return outcome
	}

	tool, ok := s.actionTools.Get(call.Name)
	if !ok {
		return refusedOutcome(aitrace.OutcomeInvalid, aianalytics.ReasonUnregistered,
			"Tool %q does not exist.", call.Name)
	}

	policy := s.ruled(ctx, req, tool.Policy())
	if !selfScoped {
		if outcome, denied := s.authorize(
			ctx, req.Actor, call.Name, policy.Resource, policy.Operation,
		); denied {
			return outcome
		}
	}

	contract, err := contractCall(
		s.arguments, call.Name, tool.ParamSchema(), call.Arguments, organizationToday(req),
	)
	if err != nil {
		return argumentOutcome(call.Name, tool.ParamSchema(), err)
	}
	call.Arguments = contract.args
	p.argumentNote = contract.note()
	if selfScoped {
		// Whose records these are is the runtime's to say, not the model's:
		// anything the model sent under this key is overwritten. A copy, so
		// the model's own call as recorded in the thread is left as it sent it.
		owned := make(map[string]any, len(call.Arguments)+1)
		maps.Copy(owned, call.Arguments)
		owned[agentports.SelfScopeOwnerParam] = req.Actor.UserID.String()
		call.Arguments = owned
	}
	if resolver, resolves := tool.(agentports.ToolSelectionResolver); resolves {
		resolved, rErr := resolver.ResolveSelection(
			ctx, runToolParams(req, p.idempotencyKey, call.Arguments),
		)
		if rErr != nil {
			return unresolvedSelection(call.Name, "proposed or run", rErr)
		}
		call.Arguments = resolved
	}
	p.call = call
	if outcome, refused := oversizedChange(
		req,
		call.Name,
		tool.ParamSchema(),
		call.Arguments,
	); refused {
		return outcome
	}
	tierParams := runToolParams(req, call.ID, call.Arguments)
	decision := s.decideCall(ctx, agenttoolpolicy.DecideInput{
		Policy:         policy,
		Params:         tierParams,
		Definition:     req.Definition,
		Unattended:     req.Unattended,
		Taint:          decisionTaint(p.taint, p.afterExternal),
		MoneyAllowance: s.moneyAllowance(req, proposedSoFar),
	})
	tier, heldForExternal := afterExternalContent(decision.Tier, p.afterExternal)
	tier, heldForHours := s.heldForBusinessHours(ctx, req, tier)
	if tier != agent.TierAutoExecute {
		if pinner, pins := tool.(agentports.ToolProposalSelectionResolver); pins {
			pinned, pErr := pinner.ResolveProposalSelection(
				ctx, runToolParams(req, p.idempotencyKey, call.Arguments),
			)
			if pErr != nil {
				return unresolvedSelection(call.Name, "proposed", pErr)
			}
			call.Arguments = pinned
			p.call = call
		}
	}
	action := &agentports.PendingAction{
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
		MoneyMinor: decision.MoneyMinor,
		Tainted:    p.taint.Tainted() || p.afterExternal,
	}
	if heldForExternal && !slices.Contains(action.HeldBy, agenttoolpolicy.HeldByTainted) {
		action.HeldBy = append(slices.Clone(action.HeldBy), agenttoolpolicy.HeldByTainted)
	}
	if heldForHours {
		action.HeldBy = append(slices.Clone(action.HeldBy), agenttoolpolicy.HeldByBusinessHours)
	}
	if tier != agent.TierAutoExecute {
		action.MoneyMinor = nil
	}
	source := decision.Source
	if heldForExternal || heldForHours {
		source = agent.TierSourcePolicyDefault
	}
	stampAction(ctx, action, source, p.stepKey)

	// A tool that can check its own arguments does so now, at every tier,
	// while the model can still fix the call: before a person is asked to
	// approve a proposal that was never going to run, before a simulation
	// records what it would have done, and before an automatic write runs.
	if validator, validates := tool.(agentports.ToolValidator); validates {
		if vErr := validator.Validate(
			ctx, runToolParams(req, p.idempotencyKey, call.Arguments),
		); vErr != nil {
			verb := "run"
			if tier != agent.TierAutoExecute {
				verb = "proposed"
			}

			return refusedOutcome(
				aitrace.OutcomeInvalid,
				vErr.Error(),
				"Tool %q was not %s, because it would fail as called: %s\n"+
					"Fix the call and try again.",
				call.Name, verb, vErr.Error(),
			)
		}
	}

	if tier != agent.TierAutoExecute {
		// The same write proposed twice is one decision asked for twice. A
		// model that never learned its earlier proposal was still waiting
		// used to raise it again on every "yes", and the person got a stack
		// of identical cards.
		if pendingDuplicate(call, req.Proposals, proposedSoFar) {
			return toolOutcome{
				content: duplicateProposalText(call.Name),
				verdict: aitrace.OutcomeDuplicate,
				reason:  "the same change is already waiting for a decision",
			}
		}

		// The write is previewed as it is filed. One that its own rules would
		// refuse is not filed: a person could only reject it, so the model is
		// told why and asked for what is missing instead. A write on a record
		// an earlier step of this turn changes is the exception, since that
		// step may be what makes it valid.
		dependsOnStep := dependsOnEarlierStep(tool, &call, proposedSoFar)
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

		content := fmt.Sprintf(
			"Recorded a proposal to run %q (proposal %s). It is awaiting a person's review at "+
				"the %s tier and has not run.",
			call.Name,
			action.ProposalID,
			tier,
		) + heldReason(action.HeldBy, decision.Egress) + filingEcho(baseline, call.Arguments)
		if note := p.argumentNote; note != "" {
			content += "\n" + note
		}
		if heldForExternal {
			content += externalContentNote
		}
		if heldForHours {
			content += businessHoursNote
		}
		if req.Definition.SimulationMode {
			content += " This agent is in simulation: an approval will preview the change, not make it."
		}

		return toolOutcome{content: content, action: action, verdict: aitrace.OutcomeProposed}
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

func unresolvedSelection(toolName, verb string, err error) toolOutcome {
	return refusedOutcome(
		aitrace.OutcomeInvalid,
		err.Error(),
		"Tool %q was not %s: its selection could not be resolved to records: %s\n"+
			"Fix the call and try again.",
		toolName, verb, err.Error(),
	)
}

func runToolParams(
	req *agentports.RunRequest,
	idempotencyKey string,
	arguments map[string]any,
) agentports.ToolExecuteParams {
	return agentports.ToolExecuteParams{
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

func definitionIDOf(req *agentports.RunRequest) pulid.ID {
	if req.Definition == nil {
		return pulid.Nil
	}

	return req.Definition.ID
}

// withinBudget refuses an automatic write past its tool's daily cap. The
// model is told which cap and to say so, rather than left to try again.
func (s *Service) withinBudget(
	ctx context.Context,
	req *agentports.RunRequest,
	toolName string,
	soFar []agentports.PendingAction,
) (toolOutcome, bool) {
	if s.budgets == nil {
		return toolOutcome{}, false
	}

	refusal, err := s.budgets.CheckTool(ctx, agentports.CheckToolBudgetRequest{
		Definition: req.Definition,
		ToolName:   toolName,
		Unrecorded: executedCount(soFar, toolName),
	})
	if err != nil {
		s.logger.Error("agent tool budget check failed",
			zap.String("tool", toolName), zap.Error(err))

		return refusedOutcome(
			aitrace.OutcomeFailed,
			"its daily cap could not be checked",
			"Tool %q was not run: its budget could not be checked. Try again later.",
			toolName,
		), true
	}
	if !refusal.Refused() {
		return toolOutcome{}, false
	}

	message := refusal.Message(req.Definition.Name)

	return refusedOutcome(aitrace.OutcomeOverBudget, message,
		"Tool %q was not run. %s Tell the person, and do not retry it.",
		toolName, message), true
}

func (s *Service) moneyAllowance(
	req *agentports.RunRequest,
	soFar []agentports.PendingAction,
) agenttoolpolicy.MoneyAllowanceReader {
	if s.budgets == nil {
		return nil
	}

	return func(ctx context.Context) (agent.MoneyAllowance, error) {
		allowance, err := s.budgets.MoneyAllowance(ctx, agentports.MoneyAllowanceRequest{
			Definition: req.Definition,
			RunID:      req.RunID,
			Unrecorded: unrecordedMoney(soFar),
		})
		if err != nil {
			s.logger.Warn("agent money limits could not be read; the write waits for a person",
				zap.Error(err))
		}

		return allowance, err
	}
}

func unrecordedMoney(actions []agentports.PendingAction) agent.MoneyUse {
	use := agent.MoneyUse{}
	for i := range actions {
		action := &actions[i]
		if action.Executed && !action.Simulated && action.MoneyMinor != nil {
			use.Calls++
			use.AmountMinor += intutils.Abs(*action.MoneyMinor)
		}
	}

	return use
}

// executedCount is how many times this turn already ran a tool for real. Those
// writes are recorded as proposals only when the turn ends.
func executedCount(actions []agentports.PendingAction, toolName string) int {
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

	tool   agentports.AgentTool
	action *agentports.PendingAction
}

// executeParams is what the tool is handed, built once so the simulated and
// the executed path cannot drift apart in what they pass.
func (a actionParams) executeParams() agentports.ToolExecuteParams {
	params := runToolParams(a.req, a.idempotencyKey, a.call.Arguments)
	if a.tool.Policy().CarriesTaint {
		params.Taint = a.taint.Clone()
	}

	return params
}

func (s *Service) simulateAction(ctx context.Context, a actionParams) toolOutcome {
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
	writeCtx, write := s.startWrite(ctx, &a, true)
	if s.previews != nil {
		action.Simulation = toolsimulation.FromBaseline(call.Name, call.Arguments, baseline)
	} else {
		params := a.executeParams()
		action.Simulation = toolsimulation.Simulate(writeCtx, a.tool, &params)
	}
	write.End()

	return toolOutcome{
		verdict: aitrace.OutcomeSimulated,
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
	actor *permissionports.RequestActor,
	toolName string,
	resource permission.Resource,
	operation permission.Operation,
) (toolOutcome, bool) {
	if s.permissions == nil {
		return refusedOutcome(aitrace.OutcomeFailed, "no permission engine could authorize it",
			"Tool %q could not be authorized.", toolName), true
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

		return refusedOutcome(aitrace.OutcomeFailed, "its permission could not be checked",
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

		return refusedOutcome(
			aitrace.OutcomeDenied,
			fmt.Sprintf("lacks %s access to %s", operation, resource.String()),
			"Tool %q is not permitted: %s. %s",
			toolName,
			deniedAccess(actor, operation, resource),
			deniedAdvice(actor, operation, resource),
		), true
	}

	return toolOutcome{}, false
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
func organizationToday(req *agentports.RunRequest) toolschema.CoerceOption {
	return toolschema.WithToday(
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

func (s *Service) runQueryTool(
	ctx context.Context,
	req *agentports.RunRequest,
	tool agentports.AgentQueryTool,
	call agentports.ToolCall,
) toolOutcome {
	data, err := tool.Query(ctx, &agentports.QueryToolParams{
		OrganizationID:    req.Actor.OrganizationID,
		BusinessUnitID:    req.Actor.BusinessUnitID,
		Actor:             req.Actor,
		Timezone:          req.Context.Timezone,
		Params:            call.Arguments,
		AgentDefinitionID: req.Definition.ID,
		DataAccessCeiling: req.Definition.DataAccessSensitivity(),
		ThreadID:          req.ThreadID,
		RunID:             req.RunID,
	})
	if err != nil {
		return refusedOutcome(aitrace.OutcomeFailed, err.Error(),
			"Tool %q failed: %s", call.Name, err.Error())
	}

	encoded, document, err := encodeToolDocument(data, timeutils.NowUnix(), req.Context.Timezone)
	if err != nil {
		return refusedOutcome(aitrace.OutcomeFailed, "its result could not be encoded",
			"Tool %q returned data that could not be encoded.", call.Name)
	}

	outcome := toolOutcome{
		content: FenceToolResult(call.Name, encoded),
		data:    data,
		summary: summarizeResult(call.Name, document),
		taint:   callTaint(s.ruled(ctx, req, tool.Policy()), call, data, timeutils.NowUnix()),
		verdict: aitrace.OutcomeRan,
	}
	if recall, ok := data.(agentports.MemoryRecall); ok {
		outcome.memories = recall.RecalledMemoryIDs()
	}

	return outcome
}

func (s *Service) executeAction(ctx context.Context, a actionParams) toolOutcome {
	call := a.call
	action := a.action

	action.Executed = true
	action.Target = s.snapshotTarget(ctx, a.req, a.tool, call)

	writeCtx, write := s.startWrite(ctx, &a, false)
	action.ExecutedAt = timeutils.NowUnix()
	var (
		result    *agent.ToolExecutionResult
		saved     *agentports.SavedMemory
		refreshed *agent.Memory
		err       error
	)
	if recorder, records := a.tool.(agentports.MemoryRecordingTool); records {
		var memory *agent.Memory
		memory, err = recorder.Record(writeCtx, a.executeParams())
		if err == nil && memory != nil {
			saved, refreshed = savedOrRefreshed(memory, call.ID)
		}
	} else {
		result, err = agentports.ExecuteTool(writeCtx, a.tool, a.executeParams())
	}
	if err != nil {
		aitrace.MarkFailed(write, aitrace.OutcomeFailed)
		write.End()
		action.ExecutionError = err.Error()

		return toolOutcome{
			content: fmt.Sprintf("Tool %q failed: %s", call.Name, err.Error()),
			failed:  true,
			action:  action,
			verdict: aitrace.OutcomeFailed,
			reason:  err.Error(),
		}
	}
	action.ExecutionResult = result
	action.ExecutedVersion = s.versionAfter(writeCtx, a.req, action.Target)
	if action.ExecutedVersion != nil {
		write.SetAttributes(aitrace.AIVersionAfter.Int64(*action.ExecutedVersion))
	}
	write.End()

	outcome := toolOutcome{
		content: ranContent(call.Name, result),
		action:  action,
		taint:   callTaint(s.ruled(ctx, a.req, a.tool.Policy()), call, nil, timeutils.NowUnix()),
		verdict: aitrace.OutcomeRan,
		saved:   saved,
	}
	switch {
	case refreshed != nil:
		outcome.content = refreshedContent(call.Name, refreshed)
		outcome.memories = []pulid.ID{refreshed.ID}
	case saved != nil:
		outcome.content = savedMemoryContent(call.Name, saved)
	}
	if note := a.argumentNote; note != "" {
		outcome.content += " " + note
	}

	return outcome
}

// savedMemoryContent tells the model whether what it asked to remember is
// kept or only offered. Both used to read "ran successfully", and a model
// told the person "I'll remember that" about a memory they had asked to be
// asked about first, which then sat on a card they had not decided.
func savedMemoryContent(name string, saved *agentports.SavedMemory) string {
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
	agent.HeldByMoneyUnbounded: "this tool moves money and cannot show that a call is " +
		"within the limits it may run in on its own",
	agent.HeldByMoneyUnproven: "it could not be shown that this change is within the " +
		"limits it may run in on its own",
	agent.HeldByMoneyExactMatch: "the money does not match its records exactly by " +
		"amount and reference",
	agent.HeldByMoneyRecordState: "the record is past the point where an agent may " +
		"change its money on its own",
	agent.HeldByMoneyAgentLine:    "a person added that line, so a person removes it",
	agent.HeldByMoneyLimitsUnread: "the organization's money limits for agents could not be read",
	agent.HeldByMoneyCallCeiling: "the amount is more than one automatic change may " +
		"move",
	agent.HeldByMoneyRunCalls: "this run has already made as many automatic money " +
		"changes as it may",
	agent.HeldByMoneyRunAmount: "it would take this run past the money it may move on " +
		"its own",
	agent.HeldByMoneyDayCalls: "the organization's agents have made as many automatic " +
		"money changes today as they may",
	agent.HeldByMoneyDayAmount: "it would take the organization's agents past the money " +
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
	default:
		return "of what the change would reach"
	}
}

// savedOrRefreshed sorts what a remember returned. A memory it kept or
// offered is shown to the person with an Undo. One it only refreshed is not:
// nothing new was kept, and undoing would retire the memory they already had,
// so the turn names it as a memory it used instead.
func savedOrRefreshed(
	memory *agent.Memory,
	callID string,
) (*agentports.SavedMemory, *agent.Memory) {
	if memory.Refreshed {
		return nil, memory
	}

	return &agentports.SavedMemory{
		ID:      memory.ID,
		CallID:  callID,
		Pending: memory.Status == agent.MemoryStatusSuggested,
	}, nil
}

// refreshedContent tells the model that what it asked to save was already
// remembered, so it says so to the person rather than announcing a new memory
// and does not try again with other words.
func refreshedContent(name string, memory *agent.Memory) string {
	return fmt.Sprintf(
		"Tool %q ran successfully. This was already remembered as memory %s, so that "+
			"memory was refreshed instead of saving a duplicate; nothing new was recorded.",
		name, memory.ID,
	)
}

func (s *Service) startWrite(
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

func (s *Service) versionAfter(
	ctx context.Context,
	req *agentports.RunRequest,
	target *agentports.ProposalTarget,
) *int64 {
	if target == nil || s.versions == nil {
		return nil
	}

	version, err := s.versions.Version(ctx, req.Actor.TenantInfo(), agentports.ToolTarget{
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
	req *agentports.RunRequest,
	tool agentports.AgentTool,
	call agentports.ToolCall,
) *agentports.ProposalTarget {
	targeted, ok := tool.(agentports.TargetedTool)
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

	return &agentports.ProposalTarget{Resource: target.Resource, ID: target.ID, Version: version}
}

// ruled is a tool's policy as the organization holds it: never looser than the
// rule declared beside the tool, and lower where the organization says so.
func (s *Service) ruled(
	ctx context.Context,
	req *agentports.RunRequest,
	policy agentports.ToolPolicy,
) agentports.ToolPolicy {
	if s.rules == nil || req == nil || req.Actor == nil {
		return policy
	}

	rules, err := s.rules.For(ctx, req.Actor.TenantInfo())
	if err != nil {
		s.logger.Error(
			"the organization's tool rules could not be read; holding the tool to a proposal",
			zap.String("tool", policy.Name),
			zap.Error(err),
		)
		return agenttoolpolicy.ApplyOverride(policy, &agent.ToolRuleOverride{
			ToolName: policy.Name,
			MaxTier:  agent.TierPropose,
		})
	}

	return agenttoolpolicy.ApplyOverrides(policy, rules)
}
