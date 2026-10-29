package replystep

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/modelstream"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type keptSteps struct {
	steps   map[string]agentruntimeports.RunStep
	tenants []pagination.TenantInfo
	findErr error
}

func newKeptSteps() *keptSteps {
	return &keptSteps{steps: map[string]agentruntimeports.RunStep{}}
}

func (*keptSteps) Claim(
	context.Context,
	pagination.TenantInfo,
	*agentruntimeports.RunStep,
) (agentruntimeports.StepVerdict, error) {
	return agentruntimeports.StepVerdict{State: agentruntimeports.StepFresh}, nil
}

func (*keptSteps) Settle(
	context.Context,
	pagination.TenantInfo,
	*agentruntimeports.RunStep,
) error {
	return nil
}

func (k *keptSteps) Record(
	_ context.Context,
	tenant pagination.TenantInfo,
	step *agentruntimeports.RunStep,
) error {
	k.tenants = append(k.tenants, tenant)
	if _, held := k.steps[step.Key]; !held {
		k.steps[step.Key] = *step
	}

	return nil
}

func (k *keptSteps) Find(
	_ context.Context,
	tenant pagination.TenantInfo,
	_ agentruntimeports.RunStepOwner,
	key string,
) (agentruntimeports.RunStep, bool, error) {
	k.tenants = append(k.tenants, tenant)
	if k.findErr != nil {
		return agentruntimeports.RunStep{}, false, k.findErr
	}
	step, found := k.steps[key]

	return step, found, nil
}

func (*keptSteps) Loaded(
	context.Context,
	pagination.TenantInfo,
	agentruntimeports.RunStepOwner,
) ([]agentruntimeports.RunStep, error) {
	return nil, nil
}

func call() *Call {
	return &Call{
		Owner: agentruntimeports.RunStepOwner{
			Kind: agentruntimeports.RunStepOwnerAssistantTurn,
			ID:   pulid.MustNew("atrn_"),
		},
		Ordinal: 1,
	}
}

func request(question string) *llmports.ChatCompletionRequest {
	return &llmports.ChatCompletionRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		System:   "Help dispatch.",
		Messages: llmports.UserMessage(question),
		Tools: []kernelports.ToolSpec{{
			Name:       "shipment_get",
			Parameters: map[string]any{"type": "object", "required": []any{"proNumber"}},
		}},
	}
}

func TestKey_NamesOneCallOfOneRun(t *testing.T) {
	t.Parallel()

	base := call()
	req := request("Where is load 12345?")
	key := Key(base, req)

	require.True(t, strings.HasPrefix(key, keyPrefix))
	assert.LessOrEqual(t, len(key), 120, "the key fits the step_key column")
	assert.Equal(t, key, Key(base, req), "the same call asks under the same key")

	again := *base
	again.Ordinal = 2
	assert.NotEqual(t, key, Key(&again, req), "a second call of the turn is its own step")

	scoped := *base
	scoped.Scope = "delegate:abc"
	assert.NotEqual(t, key, Key(&scoped, req), "a delegate's call is apart from the owner's")

	other := *base
	other.Owner.ID = pulid.MustNew("atrn_")
	assert.NotEqual(t, key, Key(&other, req), "another run's call is another step")

	changed := *req
	changed.Messages = llmports.UserMessage("Where is load 67890?")
	assert.NotEqual(t, key, Key(base, &changed), "a different question is a different step")
}

func TestKey_IsEmptyForACallNoLedgerKeeps(t *testing.T) {
	t.Parallel()

	req := request("Where is load 12345?")
	assert.Empty(t, Key(nil, req))
	assert.Empty(t, Key(&Call{Ordinal: 1}, req))

	unnumbered := call()
	unnumbered.Ordinal = 0
	assert.Empty(t, Key(unnumbered, req))
	assert.Empty(t, Key(call(), nil))
}

func TestKeepThenRecall_HandsBackTheSameReply(t *testing.T) {
	t.Parallel()

	steps := newKeptSteps()
	ledger := NewLedger(steps, zap.NewNop())
	req := request("Where is load 12345?")
	lookup := &Lookup{Call: call(), Request: req}
	lookup.Key = Key(lookup.Call, req)
	reply := modelstream.Reply{
		Completion: &llmports.ChatCompletionResult{
			Text:            "Load 12345 is in Memphis.",
			ModelIdentifier: "test-model",
		},
		Usage: &agentruntimeports.RunUsage{ModelCalls: 1, InputTokens: 900},
	}

	_, found := ledger.Recall(t.Context(), lookup)
	require.False(t, found, "nothing is kept before the first reply")

	ledger.Keep(t.Context(), &Kept{Lookup: lookup, Attempt: 2, Reply: &reply})

	kept, ok := steps.steps[lookup.Key]
	require.True(t, ok)
	assert.Equal(t, agentruntimeports.RunStepCompletion, kept.Kind)
	assert.Equal(t, agentruntimeports.RunStepCompleted, kept.Status)
	assert.Equal(t, 2, kept.Attempt)
	assert.Equal(t, lookup.Call.Owner.ID, kept.OwnerID)

	recalled, found := ledger.Recall(t.Context(), lookup)
	require.True(t, found)
	assert.Equal(t, reply, recalled)
	for _, tenant := range steps.tenants {
		assert.Equal(t, req.TenantInfo, tenant, "every read and write is the request's tenant")
	}
}

func TestKeep_LeavesALoopedReplyOut(t *testing.T) {
	t.Parallel()

	steps := newKeptSteps()
	ledger := NewLedger(steps, zap.NewNop())
	req := request("Where is load 12345?")
	lookup := &Lookup{Call: call(), Request: req}
	lookup.Key = Key(lookup.Call, req)

	ledger.Keep(t.Context(), &Kept{
		Lookup:  lookup,
		Attempt: 1,
		Reply: &modelstream.Reply{
			Completion: &llmports.ChatCompletionResult{Text: "time-time-time-"},
			Looped:     true,
		},
	})

	assert.Empty(t, steps.steps, "a reply cut off for looping was never an answer")
}

func TestRecall_AsksTheModelWhenTheLedgerCannotBeRead(t *testing.T) {
	t.Parallel()

	steps := newKeptSteps()
	steps.findErr = errors.New("connection reset")
	ledger := NewLedger(steps, zap.NewNop())
	req := request("Where is load 12345?")
	lookup := &Lookup{Call: call(), Request: req}
	lookup.Key = Key(lookup.Call, req)

	_, found := ledger.Recall(t.Context(), lookup)

	assert.False(t, found, "asking the model again is always safe")
}

func TestReplayable_NeedsAWholeFinishedReply(t *testing.T) {
	t.Parallel()

	reply := &agentruntimeports.RunStepReply{
		Completion: &llmports.ChatCompletionResult{Text: "Load 12345 is in Memphis."},
	}
	tests := []struct {
		name string
		step agentruntimeports.RunStep
		want bool
	}{
		{
			name: "a kept reply",
			step: agentruntimeports.RunStep{
				Kind:    agentruntimeports.RunStepCompletion,
				Status:  agentruntimeports.RunStepCompleted,
				Outcome: agentruntimeports.RunStepOutcome{Reply: reply},
			},
			want: true,
		},
		{
			name: "a reply too large to keep",
			step: agentruntimeports.RunStep{
				Kind:   agentruntimeports.RunStepCompletion,
				Status: agentruntimeports.RunStepCompleted,
			},
		},
		{
			name: "a tool call",
			step: agentruntimeports.RunStep{
				Kind:    agentruntimeports.RunStepTool,
				Status:  agentruntimeports.RunStepCompleted,
				Outcome: agentruntimeports.RunStepOutcome{Reply: reply},
			},
		},
		{
			name: "a step never finished",
			step: agentruntimeports.RunStep{
				Kind:    agentruntimeports.RunStepCompletion,
				Status:  agentruntimeports.RunStepStarted,
				Outcome: agentruntimeports.RunStepOutcome{Reply: reply},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, Replayable(&tt.step))
		})
	}
}
