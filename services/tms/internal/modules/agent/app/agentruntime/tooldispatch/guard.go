package tooldispatch

import (
	"context"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/loopguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolcall"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/pulid"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Ordinals counts how many times each distinct call has been made in this
// attempt, so the step key of a legitimately repeated call differs from the
// first one's.
type Ordinals struct {
	Seen map[string]int
}

func NewOrdinals() *Ordinals {
	return &Ordinals{Seen: make(map[string]int, 4)}
}

// Next returns this call's ordinal and counts it.
func (o *Ordinals) Next(call *llmports.ToolCall) int {
	key, ok := loopguard.CallKey(call)
	if !ok {
		return 0
	}

	ordinal := o.Seen[key]
	o.Seen[key] = ordinal + 1

	return ordinal
}

// GuardedDispatchParams groups what running one guarded call needs.
type GuardedDispatchParams struct {
	Req            *agentruntimeports.RunRequest
	Call           llmports.ToolCall
	CompletionText string
	ProposedSoFar  []agentruntimeports.PendingAction
	MoneySpent     agenttooltrust.MoneyUse
	// Ordinal numbers this exact call within the run; the loop assigns it.
	Ordinal int
	Taint   *agent.RunTaint
	// AfterExternal says the turn has read content from outside the
	// organization, so no write may run without a person.
	AfterExternal bool
	Beyond        agent.AutonomyTier
	// ScriptStepKey is the ledger step of the script whose broker made the
	// call; a write under it is an intent and never runs.
	ScriptStepKey string
	Gate          ScriptGate
	Caller        string
	CallIndex     int
}

// ScriptGate is what decides whether a call may run a script: the model that
// asked for it and how many scripts the turn asked for up to it.
type ScriptGate struct {
	Model      string
	Provider   aiprovider.Kind
	ProviderID pulid.ID
	Position   int
	// Continues is the arguments of the script the call continues.
	Continues map[string]any
	// Reach is what the turn reached beyond the agent's own tools, which a
	// script may call as the model may.
	Reach map[string]agent.AutonomyTier
}

// GuardedDispatch runs one tool call at most once across every attempt of a
// run.
//
// The key is claimed before the tool runs, not after. Recording afterwards
// would be simpler, and would mean a crash between a write landing and its row
// landing replays the write — a second tender, a second shipment, a second
// customer email. Claiming first turns that same crash into a step nobody can
// account for, which the model is told about rather than left to repeat.
//
// A run with no ledger, or a call whose arguments cannot be canonicalised,
// falls through to the unguarded path with the provider's call id as the
// idempotency key. That is exactly what every run did before this existed.
func (s *Dispatcher) GuardedDispatch(
	ctx context.Context,
	p *GuardedDispatchParams,
) toolcall.Result {
	req := p.Req
	key := ""
	if req.Steps != nil && !req.StepOwner.ID.IsNil() {
		key = toolcall.StepKey(toolcall.StepKeyParams{
			OwnerID:  req.StepOwner.ID,
			ToolName: p.Call.Name,
			Args:     p.Call.Arguments,
			Ordinal:  p.Ordinal,
			Scope:    req.StepScope(),
		})
	}

	ctx, span := s.startToolSpan(ctx, req, &p.Call, key)
	markCaller(span, p.Caller, p.CallIndex)
	defer span.End()

	if key == "" {
		outcome := s.dispatch(ctx, &DispatchParams{
			req:            req,
			call:           p.Call,
			completionText: p.CompletionText,
			proposedSoFar:  p.ProposedSoFar,
			moneySpent:     p.MoneySpent,
			idempotencyKey: p.Call.ID,
			taint:          p.Taint,
			afterExternal:  p.AfterExternal,
			beyond:         p.Beyond,
			scriptStepKey:  p.ScriptStepKey,
			gate:           p.Gate,
			caller:         p.Caller,
		})
		finishToolSpan(span, &outcome, p.AfterExternal)

		return outcome
	}

	step := agentruntimeports.RunStep{
		OwnerKind: req.StepOwner.Kind,
		OwnerID:   req.StepOwner.ID,
		Attempt:   req.Attempt,
		Kind:      agentruntimeports.RunStepTool,
		Key:       key,
		ToolName:  p.Call.Name,
		CallID:    p.Call.ID,
		Args:      p.Call.Arguments,
	}
	step.TraceID, step.SpanID = aitrace.IDs(ctx)
	if definition := req.Definition; definition != nil {
		version := definition.Version
		step.DefinitionID = definition.ID
		step.DefinitionVersion = &version
	}
	if req.Delegation != nil {
		step.DelegateCallID = req.Delegation.CallID
	}

	outcome := s.claimAndDispatch(ctx, p, &step)
	finishToolSpan(span, &outcome, p.AfterExternal)

	return outcome
}

func (s *Dispatcher) claimAndDispatch(
	ctx context.Context,
	p *GuardedDispatchParams,
	step *agentruntimeports.RunStep,
) toolcall.Result {
	req := p.Req
	key := step.Key
	span := trace.SpanFromContext(ctx)

	verdict, err := req.Steps.Claim(ctx, req.Actor.TenantInfo(), step)
	if err != nil {
		// A ledger that cannot be written cannot protect a write, so the call
		// is refused rather than run unguarded. The alternative — running it
		// anyway — is the duplicate this whole mechanism exists to prevent,
		// and it would happen precisely when the database is already unwell.
		s.logger.Error("a tool call could not be claimed and was not run",
			zap.String("tool", p.Call.Name),
			zap.String("owner", req.StepOwner.ID.String()),
			zap.Error(err),
		)

		return toolcall.Refused(
			aitrace.OutcomeFailed,
			"the run could not record that it was about to make the call",
			"Tool %q was not run: this run could not record that it was about to. "+
				"Tell the person it could not be done right now, and do not retry it.",
			p.Call.Name,
		)
	}

	span.SetAttributes(aitrace.AIStepState.String(stepState(verdict.State)))
	switch verdict.State {
	case agentruntimeports.StepCompleted, agentruntimeports.StepFailed:
		return replayedOutcome(&verdict.Outcome)
	case agentruntimeports.StepUnknown:
		return unknownOutcome(p.Call.Name)
	case agentruntimeports.StepFresh:
	}

	outcome := s.dispatch(ctx, &DispatchParams{
		req:            req,
		call:           p.Call,
		completionText: p.CompletionText,
		proposedSoFar:  p.ProposedSoFar,
		moneySpent:     p.MoneySpent,
		idempotencyKey: key,
		taint:          p.Taint,
		afterExternal:  p.AfterExternal,
		beyond:         p.Beyond,
		stepKey:        key,
		scriptStepKey:  p.ScriptStepKey,
		gate:           p.Gate,
		caller:         p.Caller,
	})

	step.Status = agentruntimeports.RunStepCompleted
	if outcome.Failed {
		step.Status = agentruntimeports.RunStepFailed
	}
	step.Outcome = agentruntimeports.RunStepOutcome{
		Content: outcome.Content,
		Failed:  outcome.Failed,
		Action:  outcome.Action,
		Batch:   outcome.Batch,
		Taint:   outcome.Taint,
		Reason:  outcome.Reason,
		Verdict: outcome.VerdictOrDerived(),
	}
	if !outcome.Failed && showsResults(req) {
		step.Outcome.Data = stepData(outcome.Data)
		step.Outcome.RanWith = outcome.RanWith
	}
	// The settle rides a context that cancellation cannot reach. The failures
	// this ledger exists for — an activity timing out, a person pressing Stop —
	// all cancel the turn's context, and a settle that honoured the
	// cancellation would leave the step Started: the next attempt would then
	// report a write it cannot account for when we know exactly how it went.
	if err = req.Steps.Settle(
		context.WithoutCancel(ctx),
		req.Actor.TenantInfo(),
		step,
	); err != nil {
		s.logger.Error("a tool call ran and its outcome could not be recorded",
			zap.String("tool", p.Call.Name),
			zap.String("owner", req.StepOwner.ID.String()),
			zap.Error(err),
		)
	}

	return outcome
}

// replayedOutcome hands back what an earlier attempt of this run was told.
//
// The action comes back whole, including the version its proposal pinned. It
// must not be re-pinned: the pin records the record as it stood when the
// change was proposed, and taking it again now would silently approve a
// proposal against a shipment that has moved on since.
//
// The result comes back too, when it was kept, so what the original call
// showed beside the conversation is shown again rather than lost with the
// attempt that made it.
func replayedOutcome(recorded *agentruntimeports.RunStepOutcome) toolcall.Result {
	content := recorded.Content
	if content == "" {
		content = "This call was made on an earlier attempt of this run. " +
			"Its answer was not kept, but it was not run again."
	}

	outcome := toolcall.Result{
		Content: content,
		Failed:  recorded.Failed,
		Action:  recorded.Action,
		Batch:   recorded.Batch,
		Taint:   recorded.Taint,
		Verdict: recorded.Verdict,
		Reason:  recorded.Reason,
	}
	// A nil map in an interface is not a nil interface, and an observer
	// reading one would take the call for a query that returned something.
	if len(recorded.Data) > 0 {
		outcome.Data = recorded.Data
		outcome.RanWith = recorded.RanWith
	}

	return outcome
}

// maxStoredStepDataBytes bounds the result a step keeps for showing again.
// It matches the bound on the answer the model is told: a result past it is
// not kept at all, because a table missing its tail would read as complete.
const maxStoredStepDataBytes = 64 * 1024

// showsResults reports whether anything beside the run shows what its tools
// return. Only a conversation does, and only there is a result worth keeping.
func showsResults(req *agentruntimeports.RunRequest) bool {
	return req.ToolObserver != nil || !req.ThreadID.IsNil()
}

// stepData is a query's result in the JSON form the artifacts are built from,
// or nil when there is none, it is not an object, or it is too large to keep.
func stepData(data any) map[string]any {
	if data == nil {
		return nil
	}

	encoded, err := sonic.Marshal(data)
	if err != nil || len(encoded) > maxStoredStepDataBytes {
		return nil
	}

	var decoded map[string]any
	if err = sonic.Unmarshal(encoded, &decoded); err != nil || len(decoded) == 0 {
		return nil
	}

	return decoded
}

// unknownOutcome is what the model is told about a step an earlier attempt
// began and never finished.
//
// It is deliberately not a silent retry. The tool may have written something,
// or may not have; nobody knows, and the one thing worse than telling the
// person that is quietly doing it a second time.
func unknownOutcome(toolName string) toolcall.Result {
	return toolcall.Refused(
		aitrace.OutcomeUnknown,
		"an earlier attempt began it and never recorded how it ended",
		"Tool %q was begun on an earlier attempt of this run and its outcome was never "+
			"recorded, so it was not run again. It may or may not have taken effect. "+
			"Check whether the change is already in place before doing anything that "+
			"depends on it, and tell the person plainly that this one is unconfirmed.",
		toolName,
	)
}

// SeedFromLedger puts back what earlier attempts of this run already learned.
//
// The repeat guard is per-attempt and in memory, so a retry starts blank and
// would re-ask for a call that already failed identically, spending budget to
// re-learn a refusal. Only the refusals are put back.
//
// The ordinals are not. An attempt that starts over asks the model afresh,
// and the model asks for the same writes again: the first dispatch_assign_move of the
// retry is the first dispatch_assign_move of the run, and has to mint the same step
// key so the ledger answers with what already happened. Numbering it after the
// earlier attempt's calls minted a new key, and the write ran a second time.
func (s *Dispatcher) SeedFromLedger(
	ctx context.Context,
	req *agentruntimeports.RunRequest,
	repeats *loopguard.RepeatGuard,
) {
	if req.Steps == nil || req.StepOwner.ID.IsNil() {
		return
	}

	steps, err := req.Steps.Loaded(ctx, req.Actor.TenantInfo(), req.StepOwner)
	if err != nil {
		// Starting blank is where every run stood before the ledger existed.
		// The claim below still protects the writes; only the cheap lessons
		// are lost.
		s.logger.Error("could not read what earlier attempts of this run learned",
			zap.String("owner", req.StepOwner.ID.String()),
			zap.Error(err),
		)

		return
	}
	if len(steps) == 0 {
		return
	}

	replayed := 0
	for idx := range steps {
		step := &steps[idx]
		if step.Kind != agentruntimeports.RunStepTool {
			continue
		}
		replayed++

		if step.Status == agentruntimeports.RunStepFailed && step.Outcome.Content != "" {
			repeats.Record(
				&llmports.ToolCall{Name: step.ToolName, Arguments: step.Args},
				step.Outcome.Content,
			)
		}
	}

	s.logger.Info("this run is continuing from an earlier attempt",
		zap.String("owner", req.StepOwner.ID.String()),
		zap.Int("attempt", req.Attempt),
		zap.Int("toolStepsOnRecord", replayed),
	)
}
