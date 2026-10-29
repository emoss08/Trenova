package agentruntime

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/loopguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/tooldispatch"
)

// joins reports whether a call can run beside the reads already waiting. It
// holds the call to every check the loop would make before dispatching it,
// with the waiting reads counted as already spent, so a call that joins is one
// the loop would have run one at a time. Anything else (a write, a refusal, a
// repeat of a waiting read, a second fetch of a kind of card) settles the batch
// first and then takes the loop's ordinary path.
func (t *Turn) joins(batch *tooldispatch.ReadBatch, check *tooldispatch.JoinCheck) (string, bool) {
	call := check.Call
	if t.reads == nil || call.ArgumentsError != "" || check.Stuck {
		return "", false
	}
	if _, read := t.reads[call.Name]; !read || !t.Holds(call.Name) {
		return "", false
	}
	if check.Used+batch.Size() >= check.Budget || batch.Size() >= tooldispatch.MaxConcurrentReads {
		return "", false
	}
	key, ok := loopguard.CallKey(call)
	if !ok || slices.Contains(batch.Keys, key) {
		return "", false
	}
	if _, failed := t.repeats.Seen(call); failed || t.repeats.ReadBefore(call) {
		return "", false
	}
	if loopguard.IsFetch(call.Name) && batch.HoldsCallTo(call.Name) {
		return "", false
	}
	return key, true
}

// settleReads runs the waiting reads, together when there is more than one,
// and settles each in the order the model asked for them, so what the model,
// the stream and the transcript see is the order of the calls.
func (s *Service) settleReads(t *Turn, fx TurnEffects, batch *tooldispatch.ReadBatch) {
	switch batch.Size() {
	case 0:
		return
	case 1:
		outcome := fx.Dispatch(t, batch.Calls[0])
		s.settleDispatched(t, fx, &batch.Calls[0], &outcome)
	default:
		outcomes := fx.DispatchReads(t, batch.Calls)
		for idx := range batch.Calls {
			s.settleDispatched(t, fx, &batch.Calls[idx], &outcomes[idx])
		}
	}

	batch.Calls = batch.Calls[:0]
	batch.Keys = batch.Keys[:0]
}
