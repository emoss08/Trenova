package workersafety_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
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

func TestBuildSafetyScorecard(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, time.September, 2, 12, 0, 0, 0, time.UTC).Unix()
	workerID := pulid.MustNew("wrk_")

	t.Run("clean record is excellent", func(t *testing.T) {
		t.Parallel()
		card := workersafety.BuildSafetyScorecard(workerID, nil, nil, nil, now)
		assert.Equal(t, int32(100), card.Score)
		assert.Equal(t, worker.SafetyRatingExcellent, card.Rating)
		assert.Nil(t, card.CleanInspectionRate)
		assert.Nil(t, card.DaysSinceLastEvent)
	})

	t.Run(
		"points, preventable accidents, out-of-service and discipline all cost",
		func(t *testing.T) {
			t.Parallel()
			expired := now - 30*safetyDay
			events := []*workersafety.WorkerSafetyEvent{
				safetyEvent(
					workersafety.SafetyEventAccident,
					40,
					func(e *workersafety.WorkerSafetyEvent) {
						e.Points = 4
						e.Preventable = true
						e.Status = workersafety.SafetyEventStatusOpen
					},
				),
				safetyEvent(
					workersafety.SafetyEventInspection,
					20,
					func(e *workersafety.WorkerSafetyEvent) {
						e.InspectionResult = workersafety.InspectionResultOutOfService
						e.Points = 5
					},
				),
				safetyEvent(
					workersafety.SafetyEventInspection,
					100,
					func(e *workersafety.WorkerSafetyEvent) {
						e.InspectionResult = workersafety.InspectionResultPass
					},
				),
				safetyEvent(
					workersafety.SafetyEventCitation,
					500,
					func(e *workersafety.WorkerSafetyEvent) {
						e.Points = 3
						e.PointsExpireAt = &expired
					},
				),
				safetyEvent(workersafety.SafetyEventNearMiss, 5, nil),
			}
			actions := []*workersafety.WorkerDisciplinaryAction{
				{
					Level:    workersafety.DisciplinaryLevelWrittenWarning,
					Status:   workersafety.DisciplinaryStatusActive,
					IssuedAt: now - 10*safetyDay,
				},
				{
					Level:    workersafety.DisciplinaryLevelSuspension,
					Status:   workersafety.DisciplinaryStatusRescinded,
					IssuedAt: now - 5*safetyDay,
				},
			}
			recognitions := []*workersafety.WorkerRecognition{
				{OccurredAt: now - 3*safetyDay},
				{OccurredAt: now - 500*safetyDay},
			}

			card := workersafety.BuildSafetyScorecard(workerID, events, actions, recognitions, now)

			assert.Equal(t, int32(9), card.ActivePoints, "the expired citation no longer counts")
			assert.Equal(t, int32(1), card.Accidents)
			assert.Equal(t, int32(1), card.PreventableAccidents)
			assert.Equal(t, int32(0), card.Citations, "the citation is older than twelve months")
			assert.Equal(t, int32(2), card.Inspections)
			assert.Equal(t, int32(1), card.InspectionsPassed)
			assert.Equal(t, int32(1), card.OutOfServiceOrders)
			assert.Equal(t, int32(1), card.NearMisses)
			assert.Equal(t, int32(1), card.OpenEvents)
			assert.Equal(t, int32(1), card.ActiveDiscipline, "rescinded actions are off the ladder")
			assert.Equal(t, workersafety.DisciplinaryLevelWrittenWarning, card.HighestDiscipline)
			assert.Equal(t, int32(1), card.Recognitions)
			require.NotNil(t, card.CleanInspectionRate)
			assert.InDelta(t, 0.5, *card.CleanInspectionRate, 0.001)
			require.NotNil(t, card.DaysSinceLastEvent)
			assert.Equal(
				t,
				int64(40),
				*card.DaysSinceLastEvent,
				"near misses and inspections do not reset the streak",
			)

			// 100 - 9*5 - 1*10 - 1*15 - 1*5 = 25
			assert.Equal(t, int32(25), card.Score)
			assert.Equal(t, worker.SafetyRatingAtRisk, card.Rating)
		},
	)

	t.Run("points alone can put a worker on watch", func(t *testing.T) {
		t.Parallel()
		events := []*workersafety.WorkerSafetyEvent{
			safetyEvent(
				workersafety.SafetyEventCitation,
				400,
				func(e *workersafety.WorkerSafetyEvent) { e.Points = 6 },
			),
		}
		card := workersafety.BuildSafetyScorecard(workerID, events, nil, nil, now)
		assert.Equal(t, int32(70), card.Score)
		assert.Equal(t, worker.SafetyRatingWatch, card.Rating)
	})
}

func TestDisciplinaryLadder(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, time.September, 2, 12, 0, 0, 0, time.UTC).Unix()

	clean := workersafety.BuildDisciplinaryLadder(nil, now)
	assert.Equal(t, workersafety.DisciplinaryLevelCoaching, clean.SuggestedLevel)
	assert.False(t, clean.AtFinalStep)

	lapsed := now - safetyDay
	actions := []*workersafety.WorkerDisciplinaryAction{
		{
			Level:    workersafety.DisciplinaryLevelCoaching,
			Status:   workersafety.DisciplinaryStatusActive,
			IssuedAt: now - 300*safetyDay,
		},
		{
			Level:     workersafety.DisciplinaryLevelFinalWarning,
			Status:    workersafety.DisciplinaryStatusActive,
			IssuedAt:  now - 400*safetyDay,
			ExpiresAt: &lapsed,
		},
		{
			Level:    workersafety.DisciplinaryLevelWrittenWarning,
			Status:   workersafety.DisciplinaryStatusActive,
			IssuedAt: now - 60*safetyDay,
		},
	}
	ladder := workersafety.BuildDisciplinaryLadder(actions, now)
	require.Len(t, ladder.ActiveActions, 2)
	assert.Equal(
		t,
		workersafety.DisciplinaryLevelWrittenWarning,
		ladder.HighestLevel,
		"the lapsed final warning is off the ladder",
	)
	assert.Equal(t, workersafety.DisciplinaryLevelFinalWarning, ladder.SuggestedLevel)

	top := workersafety.BuildDisciplinaryLadder([]*workersafety.WorkerDisciplinaryAction{
		{
			Level:    workersafety.DisciplinaryLevelSuspension,
			Status:   workersafety.DisciplinaryStatusActive,
			IssuedAt: now - safetyDay,
		},
	}, now)
	assert.Equal(t, workersafety.DisciplinaryLevelTermination, top.SuggestedLevel)
	assert.True(t, top.AtFinalStep)
	assert.Equal(
		t,
		workersafety.DisciplinaryLevelTermination,
		workersafety.DisciplinaryLevelTermination.Next(),
	)

	action := &workersafety.WorkerDisciplinaryAction{
		WorkerID: pulid.MustNew("wrk_"), Level: workersafety.DisciplinaryLevelSuspension,
		Status: workersafety.DisciplinaryStatusActive, Reason: "Repeated late departures", IssuedAt: now,
	}
	multiErr := errortypes.NewMultiError()
	action.Validate(multiErr)
	require.True(t, multiErr.HasErrors())
	assert.Equal(t, "suspensionDays", multiErr.Errors[0].Field)
	action.DefaultExpiry()
	require.NotNil(t, action.ExpiresAt)
	assert.Equal(t, timeutils.AddMonthsUTC(now, 12), *action.ExpiresAt)

	termination := &workersafety.WorkerDisciplinaryAction{
		Level:    workersafety.DisciplinaryLevelTermination,
		IssuedAt: now,
	}
	termination.DefaultExpiry()
	assert.Nil(t, termination.ExpiresAt, "terminations never roll off")
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
