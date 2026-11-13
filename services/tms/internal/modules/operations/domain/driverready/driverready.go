package driverready

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerschedule"
)

const (
	MinUsefulSeconds    = int64(3 * 3600)
	ResetSeconds        = int64(10 * 3600)
	RestartSeconds      = int64(34 * 3600)
	ClocksMaxAgeSeconds = int64(12 * 3600)
	maxPatternDays      = 28
)

type Pattern struct {
	Template         *workerschedule.ShiftTemplate
	CycleOffsetWeeks int16
}

type Clocks struct {
	RecordedAt           int64
	DriveSeconds         int64
	ShiftSeconds         int64
	CycleSeconds         int64
	CycleTomorrowSeconds int64
}

type Input struct {
	Now          int64
	FreeAt       int64
	Clocks       *Clocks
	TimeOffUntil int64
	Pattern      *Pattern
	Location     *time.Location
}

func At(in Input) int64 {
	ready := max(in.FreeAt, in.Now)
	if c := in.Clocks; c != nil && in.Now-c.RecordedAt <= ClocksMaxAgeSeconds {
		ready += rest(ready-in.Now, c)
	}
	ready = max(ready, in.TimeOffUntil)
	if in.Pattern != nil && in.Pattern.Template != nil {
		ready = in.Pattern.nextWorkingTime(ready, in.Location)
	}

	return ready
}

func (p *Pattern) nextWorkingTime(ready int64, location *time.Location) int64 {
	if location == nil {
		location = time.UTC
	}
	local := time.Unix(ready, 0).In(location)
	if p.Template.WorksOnDate(local, p.CycleOffsetWeeks) {
		return ready
	}
	midnight := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
	for ahead := 1; ahead <= maxPatternDays; ahead++ {
		day := midnight.AddDate(0, 0, ahead)
		if p.Template.WorksOnDate(day, p.CycleOffsetWeeks) {
			return day.Add(time.Duration(p.Template.StartMinute) * time.Minute).Unix()
		}
	}

	return ready
}

func rest(working int64, c *Clocks) int64 {
	if c.CycleSeconds-working < MinUsefulSeconds {
		if c.CycleTomorrowSeconds >= MinUsefulSeconds {
			return ResetSeconds
		}

		return RestartSeconds
	}
	shift := min(c.DriveSeconds, c.ShiftSeconds)
	if working > 0 && working >= shift {
		return 0
	}
	if shift-working < MinUsefulSeconds {
		return ResetSeconds
	}

	return 0
}
