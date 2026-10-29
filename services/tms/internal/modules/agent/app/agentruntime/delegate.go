package agentruntime

import (
	"maps"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/delegation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

// handingEffects is what handing tasks does outside the turn, through the
// turn's effects.
type handingEffects struct {
	s  *Service
	t  *Turn
	fx TurnEffects
}

func (e handingEffects) Emit(event kernelports.StreamEvent) { e.fx.Emit(event) }

func (e handingEffects) Delegate(calls []delegation.Call) []delegation.Run {
	return e.fx.Delegate(e.t, calls)
}

func (e handingEffects) ToolStarted(call *llmports.ToolCall) { e.s.emitToolStarted(e.fx, call) }

func (s *Service) handing(t *Turn, fx TurnEffects) handingEffects {
	return handingEffects{s: s, t: t, fx: fx}
}

// Delegations is how many tasks the turn has handed out.
func (t *Turn) Delegations() int { return t.delegations }

// CountDelegation counts one more task handed out.
func (t *Turn) CountDelegation() { t.delegations++ }

// NextOrdinal numbers this exact call within the run.
func (t *Turn) NextOrdinal(call *llmports.ToolCall) int { return t.counts.Next(call) }

// CallIDs are the tool call ids the conversation holds, sorted, because they
// are read in workflow code and a map ranges in a different order every time.
func (t *Turn) CallIDs() []string { return slices.Sorted(maps.Keys(t.callIDs)) }

// CarryExternalContent starts a delegate's turn where the turn that handed it
// the task stands on outside content.
func (t *Turn) CarryExternalContent(external bool) {
	t.external = t.external || external
}

// ReadExternalContent reports whether the turn has read content from outside
// the organization.
func (t *Turn) ReadExternalContent() bool { return t.external }

// ReserveCallIDs marks tool call ids as taken, so a provider that reuses one
// is given a fresh id rather than a clash. A delegate's turn is handed the
// conversation's ids when it opens, and the conversation takes the delegate's
// when it ends: both keep their calls in one thread.
func (t *Turn) ReserveCallIDs(ids []string) {
	for _, id := range ids {
		if id != "" {
			t.callIDs[id] = struct{}{}
		}
	}
}
