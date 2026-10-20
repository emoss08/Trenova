package agentruntime

import (
	"context"
	"maps"
	"slices"

	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
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
	call agentports.ToolCall,
) toolOutcome {
	req := p.req
	decision := s.codeMode.Decide(req.CodeMode, p.gate.provider, p.gate.model)
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
	})
	outcome := s.runQueryToolWith(ctx, req, tool, call, broker, p.stepKey)
	outcome.taint = append(outcome.taint, broker.taintMarks()...)

	return outcome
}

// withoutScripts takes the tool that runs a script out of a turn the code-mode
// gate keeps it from, so the model is never shown a tool it may not call.
func (s *Service) withoutScripts(
	definition *agentdefinition.Definition,
	held []string,
	beyond map[string]agent.AutonomyTier,
) ([]string, map[string]agent.AutonomyTier) {
	if held == nil {
		held = s.heldTools(definition)
	}
	kept := slices.DeleteFunc(slices.Clone(held), isScriptRunner)
	if _, reached := beyond[agentextension.ToolRunScript]; reached {
		beyond = maps.Clone(beyond)
		delete(beyond, agentextension.ToolRunScript)
	}

	return kept, beyond
}

func isScriptRunner(name string) bool {
	return name == agentextension.ToolRunScript
}
