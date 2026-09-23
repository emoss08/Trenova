package agentruntime

import (
	"context"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"go.uber.org/zap"
)

// ordinals counts how many times each distinct call has been made in this
// attempt, so the step key of a legitimately repeated call differs from the
// first one's.
type ordinals struct {
	seen map[string]int
}

func newOrdinals() *ordinals {
	return &ordinals{seen: make(map[string]int, 4)}
}

// next returns this call's ordinal and counts it.
func (o *ordinals) next(call serviceports.ToolCall) int {
	key, ok := callKey(call)
	if !ok {
		return 0
	}

	ordinal := o.seen[key]
	o.seen[key] = ordinal + 1

	return ordinal
}

// guardedDispatchParams groups what running one guarded call needs.
type guardedDispatchParams struct {
	req            *serviceports.RunRequest
	call           serviceports.ToolCall
	completionText string
	proposedSoFar  []serviceports.PendingAction
	// ordinal numbers this exact call within the run; the loop assigns it.
	ordinal int
}

// guardedDispatch runs one tool call at most once across every attempt of a
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
func (s *Service) guardedDispatch(
	ctx context.Context,
	p guardedDispatchParams,
) toolOutcome {
	req := p.req
	key := ""
	if req.Steps != nil && !req.StepOwner.ID.IsNil() {
		key = StepKey(StepKeyParams{
			OwnerID:  req.StepOwner.ID,
			ToolName: p.call.Name,
			Args:     p.call.Arguments,
			Ordinal:  p.ordinal,
		})
	}

	if key == "" {
		return s.dispatch(ctx, dispatchParams{
			req:            req,
			call:           p.call,
			completionText: p.completionText,
			proposedSoFar:  p.proposedSoFar,
			idempotencyKey: p.call.ID,
		})
	}

	step := serviceports.RunStep{
		OwnerKind: req.StepOwner.Kind,
		OwnerID:   req.StepOwner.ID,
		Attempt:   req.Attempt,
		Kind:      serviceports.RunStepTool,
		Key:       key,
		ToolName:  p.call.Name,
		CallID:    p.call.ID,
		Args:      p.call.Arguments,
	}

	verdict, err := req.Steps.Claim(ctx, req.Actor.TenantInfo(), step)
	if err != nil {
		// A ledger that cannot be written cannot protect a write, so the call
		// is refused rather than run unguarded. The alternative — running it
		// anyway — is the duplicate this whole mechanism exists to prevent,
		// and it would happen precisely when the database is already unwell.
		s.logger.Error("a tool call could not be claimed and was not run",
			zap.String("tool", p.call.Name),
			zap.String("owner", req.StepOwner.ID.String()),
			zap.Error(err),
		)

		return failedOutcome(
			"Tool %q was not run: this run could not record that it was about to. "+
				"Tell the person it could not be done right now, and do not retry it.",
			p.call.Name,
		)
	}

	switch verdict.State {
	case serviceports.StepCompleted, serviceports.StepFailed:
		return replayedOutcome(verdict.Outcome)
	case serviceports.StepUnknown:
		return unknownOutcome(p.call.Name)
	}

	outcome := s.dispatch(ctx, dispatchParams{
		req:            req,
		call:           p.call,
		completionText: p.completionText,
		proposedSoFar:  p.proposedSoFar,
		idempotencyKey: key,
	})

	step.Status = serviceports.RunStepCompleted
	if outcome.failed {
		step.Status = serviceports.RunStepFailed
	}
	step.Outcome = serviceports.RunStepOutcome{
		Content: outcome.content,
		Failed:  outcome.failed,
		Action:  outcome.action,
	}
	// The settle rides a context that cancellation cannot reach. The failures
	// this ledger exists for — an activity timing out, a person pressing Stop —
	// all cancel the turn's context, and a settle that honoured the
	// cancellation would leave the step Started: the next attempt would then
	// report a write it cannot account for when we know exactly how it went.
	if err = req.Steps.Settle(context.WithoutCancel(ctx), req.Actor.TenantInfo(), step); err != nil {
		s.logger.Error("a tool call ran and its outcome could not be recorded",
			zap.String("tool", p.call.Name),
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
func replayedOutcome(recorded serviceports.RunStepOutcome) toolOutcome {
	content := recorded.Content
	if content == "" {
		content = "This call was made on an earlier attempt of this run. " +
			"Its answer was not kept, but it was not run again."
	}

	return toolOutcome{content: content, failed: recorded.Failed, action: recorded.Action}
}

// unknownOutcome is what the model is told about a step an earlier attempt
// began and never finished.
//
// It is deliberately not a silent retry. The tool may have written something,
// or may not have; nobody knows, and the one thing worse than telling the
// person that is quietly doing it a second time.
func unknownOutcome(toolName string) toolOutcome {
	return failedOutcome(
		"Tool %q was begun on an earlier attempt of this run and its outcome was never "+
			"recorded, so it was not run again. It may or may not have taken effect. "+
			"Check whether the change is already in place before doing anything that "+
			"depends on it, and tell the person plainly that this one is unconfirmed.",
		toolName,
	)
}

// seedFromLedger puts back what earlier attempts of this run already learned.
//
// The repeat guard is per-attempt and in memory, so a retry starts blank and
// would re-ask for a call that already failed identically, spending budget to
// re-learn a refusal. Only the refusals are put back.
//
// The ordinals are not. An attempt that starts over asks the model afresh,
// and the model asks for the same writes again: the first assign_move of the
// retry is the first assign_move of the run, and has to mint the same step
// key so the ledger answers with what already happened. Numbering it after the
// earlier attempt's calls minted a new key, and the write ran a second time.
func (s *Service) seedFromLedger(
	ctx context.Context,
	req *serviceports.RunRequest,
	repeats *repeatGuard,
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
	for _, step := range steps {
		if step.Kind != serviceports.RunStepTool {
			continue
		}
		replayed++

		if step.Status == serviceports.RunStepFailed && step.Outcome.Content != "" {
			repeats.record(
				serviceports.ToolCall{Name: step.ToolName, Arguments: step.Args},
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
