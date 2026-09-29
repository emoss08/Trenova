package worker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type validatable interface {
	~string
	IsValid() bool
}

func assertEveryValueIsValid[T validatable](t *testing.T, values []T, invalid T) {
	t.Helper()

	assert.NotEmpty(t, values)
	seen := make(map[T]struct{}, len(values))
	for _, value := range values {
		assert.True(t, value.IsValid(), string(value))
		_, dup := seen[value]
		assert.False(t, dup, "%s is listed twice", value)
		seen[value] = struct{}{}
	}
	assert.False(t, invalid.IsValid())
}

func TestValues_AreEveryValidValueOnce(t *testing.T) {
	t.Parallel()

	assertEveryValueIsValid(t, SafetyEventKindValues(), "Fire")
	assertEveryValueIsValid(t, SafetySeverityValues(), "Trivial")
	assertEveryValueIsValid(t, InspectionResultValues(), "Maybe")
	assertEveryValueIsValid(t, RecognitionKindValues(), "Medal")
	assertEveryValueIsValid(t, DOTTestTypeValues(), "Annual")
	assertEveryValueIsValid(t, DOTTestSubstanceValues(), "Caffeine")
	assertEveryValueIsValid(t, LeaveTypeValues(), "Sabbatical")
	assertEveryValueIsValid(t, LeaveFrequencyValues(), "Weekly")
	assertEveryValueIsValid(t, EmploymentVerificationStatusValues(), "Lost")
	assertEveryValueIsValid(t, EmploymentVerificationMethodValues(), "Pigeon")
	assertEveryValueIsValid(t, OSHACaseClassificationValues(), "Minor")
	assertEveryValueIsValid(t, OSHAIllnessTypeValues(), "Cold")
	assertEveryValueIsValid(t, InjuryTreatmentValues(), "Surgery")
	assertEveryValueIsValid(t, InjuryCaseStatusValues(), "Pending")
	assertEveryValueIsValid(t, WorkersCompClaimStatusValues(), "Appealed")
	assertEveryValueIsValid(t, PTOTypeValues(), "Sabbatical")
	assertEveryValueIsValid(t, ReviewGoalStatusValues(), "Blocked")
	assert.Len(t, InspectionResultValues(), 3, "no result is not a value to choose")
}
