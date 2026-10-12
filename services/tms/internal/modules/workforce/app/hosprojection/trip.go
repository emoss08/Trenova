package hosprojection

// PlanTrip simulates driving tripDriveMs from the given starting clocks, inserting the
// mandated 30-minute break whenever the 8-hour driving clock runs out and a full
// 10-hour reset whenever the drive or shift clock exhausts mid-trip. The cycle is
// deliberately never regained en route: rolling recovery depends on day-by-day duty
// history that cannot be guaranteed at planning time, so exhausting it fails the plan.
// A LimiterNone return means the trip completes.
func PlanTrip(tripDriveMs int64, start Clocks, limits Limits) (TripPlan, Limiter) {
	plan := TripPlan{}
	drive := start.DriveMs
	shift := start.ShiftMs
	cycle := start.CycleMs
	breakClock := start.BreakMs

	remaining := tripDriveMs
	for remaining > 0 {
		if cycle <= 0 {
			return plan, LimiterCycle
		}

		if breakClock <= 0 {
			if shift < breakRestMs {
				if !enRouteReset(&plan, &drive, &shift, &breakClock, limits) {
					return plan, LimiterShift
				}
				continue
			}
			plan.TotalMs += breakRestMs
			shift -= breakRestMs
			breakClock = limits.BreakMs
			plan.Breaks++
			continue
		}

		leg := min(remaining, drive, shift, cycle, breakClock)
		if leg <= 0 {
			limiter := LimiterShift
			if drive <= 0 {
				limiter = LimiterDrive
			}
			if !enRouteReset(&plan, &drive, &shift, &breakClock, limits) {
				return plan, limiter
			}
			continue
		}

		plan.TotalMs += leg
		remaining -= leg
		drive -= leg
		shift -= leg
		cycle -= leg
		breakClock -= leg
	}

	plan.MarginMs = min(drive, shift, cycle)
	return plan, LimiterNone
}

func enRouteReset(plan *TripPlan, drive, shift, breakClock *int64, limits Limits) bool {
	if plan.Resets >= maxEnRouteResets {
		return false
	}
	plan.TotalMs += resetRestMs
	*drive = limits.DriveMs
	*shift = limits.ShiftMs
	*breakClock = limits.BreakMs
	plan.Resets++
	return true
}
