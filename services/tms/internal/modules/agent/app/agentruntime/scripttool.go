package agentruntime

import (
	"context"
	"maps"
	"slices"

	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"go.uber.org/zap"
)

// runScriptTool runs the tool that runs a script, once the call has passed
// every check a read passes: the code-mode gate decides on the model that
// asked, the turn's script count is held, and every read the script makes is
// handed to a broker bound to this call. The broker's taint marks, not the
// script's word, are what the call reports it read.
func (s *Service) runScriptTool(
	ctx context.Context,
	p *dispatchParams,
	tool agentports.AgentQueryTool,
	call *agentports.ToolCall,
) toolOutcome {
	req := p.req
	decision := s.codeMode.Decide(req.CodeMode, p.gate.model, s.scriptSettings(ctx, req, p.gate))
	if !decision.Allowed {
		return refusedOutcome(aitrace.OutcomeDenied, decision.Reason,
			"Tool %q was not run: %s. Answer with the reads you hold, one call at a time.",
			call.Name, decision.Reason)
	}
	if p.gate.position > codemode.ScriptsPerTurn {
		return refusedOutcome(aitrace.OutcomeOverBudget, "the turn ran every script it may",
			"Tool %q was not run: a turn runs at most %d scripts. Answer with what the scripts "+
				"already returned, or with discrete reads.", call.Name, codemode.ScriptsPerTurn)
	}

	broker := s.newScriptBroker(ctx, &scriptBrokerParams{
		req:           req,
		parentKey:     p.stepKey,
		parentCallID:  call.ID,
		taint:         p.taint,
		afterExternal: p.afterExternal,
		writes:        decision.Writes,
	})
	outcome := s.runQueryToolWith(
		ctx,
		req,
		tool,
		call,
		&queryExtras{broker: broker, stepKey: p.stepKey},
	)
	marks := broker.taintMarks()
	outcome.taint = append(outcome.taint, marks...)

	read := p.taint.Clone()
	if read == nil && len(marks) > 0 {
		read = &agent.RunTaint{}
	}
	read.Absorb(marks)
	outcome.batch = broker.settle(ctx, &scriptSettlement{
		completed:     !outcome.failed && scriptCompleted(outcome.data),
		taint:         read,
		afterExternal: p.afterExternal,
		beyond:        p.beyond,
		moneySpent:    p.moneySpent,
	})

	return outcome
}

// scriptsOffered reports whether a turn opening now may be shown the script
// tool: some provider that may answer it has scripts on. Only a turn left to
// the providers' settings asks; a failed read offers nothing.
func (s *Service) scriptsOffered(ctx context.Context, req *agentports.RunRequest) bool {
	if req.CodeMode != codemode.OverrideDefault || s.scripts == nil || req.Actor == nil {
		return false
	}
	offered, err := s.scripts.ScriptsOffered(ctx, req.Actor.TenantInfo())
	if err != nil {
		s.logger.Warn("could not read the providers' script settings; scripts are not offered",
			zap.Error(err))
		return false
	}

	return offered
}

// scriptSettings is what the provider that asked for the script holds now.
// A failed read runs no script.
func (s *Service) scriptSettings(
	ctx context.Context,
	req *agentports.RunRequest,
	gate scriptGate,
) aiprovider.ScriptSettings {
	if req.CodeMode != codemode.OverrideDefault || s.scripts == nil || req.Actor == nil {
		return aiprovider.ScriptSettings{}
	}
	settings, err := s.scripts.ScriptSettings(ctx, gate.providerID, req.Actor.TenantInfo())
	if err != nil {
		s.logger.Warn("could not read the provider's script settings; the script is refused",
			zap.String("provider", gate.providerID.String()), zap.Error(err))
		return aiprovider.ScriptSettings{}
	}

	return settings
}

// withoutScripts takes the tool that runs a script out of a turn the code-mode
// gate keeps it from, so the model is never shown a tool it may not call.
func (s *Service) withoutScripts(
	definition *agentdefinition.Definition,
	held []string,
	beyond map[string]agent.AutonomyTier,
) (kept []string, reached map[string]agent.AutonomyTier) {
	if held == nil {
		held = s.heldTools(definition)
	}
	kept = slices.DeleteFunc(slices.Clone(held), isScriptRunner)
	reached = beyond
	if _, found := beyond[codemode.ToolRunScript]; found {
		reached = maps.Clone(beyond)
		delete(reached, codemode.ToolRunScript)
	}

	return kept, reached
}

func isScriptRunner(name string) bool {
	return name == codemode.ToolRunScript
}

func scriptCompleted(data any) bool {
	completion, ok := data.(codemode.Completion)

	return ok && completion.ScriptCompleted()
}
