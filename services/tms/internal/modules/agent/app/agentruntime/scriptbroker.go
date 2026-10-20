package agentruntime

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/toolschema"
)

const (
	scriptScopePrefix = "script:"
	scriptCallMark    = ".script."
	withheldKey       = "withheldByAccess"
)

const scriptWriteRefusal = "A script reads only: %q changes something, and changes from a script " +
	"arrive in Phase 5b. Propose the change with its own tool, outside the script."

type scriptBroker struct {
	s             *Service
	req           *agentports.RunRequest
	taint         *agent.RunTaint
	afterExternal bool
	index         []codemode.Read
	names         map[string]struct{}

	mu       sync.Mutex
	callID   string
	limits   codemode.Limits
	slots    chan struct{}
	calls    []codemode.CallSummary
	usage    codemode.Usage
	marks    []agent.TaintMark
	withheld []string
}

type scriptBrokerParams struct {
	req           *agentports.RunRequest
	parentKey     string
	parentCallID  string
	taint         *agent.RunTaint
	afterExternal bool
}

func (s *Service) newScriptBroker(ctx context.Context, p *scriptBrokerParams) *scriptBroker {
	scoped := *p.req
	scope := p.parentKey
	if scope == "" {
		scope = p.parentCallID
	}
	scoped.ScriptScope = scriptScopePrefix + scope

	broker := &scriptBroker{
		s:             s,
		req:           &scoped,
		taint:         p.taint.Clone(),
		afterExternal: p.afterExternal,
		callID:        p.parentCallID,
		limits:        codemode.LimitsFor(codemode.MinSeconds, codemode.MinCalls),
	}
	broker.slots = make(chan struct{}, broker.limits.Concurrent)
	broker.index = s.scriptIndex(ctx, p.req)
	broker.names = make(map[string]struct{}, len(broker.index))
	for idx := range broker.index {
		broker.names[broker.index[idx].Name] = struct{}{}
	}

	return broker
}

// scriptIndex is every read the turn holds that a script may call, with the
// schema the model reads for it, in name order.
func (s *Service) scriptIndex(ctx context.Context, req *agentports.RunRequest) []codemode.Read {
	held := s.heldTools(req.Definition)
	candidates := make([]string, 0, len(held))
	for _, name := range held {
		tool, ok := s.queryTools.Get(name)
		if !ok || !scriptMayCall(tool) {
			continue
		}
		candidates = append(candidates, name)
	}
	permitted := s.permittedTools(ctx, req.Actor, candidates)
	slices.Sort(permitted)
	permitted = slices.Compact(permitted)

	index := make([]codemode.Read, 0, len(permitted))
	for _, name := range permitted {
		tool, _ := s.queryTools.Get(name)
		index = append(index, codemode.Read{
			Name:        name,
			Description: tool.Description(),
			Parameters:  toolschema.ForModel(tool.ParamSchema()),
		})
	}

	return index
}

func scriptMayCall(tool agentports.AgentQueryTool) bool {
	name := tool.Name()

	policy := tool.Policy()

	return policy.CallableFromScript() &&
		!ReadsExternalContent(name) &&
		!agentextension.IsExtensionTool(name)
}

// ScriptCallID names a read a script made after the call that ran the script,
// so its step in the ledger reads back to the script's own.
func ScriptCallID(parent string, seq int) string {
	return parent + scriptCallMark + strconv.Itoa(seq)
}

func (b *scriptBroker) Index() []codemode.Read {
	return b.index
}

func (b *scriptBroker) Bound(limits *codemode.Limits) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.limits = *limits
	b.slots = make(chan struct{}, max(limits.Concurrent, 1))
}

func (b *scriptBroker) Read(ctx context.Context, call codemode.Call) codemode.Reply {
	if reply, refused := b.refusal(call); refused {
		b.record(&codemode.CallSummary{
			Seq: call.Seq, Tool: call.Tool, Status: reply.Status, Message: reply.Message,
		})
		return reply
	}

	b.mu.Lock()
	if b.usage.Calls >= b.limits.Calls {
		b.hitLocked(codemode.LimitCalls)
		b.mu.Unlock()
		reply := limitReply(codemode.LimitCalls, "the script has made every read it may")
		b.record(&codemode.CallSummary{
			Seq: call.Seq, Tool: call.Tool, Status: reply.Status, Message: reply.Message,
		})
		return reply
	}
	b.usage.Calls++
	slots := b.slots
	b.mu.Unlock()

	select {
	case slots <- struct{}{}:
	case <-ctx.Done():
		return codemode.Reply{Status: codemode.CallFailed, Message: "the script was stopped"}
	}
	defer func() { <-slots }()

	started := time.Now()
	outcome := b.s.guardedDispatch(ctx, &guardedDispatchParams{
		req: b.req,
		call: agentports.ToolCall{
			ID:        ScriptCallID(b.callID, call.Seq),
			Name:      call.Tool,
			Arguments: call.Arguments,
		},
		ordinal:       call.Seq,
		taint:         b.taint,
		afterExternal: b.afterExternal,
		caller:        aitrace.CallerScript,
		callIndex:     call.Seq,
	})
	elapsed := time.Since(started).Milliseconds()

	b.mu.Lock()
	b.marks = append(b.marks, outcome.taint...)
	b.mu.Unlock()

	if outcome.failed {
		reply := codemode.Reply{Status: failedStatus(outcome.verdict), Message: outcome.content}
		b.record(&codemode.CallSummary{
			Seq: call.Seq, Tool: call.Tool, Status: reply.Status, BrokerMs: elapsed,
			Message: reply.Message,
		})
		return reply
	}

	return b.deliver(call, outcome.data, elapsed)
}

func (b *scriptBroker) refusal(call codemode.Call) (codemode.Reply, bool) {
	if _, indexed := b.names[call.Tool]; indexed {
		return codemode.Reply{}, false
	}
	if _, isAction := b.s.actionTools.Get(call.Tool); isAction {
		return codemode.Reply{
			Status:  codemode.CallInvalid,
			Message: fmt.Sprintf(scriptWriteRefusal, call.Tool),
		}, true
	}
	if tool, isQuery := b.s.queryTools.Get(call.Tool); isQuery &&
		tool.Policy().EffectiveEffect() != agent.ToolEffectLookup {
		return codemode.Reply{
			Status:  codemode.CallInvalid,
			Message: fmt.Sprintf(scriptWriteRefusal, call.Tool),
		}, true
	}

	return codemode.Reply{
		Status: codemode.CallRefused,
		Message: fmt.Sprintf("%q is not a read this script may call; trenova.tools() lists the "+
			"reads it holds", call.Tool),
	}, true
}

func (b *scriptBroker) deliver(call codemode.Call, data any, elapsed int64) codemode.Reply {
	encoded, err := sonic.Marshal(data)
	if err != nil {
		reply := codemode.Reply{
			Status:  codemode.CallFailed,
			Message: "the read's answer could not be encoded",
		}
		b.record(&codemode.CallSummary{
			Seq: call.Seq, Tool: call.Tool, Status: reply.Status, BrokerMs: elapsed,
			Message: reply.Message,
		})
		return reply
	}
	rows, withheld := shapeOf(encoded)

	b.mu.Lock()
	limits := b.limits
	var hit codemode.Limit
	switch {
	case len(encoded) > limits.CallBytes:
		hit = codemode.LimitCall
	case b.usage.Rows+rows > limits.Rows:
		hit = codemode.LimitRows
	case b.usage.Bytes+len(encoded) > limits.Bytes:
		hit = codemode.LimitBytes
	}
	if hit != codemode.LimitNone {
		b.hitLocked(hit)
		b.mu.Unlock()
		reply := limitReply(hit, limitMessage(hit))
		b.record(&codemode.CallSummary{
			Seq: call.Seq, Tool: call.Tool, Status: reply.Status, Rows: rows,
			Bytes: len(encoded), BrokerMs: elapsed, Message: reply.Message,
		})
		return reply
	}
	b.usage.Rows += rows
	b.usage.Bytes += len(encoded)
	for _, field := range withheld {
		if !slices.Contains(b.withheld, field) {
			b.withheld = append(b.withheld, field)
		}
	}
	b.mu.Unlock()

	b.record(&codemode.CallSummary{
		Seq: call.Seq, Tool: call.Tool, Status: codemode.CallOK, Rows: rows,
		Bytes: len(encoded), BrokerMs: elapsed,
	})

	return codemode.Reply{Status: codemode.CallOK, Data: encoded}
}

func (b *scriptBroker) hitLocked(limit codemode.Limit) {
	if b.usage.Hit == codemode.LimitNone {
		b.usage.Hit = limit
	}
}

func (b *scriptBroker) record(summary *codemode.CallSummary) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.calls = append(b.calls, *summary)
}

func (b *scriptBroker) NoteReply(seq int, ms int64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for idx := range b.calls {
		if b.calls[idx].Seq == seq {
			b.calls[idx].ReplyMs = ms
			return
		}
	}
}

func (b *scriptBroker) Summary() codemode.Summary {
	b.mu.Lock()
	defer b.mu.Unlock()

	calls := slices.Clone(b.calls)
	slices.SortStableFunc(calls, func(a, c codemode.CallSummary) int { return a.Seq - c.Seq })

	return codemode.Summary{
		Calls:    calls,
		Usage:    b.usage,
		Withheld: slices.Clone(b.withheld),
	}
}

func (b *scriptBroker) taintMarks() []agent.TaintMark {
	b.mu.Lock()
	defer b.mu.Unlock()

	return slices.Clone(b.marks)
}

func failedStatus(verdict string) codemode.CallStatus {
	switch verdict {
	case aitrace.OutcomeDenied:
		return codemode.CallRefused
	case aitrace.OutcomeInvalid:
		return codemode.CallInvalid
	case aitrace.OutcomeOverBudget:
		return codemode.CallLimit
	default:
		return codemode.CallFailed
	}
}

func limitReply(limit codemode.Limit, message string) codemode.Reply {
	return codemode.Reply{Status: codemode.CallLimit, Limit: limit, Message: message}
}

var limitMessages = map[codemode.Limit]string{
	codemode.LimitCall:  "the read's answer is larger than one read may hand a script; narrow it",
	codemode.LimitRows:  "the script has read every row it may",
	codemode.LimitBytes: "the script has read every byte it may",
}

func limitMessage(limit codemode.Limit) string {
	if message, ok := limitMessages[limit]; ok {
		return message
	}

	return "the script reached a limit"
}

// shapeOf counts the rows an answer holds (a list's length, or the longest
// list at its top level) and the fields its tool withheld from the caller.
func shapeOf(encoded []byte) (rows int, withheld []string) {
	var decoded any
	if err := sonic.Unmarshal(encoded, &decoded); err != nil {
		return 1, nil
	}

	switch value := decoded.(type) {
	case []any:
		return len(value), nil
	case map[string]any:
		rows := 1
		for key, field := range value {
			if list, isList := field.([]any); isList && key != withheldKey && len(list) > rows {
				rows = len(list)
			}
		}
		return rows, withheldOf(value[withheldKey])
	default:
		return 1, nil
	}
}

func withheldOf(raw any) []string {
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	fields := make([]string, 0, len(list))
	for _, entry := range list {
		if field, isText := entry.(string); isText {
			fields = append(fields, field)
		}
	}

	return fields
}
