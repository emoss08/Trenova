package agentruntime

import (
	"maps"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
)

// maxConcurrentReads bounds how many reads one batch runs at once, so a model
// that asks for dozens of lookups in one reply does not open as many queries
// against the database together.
const maxConcurrentReads = 8

// readBatch is the consecutive reads of one completion waiting to run
// together. Each was checked as it joined exactly as the loop checks any call
// it is about to dispatch, so running them is all that is left.
type readBatch struct {
	calls []DispatchCall
	keys  []string
}

func (b *readBatch) size() int { return len(b.calls) }

// holdsCallTo reports a get_ tool the batch already calls. The card a get_
// call shows is bunched with the turn's earlier cards of the same tool, read
// back by their call ids, so two of them must not run at once.
func (b *readBatch) holdsCallTo(name string) bool {
	return slices.ContainsFunc(b.calls, func(call DispatchCall) bool {
		return call.Call.Name == name
	})
}

// concurrentReads are the held tools that only look something up and may run
// beside each other: registered query tools whose effect is a lookup. It is
// taken when the turn opens, because in workflow code the decision has to
// replay the same way whatever a later release's catalog says.
func (s *Service) concurrentReads(held []string) map[string]struct{} {
	reads := make(map[string]struct{}, len(held))
	for _, name := range held {
		tool, ok := s.queryTools.Get(name)
		if !ok || agenttoolports.EffectOf(tool) != agent.ToolEffectLookup {
			continue
		}
		reads[name] = struct{}{}
	}

	return reads
}

func readsState(reads map[string]struct{}) []string {
	if len(reads) == 0 {
		return nil
	}

	return slices.Sorted(maps.Keys(reads))
}

func restoreReads(names []string) map[string]struct{} {
	if len(names) == 0 {
		return nil
	}
	reads := make(map[string]struct{}, len(names))
	for _, name := range names {
		reads[name] = struct{}{}
	}

	return reads
}

// joinCheck is what the loop knows when it weighs a call against the batch.
type joinCheck struct {
	call   llmports.ToolCall
	used   int
	budget int
	stuck  bool
}

// joins reports whether a call can run beside the reads already waiting. It
// holds the call to every check the loop would make before dispatching it,
// with the waiting reads counted as already spent, so a call that joins is one
// the loop would have run one at a time. Anything else (a write, a refusal, a
// repeat of a waiting read, a second fetch of a kind of card) settles the batch
// first and then takes the loop's ordinary path.
func (t *Turn) joins(batch *readBatch, check *joinCheck) (string, bool) {
	call := check.call
	if t.reads == nil || call.ArgumentsError != "" || check.stuck {
		return "", false
	}
	if _, read := t.reads[call.Name]; !read || !t.holds(call.Name) {
		return "", false
	}
	if check.used+batch.size() >= check.budget || batch.size() >= maxConcurrentReads {
		return "", false
	}
	key, ok := callKey(call)
	if !ok || slices.Contains(batch.keys, key) {
		return "", false
	}
	if _, failed := t.repeats.seen(call); failed || t.repeats.readBefore(call) {
		return "", false
	}
	if isFetch(call.Name) && batch.holdsCallTo(call.Name) {
		return "", false
	}
	return key, true
}

// settleReads runs the waiting reads, together when there is more than one,
// and settles each in the order the model asked for them, so what the model,
// the stream and the transcript see is the order of the calls.
func (s *Service) settleReads(t *Turn, fx TurnEffects, batch *readBatch) {
	switch batch.size() {
	case 0:
		return
	case 1:
		outcome := fx.Dispatch(t, batch.calls[0])
		s.settleDispatched(t, fx, &batch.calls[0], &outcome)
	default:
		outcomes := fx.DispatchReads(t, batch.calls)
		for idx := range batch.calls {
			s.settleDispatched(t, fx, &batch.calls[idx], &outcomes[idx])
		}
	}

	batch.calls = batch.calls[:0]
	batch.keys = batch.keys[:0]
}
