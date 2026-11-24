package workforcequerytools

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/weekhours"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/weekhoursports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeWeekHours struct {
	leading *weekhoursports.Leading
	asked   *weekhoursports.LeadingRequest
}

func (f *fakeWeekHours) Workers(
	context.Context,
	*weekhoursports.WorkersRequest,
) (map[pulid.ID]*weekhoursports.WorkerWeek, error) {
	return nil, nil
}

func (f *fakeWeekHours) Leading(
	_ context.Context,
	req *weekhoursports.LeadingRequest,
) (*weekhoursports.Leading, error) {
	f.asked = req

	return f.leading, nil
}

type fakeTeam struct {
	ids []pulid.ID
	err error
}

func (f fakeTeam) Narrow(
	context.Context,
	*permissionports.RequestActor,
	pagination.TenantInfo,
) ([]pulid.ID, error) {
	return f.ids, f.err
}

func TestWeekHoursListsTheWeekAndWithholdsAPartialCount(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.November, 22, 0, 0, 0, 0, time.UTC)
	days := [weekhours.DaysInWeek]weekhours.Day{}
	for idx := range days {
		days[idx] = weekhours.Day{Date: start.AddDate(0, 0, idx).Unix(), State: weekhours.DayOff}
	}
	days[0] = weekhours.Day{Date: days[0].Date, State: weekhours.DayWorked, Minutes: 600}
	days[6] = weekhours.Day{
		Date: days[6].Date, State: weekhours.DayScheduled, Minutes: 600, Crosses: true,
	}
	hector := &weekhoursports.WorkerWeek{
		WorkerID: pulid.MustNew("wrk_"),
		Name:     "Hector Alvarez",
		Week: weekhours.Week{
			Days: days, Worked: 690, Remaining: 2400, Threshold: 2400, Shift: "Dock nights",
			CrossDay: 6,
		},
	}
	weeks := &fakeWeekHours{leading: &weekhoursports.Leading{
		Start: start.Unix(), End: start.AddDate(0, 0, 7).Unix(),
		Rows: []*weekhoursports.WorkerWeek{hector}, OnRota: 6, Over: 3, Exact: true,
	}}
	managers := []pulid.ID{pulid.MustNew("usr_")}
	call := &weekHoursCall{
		weeks:  weeks,
		narrow: fakeTeam{ids: managers},
		params: &agenttoolports.QueryToolParams{
			Actor:  &permissionports.RequestActor{},
			Params: map[string]any{"limit": 5},
		},
	}

	result, err := readWeekHours(context.Background(), call)
	require.NoError(t, err)
	assert.Equal(t, managers, weeks.asked.ManagerIDs)
	assert.Equal(t, 5+weekHoursRoomRows, weeks.asked.Limit)
	require.NotNil(t, result.PastLine)
	assert.Equal(t, int64(3), *result.PastLine)
	require.Len(t, result.Workers, 1)
	row := result.Workers[0]
	assert.Equal(t, "11:30", row.Worked)
	assert.Equal(t, "40:00", row.StillScheduled)
	assert.Equal(t, "11:30", row.ProjectedOvertime)
	assert.Equal(t, "0:00", row.RoomUnderLine)
	assert.Equal(t, "Sat Nov 28", row.CrossesOn)
	assert.Equal(t, "worked", row.Days[0].State)
	assert.Contains(t, result.HowToShow, "rotaWeek")

	weeks.leading.Exact = false
	result, err = readWeekHours(context.Background(), call)
	require.NoError(t, err)
	assert.Nil(t, result.PastLine)
	assert.NotEmpty(t, result.Partial)
}

func TestWeekHoursSaysSoWhenTheTeamIsNobody(t *testing.T) {
	t.Parallel()

	weeks := &fakeWeekHours{}
	result, err := readWeekHours(context.Background(), &weekHoursCall{
		weeks:  weeks,
		narrow: fakeTeam{err: weekhoursports.ErrNoTeam},
		params: &agenttoolports.QueryToolParams{Actor: &permissionports.RequestActor{}},
	})
	require.NoError(t, err)
	assert.Nil(t, weeks.asked)
	assert.Empty(t, result.Workers)
	assert.NotEmpty(t, result.Restricted)
}

func TestWeekHoursAreWithheldBelowTheTimesheetsRank(t *testing.T) {
	t.Parallel()

	weeks := &fakeWeekHours{}
	result, err := readWeekHours(context.Background(), &weekHoursCall{
		weeks:       weeks,
		narrow:      fakeTeam{},
		params:      &agenttoolports.QueryToolParams{Actor: &permissionports.RequestActor{}},
		hoursHidden: true,
	})
	require.NoError(t, err)
	assert.Nil(t, weeks.asked, "nothing is read when the hours cannot be shown")
	assert.Empty(t, result.Workers)
	assert.Equal(t, []string{"hours"}, result.Withheld)
	assert.NotEmpty(t, result.Restricted)
}
