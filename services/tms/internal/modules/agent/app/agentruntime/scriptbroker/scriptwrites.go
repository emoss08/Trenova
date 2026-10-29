package scriptbroker

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/scriptoutbox"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolaccess"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/tooldispatch"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
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
func scriptWriteIndex(
	ctx context.Context,
	s *tooldispatch.Dispatcher,
	req *agentruntimeports.RunRequest,
) []codemode.Read {
	held := s.Access().HeldTools(req.Definition)
	candidates := make([]string, 0, len(held))
	for _, name := range held {
		tool, ok := s.ActionTools().Get(name)
		if !ok || !agenttoolports.PreparableFromScript(tool) {
			continue
		}
		candidates = append(candidates, name)
	}
	permitted := s.Access().PermittedTools(ctx, req.Actor, candidates)
	slices.Sort(permitted)
	permitted = slices.Compact(permitted)

	index := make([]codemode.Read, 0, len(permitted))
	for _, name := range permitted {
		tool, _ := s.ActionTools().Get(name)
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
func (b *Broker) prepare(ctx context.Context, call codemode.Call) codemode.Reply {
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

	tool, ok := b.s.ActionTools().Get(call.Tool)
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

func (b *Broker) decideIntent(
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

	callID := CallID(b.callID, call.Seq)
	if idx > 0 {
		callID += splitCallMark + strconv.Itoa(idx)
	}
	outcome := b.s.GuardedDispatch(ctx, &tooldispatch.GuardedDispatchParams{
		Req: b.req,
		Call: llmports.ToolCall{
			ID:        callID,
			Name:      call.Tool,
			Arguments: arguments,
		},
		Ordinal:       call.Seq,
		Taint:         b.taint,
		AfterExternal: b.afterExternal,
		Caller:        aitrace.CallerScript,
		CallIndex:     call.Seq,
		ScriptStepKey: b.scriptStepKey,
	})
	if outcome.Failed || outcome.Action == nil {
		return heldIntent{}, codemode.Reply{
			Status:  failedStatus(outcome.Verdict),
			Message: outcome.Content,
		}, true
	}

	return heldIntent{
		admission: admission,
		action:    outcome.Action,
		summary:   stringutils.Ellipsize(outcome.Content, maxPreparedSummaryChars),
	}, codemode.Reply{}, false
}

func refusedReply(err error) codemode.Reply {
	return codemode.Reply{Status: codemode.CallRefused, Message: err.Error()}
}

func (b *Broker) answered(
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

func (b *Broker) notePrepared(seq int, tool, summary string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.prepared = append(
		b.prepared,
		codemode.PreparedChange{Intent: seq, Tool: tool, Summary: summary},
	)
}

func (b *Broker) batchLocked() codemode.Batch {
	batch := codemode.Batch{Changes: slices.Clone(b.prepared)}
	if b.outbox != nil {
		if refusal := b.outbox.Refusal(); refusal != nil {
			batch.Refusal = refusal.Error()
		}
	}

	return batch
}

// Settle closes the script's outbox and returns the batch to file: nothing
// for a script that did not finish or whose batch was refused.
func (b *Broker) Settle(
	ctx context.Context,
	p *tooldispatch.ScriptSettlement,
) []agentruntimeports.PendingAction {
	if b.outbox == nil {
		return nil
	}
	req := b.req
	taint := toolaccess.DecisionTaint(p.Taint, p.AfterExternal)
	actions, err := b.outbox.Settle(scriptoutbox.Settlement{
		Completed:       p.Completed,
		Tainted:         p.Taint.Tainted() || p.Taint.Unknown(),
		ExternalContent: p.AfterExternal,
		Redecide: func(action *agentruntimeports.PendingAction) agenttoolpolicy.Decision {
			tool, ok := b.s.ActionTools().Get(action.ToolName)
			if !ok {
				return agenttoolpolicy.Decision{Tier: agent.TierPropose}
			}

			return b.s.DecideCall(ctx, agenttoolpolicy.DecideInput{
				Policy: b.s.Ruled(ctx, req, tool.Policy()),
				Params: tooldispatch.RunToolParams(
					req,
					action.ToolCallID,
					action.Arguments,
				),
				Definition:     req.Definition,
				HolderCeiling:  p.Beyond,
				Unattended:     req.Unattended,
				Taint:          taint,
				MoneyAllowance: b.s.MoneyAllowance(req, p.MoneySpent),
				FromScript:     true,
			})
		},
	})
	if err != nil {
		return nil
	}

	return actions
}
