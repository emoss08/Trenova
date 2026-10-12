package hosprojection

// strategyCandidate is one rest plan with the clocks it yields at departure.
type strategyCandidate struct {
	strategy          Strategy
	clocks            Clocks
	restStartDeadline int64
}

type engine struct {
	in   Input
	runs []restRun
}

func newEngine(in *Input) *engine {
	return &engine{in: *in, runs: restRuns(in.Timeline, in.Now)}
}

func (e *engine) hasTimeline() bool { return len(e.in.Timeline) > 0 }

// cycleWindowSec is the rolling window the driver's cycle is measured over.
func (e *engine) cycleWindowSec() int64 {
	if e.in.Limits.CycleMs == sixtyHourCycleMs {
		return sevenDayWindowSec
	}
	return eightDayWindowSec
}

// timelineCoversCycleWindow reports whether persisted logs reach back far enough to
// recompute the cycle from scratch at the departure time.
func (e *engine) timelineCoversCycleWindow(departure int64) bool {
	if !e.hasTimeline() {
		return false
	}
	return e.in.Timeline[0].StartAt <= departure-e.cycleWindowSec()
}

// cycleAvailableAt projects the 60/70-hour clock forward to the departure, assuming no
// further duty before then. With full timeline coverage the rolling window is summed
// exactly, so hours regained as old duty days age out are credited. Without coverage
// the snapshot clock is used, topped up with the provider's cycleTomorrow figure once
// the departure falls on a later calendar day.
func (e *engine) cycleAvailableAt(departure int64) int64 {
	if e.timelineCoversCycleWindow(departure) {
		used := onDutyMsBetween(e.in.Timeline, departure-e.cycleWindowSec(), e.in.Now)
		return e.in.Limits.CycleMs - used
	}

	available := e.in.Clocks.CycleMs
	if e.in.CycleTomorrowMs > available && departure/86400 > e.in.ClocksAt/86400 {
		available = e.in.CycleTomorrowMs
	}
	return available
}

// breakAvailableAt projects the 8-hour break clock to departure: any completed rest of
// 30 minutes or more restarts it, and driving after that restart burns it.
func (e *engine) breakAvailableAt(departure int64) int64 {
	if !e.hasTimeline() {
		return e.in.Clocks.BreakMs
	}
	restartAt := lastQualifyingBreakEnd(e.in.Timeline, departure)
	if restartAt == 0 {
		return e.in.Clocks.BreakMs
	}
	return e.in.Limits.BreakMs - drivingMsBetween(e.in.Timeline, restartAt, departure)
}

// restStartFloor is the earliest a fresh pre-departure rest can begin: now, or the
// start of the rest the driver is already in, which banks the time already spent.
func (e *engine) restStartFloor() int64 {
	if start := trailingRestStart(e.in.Timeline, e.in.Now); start > 0 {
		return start
	}
	return e.in.Now
}

// currentClocks projects the snapshot clocks to departure with no additional rest. The
// 14-hour window keeps burning wall-clock time; the drive clock does not.
func (e *engine) currentClocks(departure int64) (strategyCandidate, bool) {
	elapsedMs := max((departure-e.in.ClocksAt)*1000, 0)

	clocks := Clocks{
		DriveMs: e.in.Clocks.DriveMs,
		ShiftMs: e.in.Clocks.ShiftMs - elapsedMs,
		CycleMs: e.cycleAvailableAt(departure),
		BreakMs: e.projectedBreakMs(departure),
	}

	if clocks.DriveMs <= 0 || clocks.ShiftMs <= 0 {
		return strategyCandidate{}, false
	}
	return strategyCandidate{strategy: StrategyCurrentClocks, clocks: clocks}, true
}

// projectedBreakMs projects the 8-hour break clock to departure: a driver resting for
// 30 minutes or more before departing restarts it in full.
func (e *engine) projectedBreakMs(departure int64) int64 {
	if restStart := trailingRestStart(e.in.Timeline, e.in.Now); restStart > 0 &&
		departure-restStart >= 30*60 {
		return e.in.Limits.BreakMs
	}
	if !e.hasTimeline() {
		if isRestStatus(e.in.DutyStatus) && departure-e.in.Now >= 30*60 {
			return e.in.Limits.BreakMs
		}
		return e.in.Clocks.BreakMs
	}
	return min(e.breakAvailableAt(departure), e.in.Limits.BreakMs)
}

// tenHourReset models 10 consecutive off-duty hours completing before departure,
// restoring the drive and shift clocks in full. The cycle is untouched.
func (e *engine) tenHourReset(departure int64) (strategyCandidate, bool) {
	restStart := e.restStartFloor()
	if restStart+resetRestMs/1000 > departure {
		return strategyCandidate{}, false
	}
	return strategyCandidate{
		strategy: StrategyTenHourReset,
		clocks: Clocks{
			DriveMs: e.in.Limits.DriveMs,
			ShiftMs: e.in.Limits.ShiftMs,
			CycleMs: e.cycleAvailableAt(departure),
			BreakMs: e.in.Limits.BreakMs,
		},
		restStartDeadline: departure - resetRestMs/1000,
	}, true
}

// restart34 models 34 consecutive off-duty hours completing before departure, which
// restarts the 60/70-hour cycle on top of everything a 10-hour reset restores.
func (e *engine) restart34(departure int64) (strategyCandidate, bool) {
	if e.splitDisallowedJurisdiction() {
		return strategyCandidate{}, false
	}
	restStart := e.restStartFloor()
	if restStart+restartMs/1000 > departure {
		return strategyCandidate{}, false
	}
	return strategyCandidate{
		strategy: StrategyRestart34,
		clocks: Clocks{
			DriveMs: e.in.Limits.DriveMs,
			ShiftMs: e.in.Limits.ShiftMs,
			CycleMs: e.in.Limits.CycleMs,
			BreakMs: e.in.Limits.BreakMs,
		},
		restStartDeadline: departure - restartMs/1000,
	}, true
}

func (e *engine) splitDisallowedJurisdiction() bool {
	return e.in.Jurisdiction == canadaSouthJurisCS || e.in.Jurisdiction == canadaNorthJurisCN
}

// splitPair is one qualifying sleeper-berth pairing candidate.
type splitPair struct {
	recalcPoint       int64
	secondStart       int64
	secondEnd         int64
	restStartDeadline int64
}

// splitSleeper searches the timeline for the best qualifying split sleeper pairing
// (8/2 or 7/3): one contiguous sleeper-berth period of at least 7 hours paired with a
// separate rest of at least 2 hours, together totaling at least 10 hours, with the
// pair complete by departure. When paired, neither period counts against the 14-hour
// window and clocks recalculate from the end of the first period; the 11-hour drive
// clock and the cycle are never extended (49 CFR 395.1(g)).
func (e *engine) splitSleeper(departure int64) (strategyCandidate, bool) {
	if e.splitDisallowedJurisdiction() || e.in.Limits.DriveMs != standardDriveMs {
		return strategyCandidate{}, false
	}
	if !e.hasTimeline() {
		return strategyCandidate{}, false
	}

	best := strategyCandidate{}
	bestScore := int64(0)
	found := false
	for _, pair := range e.splitPairs(departure) {
		candidate, ok := e.evaluateSplitPair(departure, pair)
		if !ok {
			continue
		}
		score := min(candidate.clocks.DriveMs, candidate.clocks.ShiftMs)
		if !found || score > bestScore {
			best = candidate
			bestScore = score
			found = true
		}
	}
	return best, found
}

type restPeriod struct {
	start int64
	end   int64
}

// splitPeriods extracts qualifying-period candidates from the rest runs: contiguous
// sleeper stretches of at least 7 hours and any rest of at least 2 hours. The open
// trailing run extends to departure, assuming the driver rests until then.
func (e *engine) splitPeriods(departure int64) (sleepers, rests []restPeriod) {
	sleepers = make([]restPeriod, 0, len(e.runs))
	rests = make([]restPeriod, 0, len(e.runs))
	for _, run := range e.runs {
		end := run.endAt
		sleeperEnd := run.sleeperEnd
		if run.open {
			end = departure
			if run.sleeperStart > 0 && run.sleeperEnd == run.endAt {
				sleeperEnd = departure
			}
		}
		if run.sleeperStart > 0 && sleeperEnd-run.sleeperStart >= splitLongSleeperMs/1000 {
			sleepers = append(sleepers, restPeriod{start: run.sleeperStart, end: sleeperEnd})
		}
		if end-run.startAt >= splitShortRestMs/1000 {
			rests = append(rests, restPeriod{start: run.startAt, end: end})
		}
	}
	return sleepers, rests
}

// splitPairs enumerates completed pairings from the timeline plus, when only the long
// sleeper period exists, a pairing completed by a planned rest that ends exactly at
// departure.
func (e *engine) splitPairs(departure int64) []splitPair {
	sleepers, rests := e.splitPeriods(departure)

	pairs := make([]splitPair, 0, len(sleepers)*2)
	for _, sleeper := range sleepers {
		if sleeper.end > departure {
			continue
		}
		pairs = append(pairs, completedPairsFor(sleeper, rests, departure)...)

		sleeperLen := (sleeper.end - sleeper.start) * 1000
		requiredSecondMs := max(splitShortRestMs, splitPairTotalMs-sleeperLen)
		plannedStart := departure - requiredSecondMs/1000
		if plannedStart >= max(e.in.Now, sleeper.end) {
			pairs = append(pairs, splitPair{
				recalcPoint:       sleeper.end,
				secondStart:       plannedStart,
				secondEnd:         departure,
				restStartDeadline: plannedStart,
			})
		}
	}
	return pairs
}

func completedPairsFor(sleeper restPeriod, rests []restPeriod, departure int64) []splitPair {
	sleeperLen := (sleeper.end - sleeper.start) * 1000

	pairs := make([]splitPair, 0, len(rests))
	for _, rest := range rests {
		if rest.end > departure {
			continue
		}
		if rest.start <= sleeper.end && sleeper.start <= rest.end {
			continue
		}
		if restLen := (rest.end - rest.start) * 1000; sleeperLen+restLen < splitPairTotalMs {
			continue
		}
		first, second := sleeper, rest
		if rest.end < sleeper.end {
			first, second = rest, sleeper
		}
		pairs = append(pairs, splitPair{
			recalcPoint: first.end,
			secondStart: second.start,
			secondEnd:   second.end,
		})
	}
	return pairs
}

// evaluateSplitPair recalculates the clocks from the end of the pair's first period.
// Only the second period lies inside the recalculated window, so its full length is
// excluded from the 14-hour count.
func (e *engine) evaluateSplitPair(departure int64, pair splitPair) (strategyCandidate, bool) {
	windowMs := (departure - pair.recalcPoint) * 1000
	if windowMs <= 0 {
		return strategyCandidate{}, false
	}
	excludedMs := (pair.secondEnd - pair.secondStart) * 1000

	shift := min(e.in.Limits.ShiftMs-windowMs+excludedMs, e.in.Limits.ShiftMs)
	drive := e.in.Limits.DriveMs - drivingMsBetween(e.in.Timeline, pair.recalcPoint, departure)
	if shift <= 0 || drive <= 0 {
		return strategyCandidate{}, false
	}

	return strategyCandidate{
		strategy: StrategySplitSleeper,
		clocks: Clocks{
			DriveMs: drive,
			ShiftMs: shift,
			CycleMs: e.cycleAvailableAt(departure),
			BreakMs: e.in.Limits.BreakMs,
		},
		restStartDeadline: pair.restStartDeadline,
	}, true
}
