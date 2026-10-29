package scriptbroker

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/scriptoutbox"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolaccess"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/tooldispatch"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/toolschema"
)

const (
	scriptScopePrefix = "script:"
	scriptCallMark    = ".script."
	withheldKey       = "withheldByAccess"
)

const scriptWriteRefusal = "A script on this model reads only: %q changes something. Propose " +
	"the change with its own tool, outside the script."

const scriptUnpreparedRefusal = "%q changes something a script may not prepare: a script " +
	"prepares only changes that name the record they change. Propose it with its own tool, " +
	"outside the script."

type Broker struct {
	s             *tooldispatch.Dispatcher
	req           *agentruntimeports.RunRequest
	taint         *agent.RunTaint
	afterExternal bool
	index         []codemode.Read
	names         map[string]struct{}
	writeNames    map[string]struct{}
	scriptStepKey string
	outbox        *scriptoutbox.Outbox
	// reach is what the turn reached beyond the agent's own tools, each read
	// dispatched on its tier as the model's own call of it would be.
	reach map[string]agent.AutonomyTier

	mu       sync.Mutex
	callID   string
	limits   codemode.Limits
	slots    chan struct{}
	calls    []codemode.CallSummary
	usage    codemode.Usage
	marks    []agent.TaintMark
	withheld []string
	prepared []codemode.PreparedChange
}

// Open opens the broker a script's calls go through.
func Open(
	ctx context.Context,
	s *tooldispatch.Dispatcher,
	p *tooldispatch.ScriptBrokerParams,
) tooldispatch.ScriptBroker {
	return New(ctx, s, p)
}

func New(
	ctx context.Context,
	s *tooldispatch.Dispatcher,
	p *tooldispatch.ScriptBrokerParams,
) *Broker {
	scoped := *p.Req
	scope := p.ParentKey
	if scope == "" {
		scope = p.ParentCallID
	}
	scoped.ScriptScope = scriptScopePrefix + scope

	broker := &Broker{
		s:             s,
		req:           &scoped,
		taint:         p.Taint.Clone(),
		afterExternal: p.AfterExternal,
		callID:        p.ParentCallID,
		limits:        codemode.LimitsFor(codemode.MinSeconds, codemode.MinCalls),
		reach:         p.Reach,
	}
	broker.slots = make(chan struct{}, broker.limits.Concurrent)
	broker.index = scriptIndex(ctx, s, p.Req, p.Reach)
	broker.writeNames = map[string]struct{}{}
	if p.Writes {
		broker.scriptStepKey = scope
		broker.outbox = scriptoutbox.NewOutbox(scope, p.Req.Definition.ChangeLimit())
		for _, write := range scriptWriteIndex(ctx, s, p.Req) {
			broker.index = append(broker.index, write)
			broker.writeNames[write.Name] = struct{}{}
		}
	}
	broker.names = make(map[string]struct{}, len(broker.index))
	for idx := range broker.index {
		broker.names[broker.index[idx].Name] = struct{}{}
	}

	return broker
}

// scriptIndex is every read the turn holds that a script may call, with the
// schema the model reads for it, in name order: the agent's own and those the
// turn reached from other desks through find_tools, which the model may call.
func scriptIndex(
	ctx context.Context,
	s *tooldispatch.Dispatcher,
	req *agentruntimeports.RunRequest,
	reach map[string]agent.AutonomyTier,
) []codemode.Read {
	held := slices.Concat(s.Access().HeldTools(req.Definition), toolaccess.BeyondNames(reach))
	candidates := make([]string, 0, len(held))
	for _, name := range held {
		tool, ok := s.QueryTools().Get(name)
		if !ok || !scriptMayCall(tool) {
			continue
		}
		candidates = append(candidates, name)
	}
	permitted := s.Access().PermittedTools(ctx, req.Actor, candidates)
	slices.Sort(permitted)
	permitted = slices.Compact(permitted)

	index := make([]codemode.Read, 0, len(permitted))
	for _, name := range permitted {
		tool, _ := s.QueryTools().Get(name)
		index = append(index, codemode.Read{
			Name:        name,
			Description: tool.Description(),
			Parameters:  toolschema.ForModel(tool.ParamSchema()),
		})
	}

	return index
}

func scriptMayCall(tool agenttoolports.AgentQueryTool) bool {
	name := tool.Name()

	policy := tool.Policy()

	return policy.CallableFromScript() &&
		!toolaccess.ReadsExternalContent(name) &&
		!agentextension.IsExtensionTool(name)
}

func queryEffect(tool agenttoolports.AgentQueryTool) agenttooltrust.ToolEffect {
	policy := tool.Policy()

	return policy.EffectiveEffect()
}

// CallID names a read a script made after the call that ran the script,
// so its step in the ledger reads back to the script's own.
func CallID(parent string, seq int) string {
	return parent + scriptCallMark + strconv.Itoa(seq)
}

func (b *Broker) Index() []codemode.Read {
	return b.index
}

func (b *Broker) Bound(limits *codemode.Limits) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.limits = *limits
	b.slots = make(chan struct{}, max(limits.Concurrent, 1))
}

func (b *Broker) Read(ctx context.Context, call codemode.Call) codemode.Reply {
	if reply, refused := b.refusal(call); refused {
		b.record(&codemode.CallSummary{
			Seq: call.Seq, Tool: call.Tool, Status: reply.Status, Message: reply.Message,
		})
		return reply
	}
	if _, writes := b.writeNames[call.Tool]; writes {
		return b.prepare(ctx, call)
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
	outcome := b.s.GuardedDispatch(ctx, &tooldispatch.GuardedDispatchParams{
		Req: b.req,
		Call: llmports.ToolCall{
			ID:        CallID(b.callID, call.Seq),
			Name:      call.Tool,
			Arguments: call.Arguments,
		},
		Ordinal:       call.Seq,
		Taint:         b.taint,
		AfterExternal: b.afterExternal,
		Beyond:        b.reach[call.Tool],
		Caller:        aitrace.CallerScript,
		CallIndex:     call.Seq,
	})
	elapsed := time.Since(started).Milliseconds()

	b.mu.Lock()
	b.marks = append(b.marks, outcome.Taint...)
	b.mu.Unlock()

	if outcome.Failed {
		reply := codemode.Reply{Status: failedStatus(outcome.Verdict), Message: outcome.Content}
		b.record(&codemode.CallSummary{
			Seq: call.Seq, Tool: call.Tool, Status: reply.Status, BrokerMs: elapsed,
			Message: reply.Message,
		})
		return reply
	}

	return b.deliver(call, outcome.Data, elapsed)
}

func (b *Broker) refusal(call codemode.Call) (codemode.Reply, bool) {
	if _, indexed := b.names[call.Tool]; indexed {
		return codemode.Reply{}, false
	}
	if _, isAction := b.s.ActionTools().Get(call.Tool); isAction {
		refusal := scriptWriteRefusal
		if b.outbox != nil {
			refusal = scriptUnpreparedRefusal
		}
		return codemode.Reply{
			Status:  codemode.CallInvalid,
			Message: fmt.Sprintf(refusal, call.Tool),
		}, true
	}
	if tool, isQuery := b.s.QueryTools().Get(call.Tool); isQuery &&
		queryEffect(tool) != agenttooltrust.ToolEffectLookup {
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

func (b *Broker) deliver(call codemode.Call, data any, elapsed int64) codemode.Reply {
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

func (b *Broker) hitLocked(limit codemode.Limit) {
	if b.usage.Hit == codemode.LimitNone {
		b.usage.Hit = limit
	}
}

func (b *Broker) record(summary *codemode.CallSummary) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.calls = append(b.calls, *summary)
}

func (b *Broker) NoteReply(seq int, ms int64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for idx := range b.calls {
		if b.calls[idx].Seq == seq {
			b.calls[idx].ReplyMs = ms
			return
		}
	}
}

func (b *Broker) Summary() codemode.Summary {
	b.mu.Lock()
	defer b.mu.Unlock()

	calls := slices.Clone(b.calls)
	slices.SortStableFunc(calls, func(a, c codemode.CallSummary) int { return a.Seq - c.Seq })

	return codemode.Summary{
		Calls:    calls,
		Usage:    b.usage,
		Withheld: slices.Clone(b.withheld),
		Batch:    b.batchLocked(),
	}
}

func (b *Broker) TaintMarks() []agent.TaintMark {
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

func limitMessage(limit codemode.Limit) string {
	switch limit {
	case codemode.LimitCall:
		return "the read's answer is larger than one read may hand a script; narrow it"
	case codemode.LimitRows:
		return "the script has read every row it may"
	case codemode.LimitBytes:
		return "the script has read every byte it may"
	case codemode.LimitCalls:
		return "the script has made every read it may"
	case codemode.LimitNone, codemode.LimitWall, codemode.LimitOutput, codemode.LimitResult,
		codemode.LimitMemory, codemode.LimitCPU, codemode.LimitFrame:
		return "the script reached a limit"
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
		longest := 1
		for key, field := range value {
			if list, isList := field.([]any); isList && key != withheldKey && len(list) > longest {
				longest = len(list)
			}
		}
		return longest, withheldOf(value[withheldKey])
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
