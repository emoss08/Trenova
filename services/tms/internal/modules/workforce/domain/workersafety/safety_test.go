package workersafety_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerreview"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workersafety"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const safetyDay = int64(86400)

func safetyEvent(
	kind workersafety.SafetyEventKind,
	occurredAgo int64,
	mutate func(*workersafety.WorkerSafetyEvent),
) *workersafety.WorkerSafetyEvent {
	now := time.Date(2026, time.September, 2, 12, 0, 0, 0, time.UTC).Unix()
	e := &workersafety.WorkerSafetyEvent{
		ID:          pulid.MustNew("wsev_"),
		WorkerID:    pulid.MustNew("wrk_"),
		Kind:        kind,
		Severity:    workersafety.SafetySeverityMinor,
		Status:      workersafety.SafetyEventStatusClosed,
		OccurredAt:  now - occurredAgo*safetyDay,
		Description: "test",
	}
	if mutate != nil {
		mutate(e)
	}
	return e
}

func TestDefaultSafetyPoints(t *testing.T) {
	t.Parallel()
	assert.Equal(
		t,
		int32(2),
		workersafety.DefaultSafetyPoints(
			workersafety.SafetyEventAccident,
			workersafety.SafetySeverityMinor,
			false,
			"",
		),
	)
	assert.Equal(
		t,
		int32(8),
		workersafety.DefaultSafetyPoints(
			workersafety.SafetyEventAccident,
			workersafety.SafetySeverityMajor,
			true,
			"",
		),
	)
	assert.Equal(
		t,
		int32(4),
		workersafety.DefaultSafetyPoints(
			workersafety.SafetyEventIncident,
			workersafety.SafetySeverityCritical,
			true,
			"",
		),
	)
	assert.Equal(
		t,
		int32(3),
		workersafety.DefaultSafetyPoints(
			workersafety.SafetyEventCitation,
			workersafety.SafetySeverityModerate,
			false,
			"",
		),
	)
	assert.Equal(
		t,
		int32(0),
		workersafety.DefaultSafetyPoints(
			workersafety.SafetyEventNearMiss,
			workersafety.SafetySeverityCritical,
			true,
			"",
		),
	)
	assert.Equal(
		t,
		int32(0),
		workersafety.DefaultSafetyPoints(
			workersafety.SafetyEventInspection,
			workersafety.SafetySeverityMinor,
			false,
			workersafety.InspectionResultPass,
		),
	)
	assert.Equal(
		t,
		int32(5),
		workersafety.DefaultSafetyPoints(
			workersafety.SafetyEventInspection,
			workersafety.SafetySeverityMinor,
			false,
			workersafety.InspectionResultOutOfService,
		),
	)
}

func TestSafetyEventValidate(t *testing.T) {
	t.Parallel()

	t.Run("inspections need a result and only inspections carry one", func(t *testing.T) {
		t.Parallel()
		e := safetyEvent(workersafety.SafetyEventInspection, 1, nil)
		e.Status = workersafety.SafetyEventStatusOpen
		multiErr := errortypes.NewMultiError()
		e.Validate(multiErr)
		require.True(t, multiErr.HasErrors())
		assert.Equal(t, "inspectionResult", multiErr.Errors[0].Field)

		c := safetyEvent(
			workersafety.SafetyEventCitation,
			1,
			func(e *workersafety.WorkerSafetyEvent) {
				e.Status = workersafety.SafetyEventStatusOpen
				e.InspectionResult = workersafety.InspectionResultPass
			},
		)
		multiErr = errortypes.NewMultiError()
		c.Validate(multiErr)
		require.True(t, multiErr.HasErrors())
		assert.Equal(t, "inspectionResult", multiErr.Errors[0].Field)
	})

	t.Run("closing needs a resolution and money cannot be negative", func(t *testing.T) {
		t.Parallel()
		e := safetyEvent(
			workersafety.SafetyEventAccident,
			3,
			func(e *workersafety.WorkerSafetyEvent) {
				e.CostAmount = decimal.NewNullDecimal(decimal.NewFromInt(-5))
			},
		)
		multiErr := errortypes.NewMultiError()
		e.Validate(multiErr)
		fields := make([]string, 0, len(multiErr.Errors))
		for _, err := range multiErr.Errors {
			fields = append(fields, err.Field)
		}
		assert.Contains(t, fields, "resolution")
		assert.Contains(t, fields, "costAmount")
	})
}

func TestSafetyEventPoints(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, time.September, 2, 12, 0, 0, 0, time.UTC).Unix()
	e := safetyEvent(
		workersafety.SafetyEventAccident,
		10,
		func(e *workersafety.WorkerSafetyEvent) { e.Points = 4 },
	)
	e.DefaultPointsExpiry()
	require.NotNil(t, e.PointsExpireAt)
	assert.Equal(
		t,
		timeutils.AddMonthsUTC(e.OccurredAt, 24),
		*e.PointsExpireAt,
		"points roll off after two years",
	)
	assert.Equal(t, int32(4), e.ActivePoints(now))
	assert.Equal(t, int32(0), e.ActivePoints(*e.PointsExpireAt), "expired on the day")

	zero := safetyEvent(workersafety.SafetyEventNearMiss, 1, nil)
	zero.DefaultPointsExpiry()
	assert.Nil(t, zero.PointsExpireAt)
}

func TestPerformanceReviewScoring(t *testing.T) {
	t.Parallel()
	template := &workerreview.PerformanceReviewTemplate{
		Items: []workerreview.ReviewItem{
			{Key: "safety", Label: "Safety", Weight: 3},
			{Key: "service", Label: "Customer service", Weight: 2},
			{Key: "paperwork", Label: "Paperwork", Weight: 1},
		},
	}
	ratings := workerreview.RatingsFromTemplate(template)
	require.Len(t, ratings, 3)
	assert.False(t, workerreview.ComputeOverallScore(ratings).Valid, "nothing rated yet")

	five, three, four := int32(5), int32(3), int32(4)
	ratings[0].Score = &five
	ratings[1].Score = &three
	ratings[2].Score = &four
	score := workerreview.ComputeOverallScore(ratings)
	require.True(t, score.Valid)
	// (5*3 + 3*2 + 4*1) / 6 = 25/6 = 4.17
	assert.Equal(t, "4.17", score.Decimal.StringFixed(2))

	review := &workerreview.PerformanceReview{
		WorkerID: pulid.MustNew(
			"wrk_",
		), TemplateID: pulid.MustNew("prt_"), Status: workerreview.ReviewStatusDraft,
		Title: "H1 2026", PeriodStart: 1_700_000_000, PeriodEnd: 1_710_000_000,
		Ratings: workerreview.RatingsFromTemplate(template),
	}
	draft := errortypes.NewMultiError()
	review.Validate(draft)
	assert.False(t, draft.HasErrors(), "a draft may be unrated")

	submit := errortypes.NewMultiError()
	review.ValidateForSubmit(submit)
	require.True(t, submit.HasErrors())
	fields := make([]string, 0, len(submit.Errors))
	for _, err := range submit.Errors {
		fields = append(fields, err.Field)
	}
	assert.Contains(t, fields, "ratings[0].score")
	assert.Contains(t, fields, "summary")

	cadence := int32(6)
	template.CadenceMonths = &cadence
	next := review.NextReviewDate(template)
	require.NotNil(t, next)
	assert.Equal(t, timeutils.AddMonthsUTC(review.PeriodEnd, 6), *next)
}
