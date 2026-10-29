package agentruntime

import (
	"fmt"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/delegation"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func callTurn(id, name string, args map[string]any) *llmports.ChatCompletionResult {
	return &llmports.ChatCompletionResult{
		ToolCalls:       []llmports.ToolCall{{ID: id, Name: name, Arguments: args}},
		ModelIdentifier: "test-model",
	}
}

type handOffParams struct {
	lookup   *agenttest.StubQueryTool
	delegate agentturn.RuntimeDelegate
	args     map[string]any
}

// handOff drives a turn that reads a shipment with shipment_get (call_read)
// and then hands a task over with the given arguments (call_hand).
func handOff(t *testing.T, p handOffParams) (*agentruntimeports.RunResult, *delegateRecorder) {
	t.Helper()

	lookup := p.lookup
	if lookup == nil {
		lookup = &agenttest.StubQueryTool{
			ToolName: "shipment_get",
			Result:   map[string]any{"proNumber": "PRO-1001", "bol": "BOL-77"},
		}
	}
	args := map[string]any{"agentId": p.delegate.ID.String(), "task": "Copy it."}
	for key, value := range p.args {
		args[key] = value
	}
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		callTurn("call_read", lookup.ToolName, map[string]any{"shipmentId": "shp_1"}),
		callTurn("call_hand", runtimetools.DelegateTaskName, args),
		runtimetest.TextTurn("Done."),
	}}
	rt := newRuntime(completion,
		&stubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{lookup}},
		&stubActionRegistry{}, nil)
	req := delegatingRequest(p.delegate)
	req.Definition = runtimetest.Definition(lookup.ToolName)
	fx := &delegateRecorder{run: scriptedDelegateRun(p.delegate)}

	return driveWith(t, rt, req, fx), fx
}

func handOffResult(result *agentruntimeports.RunResult) *conversation.Message {
	for idx := range result.Messages {
		message := &result.Messages[idx]
		if message.Role == conversation.RoleTool && message.ToolCallID == "call_hand" {
			return message
		}
	}

	return nil
}

/*
The Dispatch desk read a shipment and asked the Shipment Desk to copy it in
prose; the Shipment Desk read it again and retyped it wrong. Now the task
carries the record and the result the asking agent already read, as data.
*/
func TestDelegate_HandsOverTheRecordsAndTheResultsItNames(t *testing.T) {
	t.Parallel()

	delegate := runtimetest.ShipmentDesk()
	result, fx := handOff(t, handOffParams{
		delegate: delegate,
		args: map[string]any{
			"records":      []any{map[string]any{"entityType": "shipment", "id": "shp_1"}},
			"shareResults": []any{"call_read"},
		},
	})

	require.Len(t, fx.calls, 1)
	handed := fx.calls[0].Context
	require.NotNil(t, handed)
	assert.Equal(t, []agentrecord.RecordRef{{EntityType: "shipment", ID: "shp_1"}}, handed.Records)
	require.Len(t, handed.Results, 1)
	assert.Equal(t, "call_read", handed.Results[0].CallID)
	assert.Equal(t, "shipment_get", handed.Results[0].ToolName)
	assert.Contains(t, handed.Results[0].Content, "PRO-1001")
	assert.NotContains(t, handed.Results[0].Content, toolresult.UntrustedOpenTag,
		"the result is handed over as the tool returned it, not in the model's fence")
	assert.False(t, handOffResult(result).ToolFailed)
}

// A hand-off that names nothing carries nothing, so the call is what it was
// before records existed.
func TestDelegate_AHandOffThatNamesNothingCarriesNoContext(t *testing.T) {
	t.Parallel()

	delegate := runtimetest.ShipmentDesk()
	_, fx := handOff(t, handOffParams{delegate: delegate})

	require.Len(t, fx.calls, 1)
	assert.Nil(t, fx.calls[0].Context)
}

// What cannot be handed over is refused with the reason, before anybody is
// asked, and the call does not count as a task handed out.
func TestDelegate_RefusesWhatItCannotHandOver(t *testing.T) {
	t.Parallel()

	tooMany := make([]any, 0, runtimetools.MaxDelegateRecords+1)
	for idx := range runtimetools.MaxDelegateRecords + 1 {
		tooMany = append(tooMany, map[string]any{
			"entityType": "shipment", "id": fmt.Sprintf("shp_%d", idx),
		})
	}
	big := strings.Repeat("x", delegation.MaxSharedResultBytes+1)

	for name, tc := range map[string]struct {
		args   map[string]any
		lookup *agenttest.StubQueryTool
		want   string
	}{
		"a record kind the registry has not": {
			args: map[string]any{"records": []any{
				map[string]any{"entityType": "spaceship", "id": "shp_1"},
			}},
			want: "records[0].entityType",
		},
		"a record without an id": {
			args: map[string]any{"records": []any{map[string]any{"entityType": "shipment"}}},
			want: "records[0]",
		},
		"an id that is not one": {
			args: map[string]any{"records": []any{
				map[string]any{"entityType": "shipment", "id": "shp_1 and void the rest"},
			}},
			want: "records[0].id",
		},
		"more records than a task carries": {
			args: map[string]any{"records": tooMany},
			want: "records",
		},
		"an argument the tool does not take": {
			args: map[string]any{"priority": "high"},
			want: "priority",
		},
		"a call this turn did not make": {
			args: map[string]any{"shareResults": []any{"call_elsewhere"}},
			want: "call_elsewhere",
		},
		"more results than a task carries": {
			args: map[string]any{"shareResults": []any{"a", "b", "c", "d", "e"}},
			want: "shareResults",
		},
		"a result too long to hand over": {
			args: map[string]any{"shareResults": []any{"call_read"}},
			lookup: &agenttest.StubQueryTool{
				ToolName: "shipment_get",
				Result:   map[string]any{"notes": big},
			},
			want: "KiB",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			delegate := runtimetest.ShipmentDesk()
			result, fx := handOff(t, handOffParams{
				delegate: delegate, args: tc.args, lookup: tc.lookup,
			})

			assert.Empty(t, fx.calls, "nobody was asked")
			answer := handOffResult(result)
			require.NotNil(t, answer)
			assert.True(t, answer.ToolFailed)
			assert.Contains(t, answer.Content, tc.want)
			assert.Nil(t, answer.DelegateReport)
			assert.Empty(t, result.Delegations)
		})
	}
}

// A failed call left no result to hand over, and saying so beats handing the
// other agent an error as if it were data.
func TestDelegate_RefusesToShareAFailedCall(t *testing.T) {
	t.Parallel()

	delegate := runtimetest.ShipmentDesk()
	result, fx := handOff(t, handOffParams{
		delegate: delegate,
		args:     map[string]any{"shareResults": []any{"call_read"}},
		lookup: &agenttest.StubQueryTool{
			ToolName: "shipment_get",
			Err:      fmt.Errorf("shipment not found"),
		},
	})

	assert.Empty(t, fx.calls)
	answer := handOffResult(result)
	require.NotNil(t, answer)
	assert.True(t, answer.ToolFailed)
	assert.Contains(t, answer.Content, "call_read")
	assert.Contains(t, answer.Content, "failed")
}

// The parent learns the proposal id of each write the delegate filed, so it
// can find the card rather than search for what was made, and is told never
// to show the person that id.
func TestDelegateReport_NamesEachWritesProposal(t *testing.T) {
	t.Parallel()

	delegate := runtimetest.ReportBuilder()
	run := scriptedDelegateRun(delegate)
	made, waiting := pulid.MustNew("ap_"), pulid.MustNew("ap_")
	run.Result.Actions[0].ProposalID = made
	run.Result.Actions[1].ProposalID = waiting

	report := delegation.Report(&delegate, "call_1", &run)

	require.Len(t, report.Made, 1)
	assert.Equal(t, made, report.Made[0].ProposalID)
	require.Len(t, report.Awaiting, 1)
	assert.Equal(t, waiting, report.Awaiting[0].ProposalID)
	assert.Equal(t, waiting, report.Bounded().Awaiting[0].ProposalID)
	assert.Contains(t, delegation.Note(&report, false), "never show them a proposalId")
}
