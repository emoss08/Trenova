package tooldispatch

import (
	"maps"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolcall"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
)

// MaxConcurrentReads bounds how many reads one batch runs at once, so a model
// that asks for dozens of lookups in one reply does not open as many queries
// against the database together.
const MaxConcurrentReads = 8

// ReadBatch is the consecutive reads of one completion waiting to run
// together. Each was checked as it joined exactly as the loop checks any call
// it is about to dispatch, so running them is all that is left.
type ReadBatch struct {
	Calls []toolcall.Call
	Keys  []string
}

func (b *ReadBatch) Size() int { return len(b.Calls) }

// HoldsCallTo reports a get_ tool the batch already calls. The card a get_
// call shows is bunched with the turn's earlier cards of the same tool, read
// back by their call ids, so two of them must not run at once.
func (b *ReadBatch) HoldsCallTo(name string) bool {
	return slices.ContainsFunc(b.Calls, func(call toolcall.Call) bool {
		return call.Call.Name == name
	})
}

// ConcurrentReads are the held tools that only look something up and may run
// beside each other: registered query tools whose effect is a lookup. It is
// taken when the turn opens, because in workflow code the decision has to
// replay the same way whatever a later release's catalog says.
func (s *Dispatcher) ConcurrentReads(held []string) map[string]struct{} {
	reads := make(map[string]struct{}, len(held))
	for _, name := range held {
		tool, ok := s.queryTools.Get(name)
		if !ok || agenttoolports.EffectOf(tool) != agenttooltrust.ToolEffectLookup {
			continue
		}
		reads[name] = struct{}{}
	}

	return reads
}

func ReadsState(reads map[string]struct{}) []string {
	if len(reads) == 0 {
		return nil
	}

	return slices.Sorted(maps.Keys(reads))
}

func RestoreReads(names []string) map[string]struct{} {
	if len(names) == 0 {
		return nil
	}
	reads := make(map[string]struct{}, len(names))
	for _, name := range names {
		reads[name] = struct{}{}
	}

	return reads
}

// JoinCheck is what the loop knows when it weighs a call against the batch.
type JoinCheck struct {
	Call   llmports.ToolCall
	Used   int
	Budget int
	Stuck  bool
}
