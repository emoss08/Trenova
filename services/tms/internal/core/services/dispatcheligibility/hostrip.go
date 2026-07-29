package dispatcheligibility

import (
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/services/hosprojection"
	"github.com/emoss08/trenova/shared/timeutils"
)

const hosTripTightMarginMs = int64(3_600_000)

type HOSTripInput struct {
	Projection *hosprojection.Result
	Control    *dispatchcontrol.DispatchControl
}

// EvaluateHOSTrip turns a forward hours-of-service projection into findings: an
// infeasible trip is a violation-grade finding on the clock that binds, while a trip
// that only works after additional rest — or with under an hour of margin — is a
// warning the dispatcher should see before committing the driver.
func EvaluateHOSTrip(in HOSTripInput) *Evaluation {
	eval := NewEvaluation(1)

	if in.Projection == nil || in.Control == nil || !in.Control.EnforceHOSCompliance {
		return eval
	}

	projection := in.Projection
	if !projection.Feasible {
		eval.Add(Finding{
			Code:       CodeHOSInsufficientForTrip,
			Severity:   enforcementSeverity(in.Control.ComplianceEnforcementLevel),
			Field:      fieldHOS,
			Message:    projection.Detail,
			Regulation: regulationForLimiter(projection.Limiter),
		})
		return eval
	}

	if projection.Strategy != hosprojection.StrategyCurrentClocks {
		eval.Add(Finding{
			Code:     CodeHOSTightForTrip,
			Severity: SeverityWarn,
			Field:    fieldHOS,
			Message:  projection.Detail,
		})
		return eval
	}

	if projection.Trip.MarginMs < hosTripTightMarginMs {
		eval.Add(Finding{
			Code:     CodeHOSTightForTrip,
			Severity: SeverityWarn,
			Field:    fieldHOS,
			Message: fmt.Sprintf(
				"Only %s of clock left after the trip",
				timeutils.FormatDurationMs(projection.Trip.MarginMs),
			),
		})
	}

	return eval
}

func regulationForLimiter(limiter hosprojection.Limiter) string {
	switch limiter {
	case hosprojection.LimiterDrive:
		return regShiftDriving
	case hosprojection.LimiterShift:
		return regShiftWindow
	case hosprojection.LimiterCycle:
		return regCycle
	case hosprojection.LimiterBreak:
		return regRestBreak
	case hosprojection.LimiterNone:
		return ""
	default:
		return ""
	}
}
