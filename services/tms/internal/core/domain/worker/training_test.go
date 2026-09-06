package worker_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const trainingDay = int64(86400)

func trainingMonths(n int32) *int32 { return &n }

func trainingTS(v int64) *int64 { return &v }

func course(name string, required bool, driverTypes ...worker.DriverType) *worker.TrainingCourse {
	return &worker.TrainingCourse{
		ID:                     pulid.MustNew("trnc_"),
		Code:                   name,
		Name:                   name,
		Category:               worker.TrainingCategorySafety,
		Status:                 domaintypes.StatusActive,
		Delivery:               worker.TrainingDeliveryClassroom,
		IsRequired:             required,
		RequiredForDriverTypes: driverTypes,
		RenewalWindowDays:      30,
		DueDaysAfterAssignment: 30,
	}
}

func TestTrainingCourseValidate(t *testing.T) {
	t.Parallel()

	t.Run("online course needs a link", func(t *testing.T) {
		t.Parallel()
		c := course("ONLINE", false)
		c.Delivery = worker.TrainingDeliveryOnline
		multiErr := errortypes.NewMultiError()
		c.Validate(multiErr)
		require.True(t, multiErr.HasErrors())
		assert.Equal(t, "contentUrl", multiErr.Errors[0].Field)

		c.ContentURL = "ftp://nope"
		multiErr = errortypes.NewMultiError()
		c.Validate(multiErr)
		require.True(t, multiErr.HasErrors(), "only http(s) links can be opened from the portal")

		c.ContentURL = "https://lms.example.com/course/1"
		multiErr = errortypes.NewMultiError()
		c.Validate(multiErr)
		assert.False(t, multiErr.HasErrors())
	})

	t.Run("passing score is a percentage", func(t *testing.T) {
		t.Parallel()
		c := course("SCORED", false)
		c.PassingScore = decimal.NewNullDecimal(decimal.NewFromInt(120))
		multiErr := errortypes.NewMultiError()
		c.Validate(multiErr)
		require.True(t, multiErr.HasErrors())
		assert.Equal(t, "passingScore", multiErr.Errors[0].Field)
	})
}

func TestTrainingCourseAppliesTo(t *testing.T) {
	t.Parallel()
	otr := &worker.Worker{DriverType: worker.DriverTypeOTR}
	local := &worker.Worker{DriverType: worker.DriverTypeLocal}

	everyone := course("ALL", true)
	otrOnly := course("OTR", true, worker.DriverTypeOTR)
	optional := course("OPT", false)
	retired := course("OLD", true)
	retired.Status = domaintypes.StatusInactive

	assert.True(t, everyone.AppliesTo(local))
	assert.True(t, otrOnly.AppliesTo(otr))
	assert.False(t, otrOnly.AppliesTo(local))
	assert.False(t, optional.AppliesTo(otr))
	assert.False(t, retired.AppliesTo(otr), "inactive courses drop out of the matrix")
}

func TestTrainingCourseGradeAndExpiry(t *testing.T) {
	t.Parallel()
	c := course("HAZ", true)
	c.PassingScore = decimal.NewNullDecimal(decimal.NewFromInt(80))
	c.ValidityMonths = trainingMonths(12)

	_, err := c.Grade(decimal.NullDecimal{})
	var verr *errortypes.Error
	require.ErrorAs(t, err, &verr, "a scored course needs a result")
	assert.Equal(t, "score", verr.Field)

	passed, err := c.Grade(decimal.NewNullDecimal(decimal.NewFromInt(80)))
	require.NoError(t, err)
	assert.True(t, passed, "passing score is inclusive")
	passed, err = c.Grade(decimal.NewNullDecimal(decimal.NewFromFloat(79.5)))
	require.NoError(t, err)
	assert.False(t, passed)

	completed := time.Date(2026, time.January, 31, 12, 0, 0, 0, time.UTC).Unix()
	expiry := c.ExpiryFor(completed)
	require.NotNil(t, expiry)
	assert.Equal(t, time.Date(2027, time.January, 31, 12, 0, 0, 0, time.UTC).Unix(), *expiry)

	oneTime := course("ONCE", true)
	assert.Nil(t, oneTime.ExpiryFor(completed))
	assert.Equal(t, completed+30*trainingDay, *oneTime.DueFor(completed))
}

func TestEvaluateTrainingHealth(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, time.September, 2, 12, 0, 0, 0, time.UTC).Unix()

	cases := []struct {
		name   string
		record *worker.WorkerTrainingRecord
		want   worker.TrainingHealth
	}{
		{"nothing", nil, worker.TrainingHealthMissing},
		{"assigned without due", &worker.WorkerTrainingRecord{Status: worker.TrainingStatusAssigned}, worker.TrainingHealthScheduled},
		{"assigned due in 20", &worker.WorkerTrainingRecord{Status: worker.TrainingStatusAssigned, DueAt: trainingTS(now + 20*trainingDay)}, worker.TrainingHealthScheduled},
		{"in progress due in 3", &worker.WorkerTrainingRecord{Status: worker.TrainingStatusInProgress, DueAt: trainingTS(now + 3*trainingDay)}, worker.TrainingHealthDueSoon},
		{"due today", &worker.WorkerTrainingRecord{Status: worker.TrainingStatusAssigned, DueAt: trainingTS(now)}, worker.TrainingHealthDueSoon},
		{"overdue", &worker.WorkerTrainingRecord{Status: worker.TrainingStatusAssigned, DueAt: trainingTS(now - trainingDay)}, worker.TrainingHealthOverdue},
		{"completed one-time", &worker.WorkerTrainingRecord{Status: worker.TrainingStatusCompleted, CompletedAt: trainingTS(now - 400*trainingDay)}, worker.TrainingHealthCurrent},
		{"completed expiring", &worker.WorkerTrainingRecord{Status: worker.TrainingStatusCompleted, ExpiresAt: trainingTS(now + 10*trainingDay)}, worker.TrainingHealthExpiringSoon},
		{"completed expired", &worker.WorkerTrainingRecord{Status: worker.TrainingStatusCompleted, ExpiresAt: trainingTS(now - trainingDay)}, worker.TrainingHealthExpired},
		{"waived", &worker.WorkerTrainingRecord{Status: worker.TrainingStatusWaived}, worker.TrainingHealthCurrent},
		{"failed", &worker.WorkerTrainingRecord{Status: worker.TrainingStatusFailed}, worker.TrainingHealthFailed},
		{"cancelled", &worker.WorkerTrainingRecord{Status: worker.TrainingStatusCancelled}, worker.TrainingHealthMissing},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, worker.EvaluateTrainingHealth(tc.record, 30, now))
		})
	}
}

func TestBuildTrainingSummary(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, time.September, 2, 12, 0, 0, 0, time.UTC).Unix()
	wrk := &worker.Worker{ID: pulid.MustNew("wrk_"), DriverType: worker.DriverTypeOTR}

	orientation := course("ORIENT", true)
	orientation.SortOrder = 10
	hazmat := course("HAZMAT", true, worker.DriverTypeOTR)
	hazmat.SortOrder = 20
	hazmat.ValidityMonths = trainingMonths(12)
	localOnly := course("LOCAL", true, worker.DriverTypeLocal)
	forklift := course("FORKLIFT", false)
	defensive := course("DEFENSIVE", false)

	records := []*worker.WorkerTrainingRecord{
		{CourseID: orientation.ID, Status: worker.TrainingStatusCompleted, CompletedAt: trainingTS(now - 300*trainingDay)},
		{CourseID: hazmat.ID, Status: worker.TrainingStatusCompleted, CompletedAt: trainingTS(now - 400*trainingDay), ExpiresAt: trainingTS(now - 35*trainingDay)},
		{CourseID: hazmat.ID, Status: worker.TrainingStatusAssigned, AssignedAt: now - 10*trainingDay, DueAt: trainingTS(now + 5*trainingDay)},
		{CourseID: forklift.ID, Status: worker.TrainingStatusCancelled},
		{CourseID: defensive.ID, Status: worker.TrainingStatusFailed, UpdatedAt: now - 2*trainingDay},
		{CourseID: defensive.ID, Status: worker.TrainingStatusCompleted, CompletedAt: trainingTS(now - 30*trainingDay)},
	}

	summary := worker.BuildTrainingSummary(
		wrk,
		[]*worker.TrainingCourse{orientation, hazmat, localOnly, forklift, defensive},
		records,
		now,
	)

	require.Len(t, summary.Items, 3, "Local-only course and the cancelled forklift record are left out")
	assert.Equal(t, 2, summary.RequiredCount)
	assert.True(t, summary.Compliant, "an open renewal that is not overdue keeps the worker compliant")

	byCode := map[string]*worker.TrainingSummaryItem{}
	for _, item := range summary.Items {
		byCode[item.Course.Code] = item
	}
	assert.Equal(t, worker.TrainingHealthCurrent, byCode["ORIENT"].Health)
	assert.Equal(t, worker.TrainingHealthDueSoon, byCode["HAZMAT"].Health, "the open assignment speaks for the course, not the expired completion")
	assert.Equal(t, worker.TrainingStatusAssigned, byCode["HAZMAT"].Record.Status)
	assert.Equal(t, worker.TrainingHealthCurrent, byCode["DEFENSIVE"].Health, "the newest closed record wins")
	assert.False(t, byCode["DEFENSIVE"].Required)
	assert.Equal(t, "ORIENT", summary.Items[0].Course.Code, "required first, then by sort order and name")

	assert.Equal(t, 2, summary.CurrentCount)
	assert.Equal(t, 1, summary.DueCount)
	assert.Empty(t, summary.RequiredGaps())

	missing := worker.BuildTrainingSummary(wrk, []*worker.TrainingCourse{orientation, hazmat}, nil, now)
	assert.False(t, missing.Compliant)
	assert.Equal(t, 2, missing.MissingCount)
	gaps := missing.RequiredGaps()
	require.Len(t, gaps, 2)
	attention := missing.Attention()
	assert.Equal(t, worker.TrainingHealthMissing, attention[0].Health)
}
