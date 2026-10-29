package tooldispatch

import (
	"context"
	"maps"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/argcheck"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolcall"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"go.uber.org/zap"
)

// runScriptTool runs the tool that runs a script, once the call has passed
// every check a read passes: the code-mode gate decides on the model that
// asked, the turn's script count is held, and every read the script makes is
// handed to a broker bound to this call. The broker's taint marks, not the
// script's word, are what the call reports it read.
func (s *Dispatcher) runScriptTool(
	ctx context.Context,
	p *DispatchParams,
	tool agenttoolports.AgentQueryTool,
	call *llmports.ToolCall,
) toolcall.Result {
	req := p.req
	decision := s.codeMode.Decide(req.CodeMode, p.gate.Model, s.scriptSettings(ctx, req, p.gate))
	if !decision.Allowed {
		return toolcall.Refused(aitrace.OutcomeDenied, decision.Reason,
			"Tool %q was not run: %s. Answer with the reads you hold, one call at a time.",
			call.Name, decision.Reason)
	}
	if p.gate.Position > codemode.ScriptsPerTurn {
		return toolcall.Refused(
			aitrace.OutcomeOverBudget,
			"the turn ran every script it may",
			"Tool %q was not run: a turn runs at most %d scripts. Answer with what the scripts "+
				"already returned, or with discrete reads.",
			call.Name,
			codemode.ScriptsPerTurn,
		)
	}

	broker := s.brokers(ctx, s, &ScriptBrokerParams{
		Req:           req,
		ParentKey:     p.stepKey,
		ParentCallID:  call.ID,
		Taint:         p.taint,
		AfterExternal: p.afterExternal,
		Writes:        decision.Writes,
		Reach:         p.gate.Reach,
	})
	outcome := s.runQueryToolWith(
		ctx,
		req,
		tool,
		call,
		&queryExtras{broker: broker, stepKey: p.stepKey, continues: p.gate.Continues},
	)
	if !outcome.Failed {
		outcome.Continuation = scriptContinuation(outcome.Data)
		outcome.Unread = scriptReadNothing(outcome.Data)
	}
	marks := broker.TaintMarks()
	outcome.Taint = append(outcome.Taint, marks...)

	read := p.taint.Clone()
	if read == nil && len(marks) > 0 {
		read = &agent.RunTaint{}
	}
	read.Absorb(marks)
	outcome.Batch = broker.Settle(ctx, &ScriptSettlement{
		Completed:     !outcome.Failed && scriptCompleted(outcome.Data),
		Taint:         read,
		AfterExternal: p.afterExternal,
		Beyond:        p.beyond,
		MoneySpent:    p.moneySpent,
	})

	return outcome
}

// scriptSettings is what the provider that asked for the script holds now.
// A failed read runs no script.
func (s *Dispatcher) scriptSettings(
	ctx context.Context,
	req *agentruntimeports.RunRequest,
	gate ScriptGate,
) aiprovider.ScriptSettings {
	if req.CodeMode != codemode.OverrideDefault || s.scripts == nil || req.Actor == nil {
		return aiprovider.ScriptSettings{}
	}
	settings, err := s.scripts.ScriptSettings(ctx, gate.ProviderID, req.Actor.TenantInfo())
	if err != nil {
		s.logger.Warn("could not read the provider's script settings; the script is refused",
			zap.String("provider", gate.ProviderID.String()), zap.Error(err))
		return aiprovider.ScriptSettings{}
	}

	return settings
}

// inheritScriptName gives a continuation the title and purpose of the script
// it continues when it leaves them out, before its arguments are checked: it
// finishes that script, under its name. Sonnet sent continuations with only
// continues and code, as the parameter asks, and the required title refused
// them.
func inheritScriptName(arguments, continues map[string]any) map[string]any {
	if continues == nil {
		return arguments
	}
	inherited := make(map[string]any, len(arguments)+2)
	maps.Copy(inherited, arguments)
	for _, key := range []string{codemode.ParamScriptTitle, codemode.ParamScriptPurpose} {
		if strings.TrimSpace(argcheck.String(inherited, key)) != "" {
			continue
		}
		if earlier := argcheck.String(continues, key); earlier != "" {
			inherited[key] = earlier
		}
	}

	return inherited
}

func scriptContinuation(data any) codemode.Continuation {
	continuable, ok := data.(codemode.Continuable)
	if !ok {
		return codemode.ContinueNone
	}

	return continuable.ScriptContinuation()
}

// continuationRefusal refuses a call that continues a script it may not,
// before its arguments are checked: a continuation leaves out what the
// script it continues already has, and told "title is required" a model
// never learned that the script it named was the trouble.
func continuationRefusal(call *llmports.ToolCall, gate ScriptGate) (toolcall.Result, bool) {
	continues := strings.TrimSpace(
		argcheck.String(call.Arguments, codemode.ParamScriptContinues),
	)
	if continues == "" || gate.Continues != nil {
		return toolcall.Result{}, false
	}

	return toolcall.Refused(aitrace.OutcomeFailed, "it continues no script it may",
		"Tool %q was not run: continues names no script of this turn that ran to the end, or "+
			"one another call already continued. Send the whole script without continues.",
		call.Name), true
}

func scriptReadNothing(data any) bool {
	unread, ok := data.(codemode.Unread)

	return ok && unread.ScriptReadNothing()
}

func scriptCompleted(data any) bool {
	completion, ok := data.(codemode.Completion)

	return ok && completion.ScriptCompleted()
}
