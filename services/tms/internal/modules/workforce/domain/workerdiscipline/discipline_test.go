package workerdiscipline_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerdiscipline"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const safetyDay = int64(86400)

func TestDisciplinaryLadder(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, time.September, 2, 12, 0, 0, 0, time.UTC).Unix()

	clean := workerdiscipline.BuildDisciplinaryLadder(nil, now)
	assert.Equal(t, workerdiscipline.DisciplinaryLevelCoaching, clean.SuggestedLevel)
	assert.False(t, clean.AtFinalStep)

	lapsed := now - safetyDay
	actions := []*workerdiscipline.WorkerDisciplinaryAction{
		{
			Level:    workerdiscipline.DisciplinaryLevelCoaching,
			Status:   workerdiscipline.DisciplinaryStatusActive,
			IssuedAt: now - 300*safetyDay,
		},
		{
			Level:     workerdiscipline.DisciplinaryLevelFinalWarning,
			Status:    workerdiscipline.DisciplinaryStatusActive,
			IssuedAt:  now - 400*safetyDay,
			ExpiresAt: &lapsed,
		},
		{
			Level:    workerdiscipline.DisciplinaryLevelWrittenWarning,
			Status:   workerdiscipline.DisciplinaryStatusActive,
			IssuedAt: now - 60*safetyDay,
		},
	}
	ladder := workerdiscipline.BuildDisciplinaryLadder(actions, now)
	require.Len(t, ladder.ActiveActions, 2)
	assert.Equal(
		t,
		workerdiscipline.DisciplinaryLevelWrittenWarning,
		ladder.HighestLevel,
		"the lapsed final warning is off the ladder",
	)
	assert.Equal(t, workerdiscipline.DisciplinaryLevelFinalWarning, ladder.SuggestedLevel)

	top := workerdiscipline.BuildDisciplinaryLadder([]*workerdiscipline.WorkerDisciplinaryAction{
		{
			Level:    workerdiscipline.DisciplinaryLevelSuspension,
			Status:   workerdiscipline.DisciplinaryStatusActive,
			IssuedAt: now - safetyDay,
		},
	}, now)
	assert.Equal(t, workerdiscipline.DisciplinaryLevelTermination, top.SuggestedLevel)
	assert.True(t, top.AtFinalStep)
	assert.Equal(
		t,
		workerdiscipline.DisciplinaryLevelTermination,
		workerdiscipline.DisciplinaryLevelTermination.Next(),
	)

	action := &workerdiscipline.WorkerDisciplinaryAction{
		WorkerID: pulid.MustNew("wrk_"), Level: workerdiscipline.DisciplinaryLevelSuspension,
		Status: workerdiscipline.DisciplinaryStatusActive, Reason: "Repeated late departures", IssuedAt: now,
	}
	multiErr := errortypes.NewMultiError()
	action.Validate(multiErr)
	require.True(t, multiErr.HasErrors())
	assert.Equal(t, "suspensionDays", multiErr.Errors[0].Field)
	action.DefaultExpiry()
	require.NotNil(t, action.ExpiresAt)
	assert.Equal(t, timeutils.AddMonthsUTC(now, 12), *action.ExpiresAt)

	termination := &workerdiscipline.WorkerDisciplinaryAction{
		Level:    workerdiscipline.DisciplinaryLevelTermination,
		IssuedAt: now,
	}
	termination.DefaultExpiry()
	assert.Nil(t, termination.ExpiresAt, "terminations never roll off")
}
