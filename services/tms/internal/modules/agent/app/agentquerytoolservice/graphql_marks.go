package agentquerytoolservice

import (
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
)

// markedOutcome is a list's outcome with the records in it that carry text
// written outside the organization, which a Marked read taints the run with
// (a carrier invoice match a carrier sent over EDI). It encodes as the
// outcome it wraps.
type markedOutcome struct {
	outcome any
	outside []agent.RecordRef
}

func (o *markedOutcome) MarshalJSON() ([]byte, error) { return sonic.Marshal(o.outcome) }

func (o *markedOutcome) TaintedRecords() []agent.RecordRef { return o.outside }

// withMarks is a list's outcome carrying the records its spec marks, or the
// outcome itself for a spec that marks none.
func withMarks(spec *graphQLListSpec, items []map[string]any, outcome any) any {
	if spec.marks == nil {
		return outcome
	}
	marked := &markedOutcome{outcome: outcome}
	for _, item := range items {
		if ref, outside := spec.marks(item); outside {
			marked.outside = append(marked.outside, ref)
		}
	}

	return marked
}
