package hosprojection

type Rest struct {
	Breaks   int
	Resets   int
	Restarts int
	RestMs   int64
}

func (r *Rest) Add(other Rest) {
	r.Breaks += other.Breaks
	r.Resets += other.Resets
	r.Restarts += other.Restarts
	r.RestMs += other.RestMs
}

type Schedule struct {
	Clocks Clocks
	Limits Limits
}

func NewSchedule(clocks Clocks, limits Limits) *Schedule {
	return &Schedule{Clocks: clampClocks(clocks), Limits: limits}
}

func (s *Schedule) usable() bool {
	return s.Limits.DriveMs > 0 && s.Limits.ShiftMs > 0 && s.Limits.CycleMs > 0 &&
		s.Limits.BreakMs > 0
}

func (s *Schedule) Drive(driveMs int64) (int64, Rest) {
	rest := Rest{}
	if driveMs <= 0 {
		return 0, rest
	}
	if !s.usable() {
		return driveMs, rest
	}

	elapsed := int64(0)
	remaining := driveMs
	for remaining > 0 {
		switch {
		case s.Clocks.CycleMs <= 0:
			s.restart()
			elapsed += restartMs
			rest.RestMs += restartMs
			rest.Restarts++
			continue
		case s.Clocks.BreakMs <= 0 && s.Clocks.ShiftMs >= breakRestMs:
			s.Clocks.ShiftMs -= breakRestMs
			s.Clocks.BreakMs = s.Limits.BreakMs
			elapsed += breakRestMs
			rest.RestMs += breakRestMs
			rest.Breaks++
			continue
		case s.Clocks.BreakMs <= 0:
			s.reset()
			elapsed += resetRestMs
			rest.RestMs += resetRestMs
			rest.Resets++
			continue
		}

		leg := min(
			remaining,
			s.Clocks.DriveMs,
			s.Clocks.ShiftMs,
			s.Clocks.CycleMs,
			s.Clocks.BreakMs,
		)
		if leg <= 0 {
			s.reset()
			elapsed += resetRestMs
			rest.RestMs += resetRestMs
			rest.Resets++
			continue
		}

		elapsed += leg
		remaining -= leg
		s.Clocks.DriveMs -= leg
		s.Clocks.ShiftMs -= leg
		s.Clocks.CycleMs -= leg
		s.Clocks.BreakMs -= leg
	}

	return elapsed, rest
}

func (s *Schedule) OnDuty(ms int64) {
	if ms <= 0 {
		return
	}
	s.Clocks.ShiftMs = max(s.Clocks.ShiftMs-ms, 0)
	s.Clocks.CycleMs = max(s.Clocks.CycleMs-ms, 0)
	if ms >= breakRestMs {
		s.Clocks.BreakMs = s.Limits.BreakMs
	}
}

func (s *Schedule) OffDuty(ms int64) {
	switch {
	case ms <= 0:
		return
	case ms >= restartMs:
		s.restart()
	case ms >= resetRestMs:
		s.reset()
	default:
		s.Clocks.ShiftMs = max(s.Clocks.ShiftMs-ms, 0)
		if ms >= breakRestMs {
			s.Clocks.BreakMs = s.Limits.BreakMs
		}
	}
}

func (s *Schedule) reset() {
	s.Clocks.DriveMs = s.Limits.DriveMs
	s.Clocks.ShiftMs = s.Limits.ShiftMs
	s.Clocks.BreakMs = s.Limits.BreakMs
}

func (s *Schedule) restart() {
	s.reset()
	s.Clocks.CycleMs = s.Limits.CycleMs
}

func clampClocks(c Clocks) Clocks {
	return Clocks{
		DriveMs: max(c.DriveMs, 0),
		ShiftMs: max(c.ShiftMs, 0),
		CycleMs: max(c.CycleMs, 0),
		BreakMs: max(c.BreakMs, 0),
	}
}
