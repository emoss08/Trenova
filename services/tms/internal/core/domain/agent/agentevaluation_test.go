package agent

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func evaluationForTest() *Evaluation {
	return &Evaluation{
		OrganizationID:    pulid.MustNew("org_"),
		BusinessUnitID:    pulid.MustNew("bu_"),
		AgentDefinitionID: pulid.MustNew("agd_"),
		Status:            EvaluationStatusPending,
		Trigger:           RunTriggerChat,
	}
}

func TestEvaluationValidate_RequiresExactlyOneSource(t *testing.T) {
	t.Parallel()

	caseID := pulid.MustNew("aec_")
	tests := []struct {
		name    string
		mutate  func(*Evaluation)
		invalid bool
	}{
		{name: "neither", mutate: func(*Evaluation) {}, invalid: true},
		{
			name: "run with subject",
			mutate: func(e *Evaluation) {
				e.SourceRunID = pulid.MustNew("ar_")
				e.SubjectType = SubjectAssistantThread
				e.SubjectID = pulid.MustNew("athr_")
			},
		},
		{
			name: "case without subject",
			mutate: func(e *Evaluation) {
				e.EvalCaseID = &caseID
			},
		},
		{
			name: "both",
			mutate: func(e *Evaluation) {
				e.SourceRunID = pulid.MustNew("ar_")
				e.SubjectType = SubjectAssistantThread
				e.SubjectID = pulid.MustNew("athr_")
				e.EvalCaseID = &caseID
			},
			invalid: true,
		},
		{
			name: "run without subject",
			mutate: func(e *Evaluation) {
				e.SourceRunID = pulid.MustNew("ar_")
			},
			invalid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			evaluation := evaluationForTest()
			tt.mutate(evaluation)
			multiErr := errortypes.NewMultiError()
			evaluation.Validate(multiErr)

			assert.Equal(t, tt.invalid, multiErr.HasErrors())
		})
	}
}

func TestEvaluationSkip_IsTerminalWithItsReason(t *testing.T) {
	t.Parallel()

	evaluation := evaluationForTest()
	evaluation.Skip("The person who asked no longer has an active account", 1_700_000_000)

	assert.Equal(t, EvaluationStatusSkipped, evaluation.Status)
	assert.True(t, evaluation.Status.Terminal())
	assert.Equal(t, "The person who asked no longer has an active account", evaluation.ErrorMessage)
	assert.Equal(t, int64(1_700_000_000), *evaluation.CompletedAt)
}

func TestCaseChecksFailedHard_ListsOnlyApplicableFailures(t *testing.T) {
	t.Parallel()

	checks := &CaseChecks{Checks: []CaseCheck{
		{Name: "held_tools", Kind: CheckKindHard, Applies: true, Passed: false},
		{Name: "forbidden_tools", Kind: CheckKindHard, Applies: true, Passed: true},
		{Name: "refusal", Kind: CheckKindHard, Applies: false, Passed: false},
		{Name: "tool_choice", Kind: CheckKindSoft, Applies: true, Passed: false},
	}}

	failed := checks.FailedHard()

	assert.Len(t, failed, 1)
	assert.Equal(t, "held_tools", failed[0].Name)
	assert.Nil(t, (*CaseChecks)(nil).FailedHard())
}
