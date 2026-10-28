package worker_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/employment"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerdot"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerleave"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerreview"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workersafety"
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

	assertEveryValueIsValid(t, workersafety.SafetyEventKindValues(), "Fire")
	assertEveryValueIsValid(t, workersafety.SafetySeverityValues(), "Trivial")
	assertEveryValueIsValid(t, workersafety.InspectionResultValues(), "Maybe")
	assertEveryValueIsValid(t, workersafety.RecognitionKindValues(), "Medal")
	assertEveryValueIsValid(t, workerdot.DOTTestTypeValues(), "Annual")
	assertEveryValueIsValid(t, workerdot.DOTTestSubstanceValues(), "Caffeine")
	assertEveryValueIsValid(t, worker.LeaveTypeValues(), "Sabbatical")
	assertEveryValueIsValid(t, workerleave.LeaveFrequencyValues(), "Weekly")
	assertEveryValueIsValid(t, employment.EmploymentVerificationStatusValues(), "Lost")
	assertEveryValueIsValid(t, employment.EmploymentVerificationMethodValues(), "Pigeon")
	assertEveryValueIsValid(t, workersafety.OSHACaseClassificationValues(), "Minor")
	assertEveryValueIsValid(t, workersafety.OSHAIllnessTypeValues(), "Cold")
	assertEveryValueIsValid(t, workersafety.InjuryTreatmentValues(), "Surgery")
	assertEveryValueIsValid(t, workersafety.InjuryCaseStatusValues(), "Pending")
	assertEveryValueIsValid(t, workersafety.WorkersCompClaimStatusValues(), "Appealed")
	assertEveryValueIsValid(t, worker.PTOTypeValues(), "Sabbatical")
	assertEveryValueIsValid(t, workerreview.ReviewGoalStatusValues(), "Blocked")
	assert.Len(t, workersafety.InspectionResultValues(), 3, "no result is not a value to choose")
}
