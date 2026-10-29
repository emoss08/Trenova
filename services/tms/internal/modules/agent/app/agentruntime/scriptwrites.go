package agentruntime

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/scriptoutbox"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	maxPreparedSummaryChars = 400
	splitCallMark           = "-"
	preparedNote            = "Prepared, not run. When the script ends, every change it " +
		"prepared is proposed as one batch that a person approves or rejects as a whole."
)

// scriptWriteIndex is every write the turn holds that a script may prepare,
// with its schema, in name order: a change to the tenant's records whose own
// preview shows a person what it would do to each record it names.
func (s *Service) scriptWriteIndex(
	ctx context.Context,
	req *agentruntimeports.RunRequest,
) []codemode.Read {
	held := s.heldTools(req.Definition)
	candidates := make([]string, 0, len(held))
	for _, name := range held {
		tool, ok := s.actionTools.Get(name)
		if !ok || !agenttoolports.PreparableFromScript(tool) {
			continue
		}
		candidates = append(candidates, name)
	}
	permitted := s.permittedTools(ctx, req.Actor, candidates)
	slices.Sort(permitted)
	permitted = slices.Compact(permitted)

	index := make([]codemode.Read, 0, len(permitted))
	for _, name := range permitted {
		tool, _ := s.actionTools.Get(name)
		index = append(index, codemode.Read{
			Name:        name,
			Description: tool.Description(),
			Parameters:  toolschema.ForModel(tool.ParamSchema()),
			Writes:      true,
		})
	}

	return index
}

// prepare takes a write the script asked for through everything dispatch
// decides for a write, as an intent under the script's own step, and holds
// it in the outbox. A call naming several records of a batch tool is that
// tool's call for each record alone. None of it runs.
func (b *scriptBroker) prepare(ctx context.Context, call codemode.Call) codemode.Reply {
	b.mu.Lock()
	if b.usage.Calls >= b.limits.Calls {
		b.hitLocked(codemode.LimitCalls)
		b.mu.Unlock()
		return b.answered(
			call,
			limitReply(codemode.LimitCalls, "the script has made every call it may"),
			0,
		)
	}
	b.usage.Calls++
	b.mu.Unlock()

	tool, ok := b.s.actionTools.Get(call.Tool)
	if !ok {
		return b.answered(call, codemode.Reply{
			Status:  codemode.CallRefused,
			Message: fmt.Sprintf("%q is not a change this script may prepare", call.Tool),
		}, 0)
	}
	singles, err := scriptoutbox.Expand(tool, call.Arguments)
	if err != nil {
		return b.answered(
			call,
			codemode.Reply{Status: codemode.CallInvalid, Message: err.Error()},
			0,
		)
	}

	started := time.Now()
	held := make([]heldIntent, 0, len(singles))
	for idx, arguments := range singles {
		intent, reply, refused := b.decideIntent(ctx, tool, call, idx, arguments)
		if refused {
			return b.answered(call, reply, time.Since(started).Milliseconds())
		}
		held = append(held, intent)
	}

	numbers := make([]int, 0, len(held))
	for idx := range held {
		seq, holdErr := b.outbox.Hold(held[idx].admission, held[idx].action)
		if holdErr != nil {
			return b.answered(call, refusedReply(holdErr), time.Since(started).Milliseconds())
		}
		numbers = append(numbers, seq)
		if held[idx].admission.Folded == 0 {
			b.notePrepared(seq, call.Tool, held[idx].summary)
		}
	}

	encoded, err := sonic.Marshal(map[string]any{
		"intents": numbers,
		"note":    preparedNote,
	})
	if err != nil {
		return b.answered(call, codemode.Reply{
			Status:  codemode.CallFailed,
			Message: "the prepared change could not be described",
		}, time.Since(started).Milliseconds())
	}

	return b.answered(call, codemode.Reply{Status: codemode.CallOK, Data: encoded},
		time.Since(started).Milliseconds())
}

type heldIntent struct {
	admission scriptoutbox.Admission
	action    *agentruntimeports.PendingAction
	summary   string
}

func (b *scriptBroker) decideIntent(
	ctx context.Context,
	tool agenttoolports.AgentTool,
	call codemode.Call,
	idx int,
	arguments map[string]any,
) (heldIntent, codemode.Reply, bool) {
	intent := scriptoutbox.Call{Tool: call.Tool, Arguments: arguments}
	if targeted, ok := tool.(agenttoolports.TargetedTool); ok {
		if target, pinned := targeted.Target(arguments); pinned {
			intent.Target = &target
		}
	}
	admission, err := b.outbox.Admit(intent)
	if err != nil {
		return heldIntent{}, refusedReply(err), true
	}
	if admission.Folded > 0 {
		return heldIntent{admission: admission}, codemode.Reply{}, false
	}

	callID := ScriptCallID(b.callID, call.Seq)
	if idx > 0 {
		callID += splitCallMark + strconv.Itoa(idx)
	}
	outcome := b.s.guardedDispatch(ctx, &guardedDispatchParams{
		req: b.req,
		call: llmports.ToolCall{
			ID:        callID,
			Name:      call.Tool,
			Arguments: arguments,
		},
		ordinal:       call.Seq,
		taint:         b.taint,
		afterExternal: b.afterExternal,
		caller:        aitrace.CallerScript,
		callIndex:     call.Seq,
		scriptStepKey: b.scriptStepKey,
	})
	if outcome.failed || outcome.action == nil {
		return heldIntent{}, codemode.Reply{
			Status:  failedStatus(outcome.verdict),
			Message: outcome.content,
		}, true
	}

	return heldIntent{
		admission: admission,
		action:    outcome.action,
		summary:   stringutils.Ellipsize(outcome.content, maxPreparedSummaryChars),
	}, codemode.Reply{}, false
}

func refusedReply(err error) codemode.Reply {
	return codemode.Reply{Status: codemode.CallRefused, Message: err.Error()}
}

func (b *scriptBroker) answered(
	call codemode.Call,
	reply codemode.Reply,
	elapsed int64,
) codemode.Reply {
	b.record(&codemode.CallSummary{
		Seq: call.Seq, Tool: call.Tool, Status: reply.Status, BrokerMs: elapsed,
		Message: reply.Message,
	})

	return reply
}

func (b *scriptBroker) notePrepared(seq int, tool, summary string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.prepared = append(
		b.prepared,
		codemode.PreparedChange{Intent: seq, Tool: tool, Summary: summary},
	)
}

func (b *scriptBroker) batchLocked() codemode.Batch {
	batch := codemode.Batch{Changes: slices.Clone(b.prepared)}
	if b.outbox != nil {
		if refusal := b.outbox.Refusal(); refusal != nil {
			batch.Refusal = refusal.Error()
		}
	}

	return batch
}

// scriptSettlement is how a script that may write ended, and what its whole
// run read: every intent is decided again with that, wherever in the script
// it was prepared.
type scriptSettlement struct {
	completed     bool
	taint         *agent.RunTaint
	afterExternal bool
	beyond        agent.AutonomyTier
	moneySpent    agenttooltrust.MoneyUse
}

// settle closes the script's outbox and returns the batch to file: nothing
// for a script that did not finish or whose batch was refused.
func (b *scriptBroker) settle(
	ctx context.Context,
	p *scriptSettlement,
) []agentruntimeports.PendingAction {
	if b.outbox == nil {
		return nil
	}
	req := b.req
	taint := decisionTaint(p.taint, p.afterExternal)
	actions, err := b.outbox.Settle(scriptoutbox.Settlement{
		Completed:       p.completed,
		Tainted:         p.taint.Tainted() || p.taint.Unknown(),
		ExternalContent: p.afterExternal,
		Redecide: func(action *agentruntimeports.PendingAction) agenttoolpolicy.Decision {
			tool, ok := b.s.actionTools.Get(action.ToolName)
			if !ok {
				return agenttoolpolicy.Decision{Tier: agent.TierPropose}
			}

			return b.s.decideCall(ctx, agenttoolpolicy.DecideInput{
				Policy:         b.s.ruled(ctx, req, tool.Policy()),
				Params:         runToolParams(req, action.ToolCallID, action.Arguments),
				Definition:     req.Definition,
				HolderCeiling:  p.beyond,
				Unattended:     req.Unattended,
				Taint:          taint,
				MoneyAllowance: b.s.moneyAllowance(req, p.moneySpent),
				FromScript:     true,
			})
		},
	})
	if err != nil {
		return nil
	}

	return actions
}
