package agentruntime

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolsimulation"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

type toolOutcome struct {
	content string
	failed  bool
	action  *serviceports.PendingAction
}

func failedOutcome(format string, args ...any) toolOutcome {
	return toolOutcome{content: fmt.Sprintf(format, args...), failed: true}
}

func (s *Service) dispatch(
	ctx context.Context,
	req *serviceports.RunRequest,
	call serviceports.ToolCall,
	completionText string,
	proposedSoFar []serviceports.PendingAction,
) toolOutcome {
	// A tool the agent holds but has not been sent this turn still resolves
	// from the registry below: the configured list is the grant, and disclosure
	// only decides what the model was shown. One it does not hold is refused,
	// and told where to look rather than left to guess again.
	if !req.Definition.AllowsTool(call.Name) {
		return failedOutcome(
			"Tool %q is not available to this agent. Call find_tools to see what is, "+
				"or use one of the tools already loaded.",
			call.Name,
		)
	}

	if tool, ok := s.queryTools.Get(call.Name); ok {
		if outcome, denied := s.authorize(ctx, req.Actor, call.Name, tool.PermissionResource(), permission.OpRead); denied {
			return outcome
		}

		return s.runQueryTool(ctx, req, tool, call)
	}

	tool, ok := s.actionTools.Get(call.Name)
	if !ok {
		return failedOutcome("Tool %q does not exist.", call.Name)
	}

	if outcome, denied := s.authorize(ctx, req.Actor, call.Name, tool.PermissionResource(), tool.PermissionOperation()); denied {
		return outcome
	}

	tier := req.Definition.EffectiveTier(call.Name, tool.DefaultAutonomyTier())
	call.Arguments = declaredArguments(tool.ParamSchema(), call.Arguments)
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
				IdempotencyKey: call.ID,
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
			call.Name, tier,
		)
		if req.Definition.SimulationMode {
			content += " This agent is in simulation: an approval will preview the change, not make it."
		}

		return toolOutcome{content: content, action: action}
	}

	// An automatic write in simulation is previewed where it would have run,
	// and recorded as such, so a person can read what the agent would have
	// done at full reach before it is given any.
	if req.Definition.SimulationMode {
		return s.simulateAction(ctx, req, tool, call, action)
	}

	if outcome, refused := s.withinBudget(ctx, req, call.Name); refused {
		return outcome
	}

	return s.executeAction(ctx, req, tool, call, action)
}

// withinBudget refuses an automatic write past its tool's daily cap. The
// model is told which cap and to say so, rather than left to try again.
func (s *Service) withinBudget(
	ctx context.Context,
	req *serviceports.RunRequest,
	toolName string,
) (toolOutcome, bool) {
	if s.budgets == nil {
		return toolOutcome{}, false
	}

	refusal, err := s.budgets.CheckTool(ctx, req.Definition, toolName)
	if err != nil {
		s.logger.Error("agent tool budget check failed",
			zap.String("tool", toolName), zap.Error(err))

		return failedOutcome("Tool %q was not run: its budget could not be checked. Try again later.", toolName), true
	}
	if !refusal.Refused() {
		return toolOutcome{}, false
	}

	return failedOutcome("Tool %q was not run. %s Tell the person, and do not retry it.",
		toolName, refusal.Message(req.Definition.Name)), true
}

func (s *Service) simulateAction(
	ctx context.Context,
	req *serviceports.RunRequest,
	tool serviceports.AgentTool,
	call serviceports.ToolCall,
	action *serviceports.PendingAction,
) toolOutcome {
	action.Simulated = true
	action.Simulation = toolsimulation.Simulate(ctx, tool, serviceports.ToolExecuteParams{
		OrganizationID: req.Actor.OrganizationID,
		BusinessUnitID: req.Actor.BusinessUnitID,
		Actor:          req.Actor,
		IdempotencyKey: call.ID,
		RunID:          req.RunID,
		Params:         call.Arguments,
	})

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
			toolName, operation, resource.String(),
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
	data, err := tool.Query(ctx, serviceports.QueryToolParams{
		OrganizationID: req.Actor.OrganizationID,
		BusinessUnitID: req.Actor.BusinessUnitID,
		Actor:          req.Actor,
		Timezone:       req.Context.Timezone,
		Params:         call.Arguments,
	})
	if err != nil {
		return failedOutcome("Tool %q failed: %s", call.Name, err.Error())
	}

	encoded, err := encodeToolResult(data, timeutils.NowUnix(), req.Context.Timezone)
	if err != nil {
		return failedOutcome("Tool %q returned data that could not be encoded.", call.Name)
	}

	return toolOutcome{content: FenceToolResult(call.Name, encoded)}
}

func (s *Service) executeAction(
	ctx context.Context,
	req *serviceports.RunRequest,
	tool serviceports.AgentTool,
	call serviceports.ToolCall,
	action *serviceports.PendingAction,
) toolOutcome {
	action.Executed = true

	err := tool.Execute(ctx, serviceports.ToolExecuteParams{
		OrganizationID: req.Actor.OrganizationID,
		BusinessUnitID: req.Actor.BusinessUnitID,
		Actor:          req.Actor,
		IdempotencyKey: call.ID,
		RunID:          req.RunID,
		Params:         call.Arguments,
	})
	if err != nil {
		action.ExecutionError = err.Error()

		return toolOutcome{
			content: fmt.Sprintf("Tool %q failed: %s", call.Name, err.Error()),
			failed:  true,
			action:  action,
		}
	}

	return toolOutcome{
		content: fmt.Sprintf("Tool %q ran successfully.", call.Name),
		action:  action,
	}
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
