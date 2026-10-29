package modeladaptertest

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aireasoning"
)

// streamWithReasoning is streamWith with the thinking captured too.
func StreamWithReasoning(
	t *testing.T,
	adapter modeladapter.Adapter,
	call *modeladapter.Call,
) (*modeladapter.Response, []string, []string) {
	t.Helper()

	thoughts := []string{}
	call.Reasoning = func(delta string) { thoughts = append(thoughts, delta) }
	resp, deltas := StreamWith(t, adapter, call)

	return resp, deltas, thoughts
}

func ReasoningCall(
	kind aiprovider.Kind,
	url string,
	effort aireasoning.ReasoningEffort,
	req *modeladapter.Request,
) *modeladapter.Call {
	call := CallFor(kind, url, req)
	call.Provider.ReasoningEffort = effort

	return call
}
