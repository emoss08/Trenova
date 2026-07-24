package telematicsservice

import (
	"regexp"
	"strconv"
)

const (
	hourMs                = int64(3_600_000)
	defaultDriveLimitMs   = 11 * hourMs
	defaultShiftLimitMs   = 14 * hourMs
	defaultCycleLimitMs   = 70 * hourMs
	defaultBreakLimitMs   = 8 * hourMs
	canadaSouthJurisdicCS = "CS"
	canadaNorthJurisdicCN = "CN"
)

var cycleHoursPattern = regexp.MustCompile(`(\d+)\s*hour`)

type HOSLimits struct {
	DriveMs int64 `json:"driveMs"`
	ShiftMs int64 `json:"shiftMs"`
	CycleMs int64 `json:"cycleMs"`
	BreakMs int64 `json:"breakMs"`
}

func LimitsForRuleset(cycle, shift, jurisdiction string) HOSLimits {
	limits := HOSLimits{
		DriveMs: defaultDriveLimitMs,
		ShiftMs: defaultShiftLimitMs,
		CycleMs: defaultCycleLimitMs,
		BreakMs: defaultBreakLimitMs,
	}

	if match := cycleHoursPattern.FindStringSubmatch(cycle); len(match) == 2 {
		if hours, err := strconv.ParseInt(match[1], 10, 64); err == nil && hours > 0 {
			limits.CycleMs = hours * hourMs
		}
	}

	switch shift {
	case "US Interstate Passenger":
		limits.DriveMs = 10 * hourMs
		limits.ShiftMs = 15 * hourMs
	case "Texas Intrastate":
		limits.DriveMs = 12 * hourMs
		limits.ShiftMs = 15 * hourMs
	}

	if jurisdiction == canadaSouthJurisdicCS || jurisdiction == canadaNorthJurisdicCN {
		limits.DriveMs = 13 * hourMs
		limits.ShiftMs = 16 * hourMs
	}

	return limits
}
