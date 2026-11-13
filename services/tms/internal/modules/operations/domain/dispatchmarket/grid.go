package dispatchmarket

import (
	"time"

	"github.com/emoss08/trenova/shared/timeutils"
)

const DaySeconds = int64(24 * 3600)

type Grid struct {
	location *time.Location
}

func GridIn(location *time.Location) Grid {
	if location == nil {
		location = time.UTC
	}

	return Grid{location: location}
}

func GridNamed(timezone string) Grid {
	return GridIn(timeutils.LoadLocation(timezone))
}

func (g Grid) Location() *time.Location {
	if g.location == nil {
		return time.UTC
	}

	return g.location
}

func (g Grid) Timezone() string {
	return g.Location().String()
}

func (g Grid) Wall(at int64) int64 {
	_, offset := time.Unix(at, 0).In(g.Location()).Zone()

	return at + int64(offset)
}

func (g Grid) Instant(wall int64) int64 {
	t := time.Unix(wall, 0).UTC()

	return time.Date(
		t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, g.Location(),
	).Unix()
}

func (g Grid) Week(at int64) int64 {
	return WeekOf(g.Wall(at))
}

func (g Grid) Slot(at int64) int16 {
	return SlotOf(g.Wall(at))
}

func (g Grid) Day(at int64) int64 {
	return g.Wall(at) / DaySeconds
}

func (g Grid) SlotStart(at int64) int64 {
	return g.Instant(SlotStart(g.Wall(at)))
}

func (g Grid) SlotEnd(at int64) int64 {
	return g.Instant(SlotStart(g.Wall(at)) + SlotSeconds)
}

func (g Grid) Starts(at int64, slots int) []int64 {
	first := SlotStart(g.Wall(at))
	starts := make([]int64, slots)
	for i := range starts {
		starts[i] = g.Instant(first + int64(i)*SlotSeconds)
	}

	return starts
}

func (g Grid) DayStart(at int64, daysBack int) int64 {
	day := g.Wall(at)/DaySeconds - int64(daysBack)

	return g.Instant(day * DaySeconds)
}

func WallDay(week int64, slot int16) int64 {
	return (week*WeekSeconds+MondayOffset)/DaySeconds + int64(slot)/SlotsPerDay
}
