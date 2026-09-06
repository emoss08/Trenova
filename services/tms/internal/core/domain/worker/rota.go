package worker

import (
	"time"

	"github.com/emoss08/trenova/shared/pulid"
)

// RotaDayState is what one worker is doing on one day. The order matters:
// anything above Scheduled overrides the pattern, and the strongest override
// wins so a driver on approved leave never shows as rostered.
type RotaDayState string

const (
	// RotaOff is a day the pattern does not cover.
	RotaOff = RotaDayState("Off")
	// RotaScheduled is a day the pattern covers and nothing overrides.
	RotaScheduled = RotaDayState("Scheduled")
	// RotaAssigned is a day the pattern covers and dispatch has already put
	// work on. It sits above Scheduled because it is the stronger fact.
	RotaAssigned = RotaDayState("Assigned")
	// RotaTimeOff is approved time off.
	RotaTimeOff = RotaDayState("TimeOff")
	// RotaLeave is an open leave case.
	RotaLeave = RotaDayState("Leave")
	// RotaUnavailable is the driver's own stated preference not to work.
	RotaUnavailable = RotaDayState("Unavailable")
)

func (s RotaDayState) String() string { return string(s) }

// rank orders the states so the strongest fact wins. Leave and time off
// outrank an assignment: if dispatch has put work on a day somebody is signed
// off, the rota has to show the conflict rather than the work.
func (s RotaDayState) rank() int {
	switch s {
	case RotaOff:
		return 0
	case RotaScheduled:
		return 1
	case RotaUnavailable:
		return 2
	case RotaAssigned:
		return 3
	case RotaTimeOff:
		return 4
	case RotaLeave:
		return 5
	default:
		return 0
	}
}

// RotaDay is one worker's one day.
type RotaDay struct {
	Date  int64
	State RotaDayState
	// Scheduled says the pattern covers this day, whatever the state ended up
	// being. It is what makes a conflict visible: a day that is both scheduled
	// and TimeOff is a hole in the roster, and a day that is only TimeOff is not.
	Scheduled       bool
	StartMinute     int16
	DurationMinutes int16
	// Preference is the driver's own statement about this weekday, carried
	// through even when something stronger decided the state, so the rota can
	// show where dispatch overrode it rather than hiding the override.
	Preference      AvailabilityPreference
	AssignmentCount int
	Note            string
}

// IsConflict reports a day the pattern rosters somebody onto that they cannot
// work. This is the number a dispatcher is actually looking for.
func (d RotaDay) IsConflict() bool {
	if !d.Scheduled {
		return false
	}
	return d.State == RotaTimeOff || d.State == RotaLeave || d.State == RotaUnavailable
}

// RotaRow is one worker's week.
type RotaRow struct {
	WorkerID      pulid.ID
	Name          string
	FleetCode     string
	FleetColor    string
	ShiftCode     string
	ShiftName     string
	ShiftColor    string
	Days          []RotaDay
	ScheduledDays int
	Conflicts     int
}

// Rota is the composed week.
type Rota struct {
	WeekStart     int64
	WeekEnd       int64
	Weeks         int
	Rows          []RotaRow
	ScheduledDays int
	Conflicts     int
}

// RotaInput is the evidence a week is composed from. Every one of these
// changes without the pattern being touched, which is why the rota is derived
// on read and never stored.
type RotaInput struct {
	WeekStart int64
	// Weeks is how many weeks the board spans. Zero means one. A board drawn
	// several weeks out is how a planner sees an A/B rotation actually
	// alternating rather than having to take it on trust.
	Weeks    int
	Location *time.Location
	Workers  []RotaWorkerInput
}

// RotaWorkerInput is one worker's evidence.
type RotaWorkerInput struct {
	WorkerID   pulid.ID
	Name       string
	FleetCode  string
	FleetColor string
	Template   *ShiftTemplate
	// CycleOffsetWeeks is which week of the template's cycle the worker starts
	// on, taken from their assignment.
	CycleOffsetWeeks int16
	// Preferences is indexed by weekday, 0 for Sunday. A missing entry means
	// no statement, which is not the same as "available" but behaves the same.
	Preferences map[int16]AvailabilityPreference
	// TimeOffDays, LeaveDays and AssignedDays are keyed by the UTC-midnight
	// epoch of the day, so a caller can build them from any source.
	TimeOffDays  map[int64]bool
	LeaveDays    map[int64]bool
	AssignedDays map[int64]int
}

const daysInWeek = 7

// BuildRota composes a week. It is a pure function over evidence gathered
// elsewhere: the pattern says what was planned, and time off, leave, dispatch
// and the driver's own preference each say what actually happened to it.
func BuildRota(in RotaInput) *Rota {
	loc := in.Location
	if loc == nil {
		loc = time.UTC
	}

	weeks := in.Weeks
	if weeks < 1 {
		weeks = 1
	}
	days := daysInWeek * weeks

	start := time.Unix(in.WeekStart, 0).In(loc)
	out := &Rota{
		WeekStart: in.WeekStart,
		WeekEnd:   start.AddDate(0, 0, days).Unix(),
		Weeks:     weeks,
		Rows:      make([]RotaRow, 0, len(in.Workers)),
	}

	for _, worker := range in.Workers {
		row := RotaRow{
			WorkerID:   worker.WorkerID,
			Name:       worker.Name,
			FleetCode:  worker.FleetCode,
			FleetColor: worker.FleetColor,
			Days:       make([]RotaDay, 0, days),
		}
		if worker.Template != nil {
			row.ShiftCode = worker.Template.Code
			row.ShiftName = worker.Template.Name
			row.ShiftColor = worker.Template.Color
		}

		// The rotation is resolved once a week rather than once a day: the
		// seven-day buckets counted from the epoch do not begin on a Sunday,
		// so a per-day answer would flip midweek and put both halves of an A/B
		// pair on at once.
		for week := range weeks {
			weekStart := start.AddDate(0, 0, week*daysInWeek).Unix()
			cycleWeek := cycleWeekFor(weekStart, worker)
			for offset := range daysInWeek {
				day := start.AddDate(0, 0, week*daysInWeek+offset)
				row.Days = append(row.Days, buildRotaDay(worker, day, cycleWeek))
			}
		}
		for _, day := range row.Days {
			if day.Scheduled {
				row.ScheduledDays++
			}
			if day.IsConflict() {
				row.Conflicts++
			}
		}

		out.ScheduledDays += row.ScheduledDays
		out.Conflicts += row.Conflicts
		out.Rows = append(out.Rows, row)
	}

	return out
}

// cycleWeekFor is which week of the rotation this whole week is. It is
// computed once from the week start rather than per day: the seven-day buckets
// counted from the epoch do not begin on a Sunday, so computing it per day
// would flip the rotation midweek and put both halves of an A/B pair on at once.
func cycleWeekFor(weekStart int64, worker RotaWorkerInput) int {
	if worker.Template == nil || worker.Template.CycleWeeks <= 1 {
		return 0
	}
	weekIndex := weekStart / (int64(daysInWeek) * 86400)
	cycle := int64(worker.Template.CycleWeeks)
	return int(((weekIndex+int64(worker.CycleOffsetWeeks))%cycle + cycle) % cycle)
}

func buildRotaDay(worker RotaWorkerInput, day time.Time, cycleWeek int) RotaDay {
	key := day.Unix()
	weekday := int(day.Weekday())

	out := RotaDay{
		Date:       key,
		State:      RotaOff,
		Preference: worker.Preferences[int16(weekday)], //nolint:gosec // 0..6
	}

	if worker.Template != nil {
		if worker.Template.WorksOn(weekday, cycleWeek) {
			out.Scheduled = true
			out.State = RotaScheduled
			out.StartMinute = worker.Template.StartMinute
			out.DurationMinutes = worker.Template.DurationMinutes
		}
	}

	if count := worker.AssignedDays[key]; count > 0 {
		out.AssignmentCount = count
		out.State = strongest(out.State, RotaAssigned)
	}
	if out.Preference == AvailabilityUnavailable {
		out.State = strongest(out.State, RotaUnavailable)
	}
	if worker.TimeOffDays[key] {
		out.State = strongest(out.State, RotaTimeOff)
	}
	if worker.LeaveDays[key] {
		out.State = strongest(out.State, RotaLeave)
	}

	return out
}

func strongest(current, candidate RotaDayState) RotaDayState {
	if candidate.rank() > current.rank() {
		return candidate
	}
	return current
}

// StartOfWeekUTC is the Sunday midnight on or before an instant, which is the
// week the rota is drawn for. Sunday because that is the day the availability
// mask is indexed from, and two different week starts in one feature would be
// a bug waiting to happen.
func StartOfWeekUTC(at int64, loc *time.Location) int64 {
	if loc == nil {
		loc = time.UTC
	}
	t := time.Unix(at, 0).In(loc)
	midnight := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
	return midnight.AddDate(0, 0, -int(midnight.Weekday())).Unix()
}
