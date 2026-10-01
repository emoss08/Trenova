package aiauditservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/aiaudit"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStepOutcome_FollowsTheVerdict(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		failed  bool
		reason  string
		verdict string
		want    aiaudit.Outcome
	}{
		{
			name:    "denied",
			failed:  true,
			reason:  "lacks worker:update",
			verdict: aitrace.OutcomeDenied,
			want:    aiaudit.OutcomeDenied,
		},
		{
			name:    "invalid arguments",
			failed:  true,
			reason:  "rate: must be positive",
			verdict: aitrace.OutcomeInvalid,
			want:    aiaudit.OutcomeRefused,
		},
		{
			name:    "duplicate proposal",
			reason:  "the same change is already waiting for a decision",
			verdict: aitrace.OutcomeDuplicate,
			want:    aiaudit.OutcomeRefused,
		},
		{
			name:    "over budget",
			failed:  true,
			reason:  "daily cap reached",
			verdict: aitrace.OutcomeOverBudget,
			want:    aiaudit.OutcomeRefused,
		},
		{
			name:    "query failed",
			failed:  true,
			reason:  "connection reset",
			verdict: aitrace.OutcomeFailed,
			want:    aiaudit.OutcomeFailed,
		},
		{
			name:    "ran",
			verdict: aitrace.OutcomeRan,
			want:    aiaudit.OutcomeRan,
		},
		{
			name:    "unrecognised verdict on a failure keeps the reason rule",
			failed:  true,
			reason:  "something",
			verdict: "mystery",
			want:    aiaudit.OutcomeDenied,
		},
		{
			name:   "no verdict with a reason derives as before",
			failed: true,
			reason: "lacks worker:update",
			want:   aiaudit.OutcomeDenied,
		},
		{
			name:   "no verdict without a reason derives as before",
			failed: true,
			want:   aiaudit.OutcomeFailed,
		},
		{
			name: "no verdict and no failure derives as before",
			want: aiaudit.OutcomeRan,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			step := &agent.AgentRunStep{Status: string(serviceports.RunStepCompleted)}
			if tt.failed {
				step.Status = string(serviceports.RunStepFailed)
			}

			got := stepOutcome(step, &serviceports.RunStepOutcome{
				Failed:  tt.failed,
				Reason:  tt.reason,
				Verdict: tt.verdict,
			})

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStepOutcome_AnActionOutranksTheVerdict(t *testing.T) {
	t.Parallel()

	got := stepOutcome(
		&agent.AgentRunStep{Status: string(serviceports.RunStepCompleted)},
		&serviceports.RunStepOutcome{
			Verdict: aitrace.OutcomeProposed,
			Action:  &serviceports.PendingAction{ToolName: "update_worker"},
		},
	)

	assert.Equal(t, aiaudit.OutcomeProposed, got)
}

func TestRunEventEvent_ARefusalFollowsItsVerdict(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		verdict any
		want    aiaudit.Outcome
	}{
		{name: "denied", verdict: aitrace.OutcomeDenied, want: aiaudit.OutcomeDenied},
		{name: "invalid", verdict: aitrace.OutcomeInvalid, want: aiaudit.OutcomeRefused},
		{name: "duplicate", verdict: aitrace.OutcomeDuplicate, want: aiaudit.OutcomeRefused},
		{name: "over budget", verdict: aitrace.OutcomeOverBudget, want: aiaudit.OutcomeRefused},
		{name: "failed", verdict: aitrace.OutcomeFailed, want: aiaudit.OutcomeFailed},
		{name: "no verdict", verdict: nil, want: aiaudit.OutcomeRefused},
		{name: "unrecognised verdict", verdict: "mystery", want: aiaudit.OutcomeRefused},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			payload := map[string]any{
				payloadKeyFailed:  true,
				payloadKeyName:    "delete_everything",
				payloadKeyContent: "There is no tool named \"delete_everything\".",
			}
			if tt.verdict != nil {
				payload[payloadKeyVerdict] = tt.verdict
			}
			row := &agent.AgentRunEvent{
				ID:         pulid.MustNew("are_"),
				OwnerKind:  string(agent.RunOwnerAgentRun),
				OwnerID:    pulid.MustNew("ar_"),
				Kind:       serviceports.AssistantEventToolFinished,
				CallID:     "call_1",
				Payload:    payload,
				OccurredAt: 1_700_000_000,
			}

			d := &deriver{redactor: testRedactor()}
			event := d.runEventEvent(emptyLookups(), row)

			require.NotNil(t, event)
			assert.Equal(t, aiaudit.KindToolRefused, event.Kind)
			assert.Equal(t, tt.want, event.Outcome)
			assert.Equal(t, "delete_everything", event.ToolName)
		})
	}
}

func TestRunEventEvent_ARefusalWithAStepIsTheStepsRow(t *testing.T) {
	t.Parallel()

	owner := pulid.MustNew("ar_")
	lookups := emptyLookups()
	lookups.calls[repositories.OwnerCall{OwnerID: owner, CallID: "call_1"}] = struct{}{}

	d := &deriver{redactor: testRedactor()}
	event := d.runEventEvent(lookups, &agent.AgentRunEvent{
		ID:        pulid.MustNew("are_"),
		OwnerKind: string(agent.RunOwnerAgentRun),
		OwnerID:   owner,
		Kind:      serviceports.AssistantEventToolFinished,
		CallID:    "call_1",
		Payload: map[string]any{
			payloadKeyFailed:  true,
			payloadKeyVerdict: aitrace.OutcomeDenied,
		},
		OccurredAt: 1_700_000_000,
	})

	assert.Nil(t, event)
}
