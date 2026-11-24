package weekhoursrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/weekhours"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/weekhoursports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func crewWeek(name string, planned int32) *weekhoursports.WorkerWeek {
	return &weekhoursports.WorkerWeek{
		WorkerID: pulid.MustNew("wrk_"),
		Name:     name,
		Week: weekhours.Week{
			Remaining: planned, Threshold: 2400, Shift: "Yard days", CrossDay: -1,
		},
	}
}

func names(rows []*weekhoursports.WorkerWeek) []string {
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.Name)
	}

	return out
}

func TestRankPutsTheFurthestOverFirstAndKeepsTheMostRoomApart(t *testing.T) {
	t.Parallel()

	weeks := map[pulid.ID]*weekhoursports.WorkerWeek{}
	for _, w := range []*weekhoursports.WorkerWeek{
		crewWeek("Hector Alvarez", 3000),
		crewWeek("Janet Cole", 2700),
		crewWeek("Dmitri Volkov", 2550),
		crewWeek("Sam Whitaker", 2400),
		crewWeek("Kwame Boateng", 1920),
		crewWeek("Rosa Lind", 960),
		crewWeek("Ana Moss", 2100),
	} {
		weeks[w.WorkerID] = w
	}

	out := rank(&weekhoursports.Leading{}, weeks, &weekhoursports.LeadingRequest{Limit: 5, Room: 2})

	assert.Equal(
		t,
		[]string{"Hector Alvarez", "Janet Cole", "Dmitri Volkov"},
		names(out.Rows),
	)
	assert.Equal(t, []string{"Rosa Lind", "Kwame Boateng"}, names(out.Room))
	assert.Equal(t, int64(3), out.Over)
	assert.Equal(t, int64(1), out.OnLine)
}

func TestRankShowsEveryoneWhenTheRotaIsSmall(t *testing.T) {
	t.Parallel()

	a, b := crewWeek("A", 2500), crewWeek("B", 1000)
	out := rank(&weekhoursports.Leading{}, map[pulid.ID]*weekhoursports.WorkerWeek{
		a.WorkerID: a, b.WorkerID: b,
	}, &weekhoursports.LeadingRequest{Limit: 8, Room: 2})

	assert.Equal(t, []string{"A", "B"}, names(out.Rows))
	assert.Empty(t, out.Room)
}
