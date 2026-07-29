package telematicsservice

import "github.com/emoss08/trenova/internal/core/services/hosprojection"

const hourMs = int64(3_600_000)

type HOSLimits struct {
	DriveMs int64 `json:"driveMs"`
	ShiftMs int64 `json:"shiftMs"`
	CycleMs int64 `json:"cycleMs"`
	BreakMs int64 `json:"breakMs"`
}

func LimitsForRuleset(cycle, shift, jurisdiction string) HOSLimits {
	limits := hosprojection.LimitsForRuleset(cycle, shift, jurisdiction)
	return HOSLimits{
		DriveMs: limits.DriveMs,
		ShiftMs: limits.ShiftMs,
		CycleMs: limits.CycleMs,
		BreakMs: limits.BreakMs,
	}
}
