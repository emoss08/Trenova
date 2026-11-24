package weekhours

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/timesheet"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerschedule"
)

const (
	DaysInWeek       = 7
	secondsPerMinute = int64(60)
)

type DayState string

const (
	DayWorked    = DayState("worked")
	DayScheduled = DayState("scheduled")
	DayMissed    = DayState("missed")
	DayAway      = DayState("away")
	DayOff       = DayState("off")
)

type Assignment struct {
	Template         *workerschedule.ShiftTemplate
	CycleOffsetWeeks int16
	From             int64
	To               *int64
}

type Span struct {
	From int64
	To   int64
}

type Input struct {
	WeekStart   int64
	Location    *time.Location
	Now         int64
	Threshold   int32
	Assignments []Assignment
	Entries     []timesheet.TimeClockEntry
	Away        []Span
}

type Day struct {
	Date    int64
	State   DayState
	Minutes int32
	Crosses bool
}

type Week struct {
	Start     int64
	Days      [DaysInWeek]Day
	Worked    int32
	Remaining int32
	Threshold int32
	Shift     string
	CrossDay  int
}

func (w *Week) Planned() int32 { return w.Worked + w.Remaining }

func (w *Week) Overtime() int32 { return max(0, w.Planned()-w.Threshold) }

func (w *Week) Room() int32 { return max(0, w.Threshold-w.Planned()) }

func (w *Week) Over() bool { return w.Planned() > w.Threshold }

func (w *Week) OnLine() bool { return w.Planned() == w.Threshold }

func (w *Week) AlreadyOver() bool { return w.Worked > w.Threshold }

func Bounds(now int64, loc *time.Location) (start, end int64) {
	loc = locationOr(loc)
	start = workerschedule.StartOfWeekUTC(now, loc)

	return start, time.Unix(start, 0).In(loc).AddDate(0, 0, DaysInWeek).Unix()
}

func TimesheetPeriod(weekStart int64, loc *time.Location) int64 {
	day := time.Unix(weekStart, 0).In(locationOr(loc))

	return time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC).Unix()
}

func ThresholdOr(minutes int32) int32 {
	if minutes > 0 {
		return minutes
	}

	return timesheet.DefaultOvertimeThresholdMinutes
}

func locationOr(loc *time.Location) *time.Location {
	if loc == nil {
		return time.UTC
	}

	return loc
}

func Project(in *Input) Week {
	loc := locationOr(in.Location)
	start := time.Unix(in.WeekStart, 0).In(loc)
	week := Week{
		Start:     in.WeekStart,
		Threshold: ThresholdOr(in.Threshold),
		Shift:     shiftName(in.Assignments, in.Now),
		CrossDay:  -1,
	}

	var running int32
	for offset := range DaysInWeek {
		day := start.AddDate(0, 0, offset)
		dayEnd := start.AddDate(0, 0, offset+1).Unix()
		week.Days[offset] = projectDay(in, day, dayEnd)
		switch week.Days[offset].State {
		case DayWorked:
			week.Worked += week.Days[offset].Minutes
		case DayScheduled:
			week.Remaining += week.Days[offset].Minutes
		case DayMissed, DayAway, DayOff:
		}
		before := running
		running += week.Days[offset].Minutes
		if week.CrossDay < 0 && before <= week.Threshold && running > week.Threshold {
			week.CrossDay = offset
			week.Days[offset].Crosses = true
		}
	}

	return week
}

func projectDay(in *Input, day time.Time, dayEnd int64) Day {
	dayStart := day.Unix()
	out := Day{Date: dayStart, State: DayOff}

	for idx := range in.Entries {
		entry := &in.Entries[idx]
		if entry.ClockedInAt >= dayStart && entry.ClockedInAt < dayEnd {
			out.Minutes += entry.PaidMinutes()
		}
	}
	if out.Minutes > 0 {
		out.State = DayWorked

		return out
	}

	template := templateOn(in.Assignments, day, dayEnd)
	if template == nil {
		return out
	}
	if awayOn(in.Away, dayStart, dayEnd) {
		out.State = DayAway

		return out
	}
	shiftStart := time.Date(
		day.Year(), day.Month(), day.Day(), 0, int(template.StartMinute), 0, 0, day.Location(),
	).Unix()
	if shiftStart+int64(template.DurationMinutes)*secondsPerMinute <= in.Now {
		out.State = DayMissed

		return out
	}
	out.State = DayScheduled
	out.Minutes = int32(template.DurationMinutes)

	return out
}

func templateOn(
	assignments []Assignment,
	day time.Time,
	dayEnd int64,
) *workerschedule.ShiftTemplate {
	var (
		found *workerschedule.ShiftTemplate
		from  int64
	)
	for idx := range assignments {
		assignment := &assignments[idx]
		if assignment.Template == nil || assignment.From >= dayEnd {
			continue
		}
		if assignment.To != nil && *assignment.To < day.Unix() {
			continue
		}
		if !assignment.Template.WorksOnDate(day, assignment.CycleOffsetWeeks) {
			continue
		}
		if found == nil || assignment.From > from {
			found, from = assignment.Template, assignment.From
		}
	}

	return found
}

func awayOn(spans []Span, dayStart, dayEnd int64) bool {
	for _, span := range spans {
		if span.From < dayEnd && (span.To <= 0 || span.To >= dayStart) {
			return true
		}
	}

	return false
}

func shiftName(assignments []Assignment, now int64) string {
	var (
		name string
		from int64
	)
	for idx := range assignments {
		assignment := &assignments[idx]
		if assignment.Template == nil {
			continue
		}
		inForce := assignment.From <= now && (assignment.To == nil || *assignment.To >= now)
		if name == "" || inForce || assignment.From > from {
			name, from = assignment.Template.Name, assignment.From
		}
		if inForce {
			return name
		}
	}

	return name
}
