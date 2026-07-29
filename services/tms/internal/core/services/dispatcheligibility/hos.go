package dispatcheligibility

import (
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/shared/timeutils"
)

// HOSStateMaxAgeSeconds is the age past which a telematics HOS snapshot is no longer
// trusted for compliance decisions.
const HOSStateMaxAgeSeconds = int64(12 * 3600)

const (
	regShiftDriving = "49 CFR 395.3(a)(3)"
	regShiftWindow  = "49 CFR 395.3(a)(2)"
	regCycle        = "49 CFR 395.3(b)"
	regRestBreak    = "49 CFR 395.3(a)(3)(ii)"

	fieldHOS = "hos"
)

type HOSInput struct {
	State     *telematics.WorkerHOSState
	Control   *dispatchcontrol.DispatchControl
	Now       int64
	ELDExempt bool
}

// EvaluateHOSClocks checks a driver's live hours-of-service clocks. A missing or stale
// snapshot produces an Info finding only — the assignment write path has always treated
// absent telematics as "cannot judge" rather than "not allowed", and the console shows
// it as an unknown verdict instead of a block. ELD-exempt drivers are not judged at
// all: a missing feed is expected for them, not a data gap.
func EvaluateHOSClocks(in HOSInput) *Evaluation {
	eval := NewEvaluation(3)

	if in.Control == nil || !in.Control.EnforceHOSCompliance || in.ELDExempt {
		return eval
	}

	if in.State == nil {
		eval.Add(Finding{
			Code:     CodeHOSMissing,
			Severity: SeverityInfo,
			Field:    fieldHOS,
			Message:  "No hours-of-service data is available for this driver",
		})
		return eval
	}

	now := in.Now
	if now == 0 {
		now = timeutils.NowUnix()
	}

	if now-in.State.RecordedAt > HOSStateMaxAgeSeconds {
		eval.Add(Finding{
			Code:     CodeHOSStale,
			Severity: SeverityInfo,
			Field:    fieldHOS,
			Message:  "Hours-of-service data is stale",
		})
		return eval
	}

	severity := enforcementSeverity(in.Control.ComplianceEnforcementLevel)
	state := in.State

	if state.ShiftDrivingViolationMs > 0 {
		eval.Add(Finding{
			Code:       CodeHOSShiftDrivingViolation,
			Severity:   severity,
			Field:      fieldHOS,
			Message:    "Driver has an active shift driving HOS violation (49 CFR 395.3(a)(3))",
			Regulation: regShiftDriving,
		})
	}
	if state.CycleViolationMs > 0 {
		eval.Add(Finding{
			Code:       CodeHOSCycleViolation,
			Severity:   severity,
			Field:      fieldHOS,
			Message:    "Driver has an active cycle HOS violation (49 CFR 395.3(b))",
			Regulation: regCycle,
		})
	}
	if state.DriveRemainingMs <= 0 {
		eval.Add(Finding{
			Code:       CodeHOSNoDriveTime,
			Severity:   severity,
			Field:      fieldHOS,
			Message:    "Driver has no drive time remaining on the 11-hour clock (49 CFR 395.3(a)(3))",
			Regulation: regShiftDriving,
		})
	}
	if state.ShiftRemainingMs <= 0 {
		eval.Add(Finding{
			Code:       CodeHOSNoShiftTime,
			Severity:   severity,
			Field:      fieldHOS,
			Message:    "Driver has no on-duty time remaining in the 14-hour window (49 CFR 395.3(a)(2))",
			Regulation: regShiftWindow,
		})
	}
	if state.CycleRemainingMs <= 0 {
		eval.Add(Finding{
			Code:       CodeHOSNoCycleTime,
			Severity:   severity,
			Field:      fieldHOS,
			Message:    "Driver has no cycle time remaining (49 CFR 395.3(b))",
			Regulation: regCycle,
		})
	}
	if state.DutyStatus == telematics.DutyStatusDriving && state.BreakRemainingMs <= 0 {
		eval.Add(Finding{
			Code:     CodeHOSBreakRequired,
			Severity: severity,
			Field:    fieldHOS,
			Message: fmt.Sprintf(
				"Driver requires a 30-minute rest break before driving (49 CFR 395.3(a)(3)(ii)); "+
					"break clock at %d minutes",
				state.BreakRemainingMs/60000,
			),
			Regulation: regRestBreak,
		})
	}

	return eval
}
