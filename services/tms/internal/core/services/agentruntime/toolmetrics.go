package agentruntime

// unregisteredToolLabel is the tool a call is counted under when it names
// nothing the runtime knows. The name comes from the model, and a label taken
// from it as sent would let one confused model mint a series per invented tool.
const unregisteredToolLabel = "unregistered"

// ReplayAware is implemented by effects that can re-run the loop over a
// recorded history. Workflow code re-executes the loop whenever a worker
// rebuilds an execution it no longer holds, and a counter bumped on every
// pass would count each call once per rebuild.
type ReplayAware interface {
	Replaying() bool
}

// countToolOutcome counts one finished call by tool and verdict. Every call
// the loop answers comes through recordToolResult, refusals included, so this
// is the one place a call is counted. A call with no verdict of its own, a
// find_tools answer or a repeated read, is counted as the verdict its result
// implies.
func (s *Service) countToolOutcome(fx TurnEffects, name string, outcome *toolOutcome) {
	if aware, ok := fx.(ReplayAware); ok && aware.Replaying() {
		return
	}

	s.metrics.RecordToolOutcome(s.toolMetricLabel(name), outcome.verdictOrDerived())
}

func (s *Service) toolMetricLabel(name string) string {
	if s.toolNamed(name) != nil {
		return name
	}
	if _, runtime := runtimePolicyNamed(name); runtime {
		return name
	}

	return unregisteredToolLabel
}
