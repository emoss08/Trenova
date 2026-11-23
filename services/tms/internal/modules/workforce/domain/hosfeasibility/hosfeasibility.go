package hosfeasibility

import (
	"fmt"
	"sort"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/shared/geoutils"
	"github.com/emoss08/trenova/shared/intutils"
)

const (
	averageLinehaulMph       = geoutils.PlanningLinehaulMph
	feasibilityBufferMs      = int64(30 * 60 * 1000)
	feasibilityTightMarginMs = int64(90 * 60 * 1000)
	feasibilityStaleSeconds  = int64(12 * 3600)
)

type Point struct {
	Latitude  float64
	Longitude float64
}

type Trip struct {
	FirstStop  *Point
	TotalMiles float64
}

type Result struct {
	WorkerID         string
	WorkerName       string
	DutyStatus       telematics.DutyStatus
	DriveRemainingMs int64
	ShiftRemainingMs int64
	CycleRemainingMs int64
	DeadheadMiles    *float64
	EstimatedDriveMs int64
	Verdict          string
	Reasons          []string
	TractorID        string
	TractorCode      string
	RecordedAt       int64
}

func Evaluate(
	state *telematics.WorkerHOSState,
	positionsByTractor map[string]*telematics.VehiclePosition,
	trip Trip,
	now int64,
) *Result {
	result := &Result{
		WorkerID:         state.WorkerID.String(),
		DutyStatus:       state.DutyStatus,
		DriveRemainingMs: state.DriveRemainingMs,
		ShiftRemainingMs: state.ShiftRemainingMs,
		CycleRemainingMs: state.CycleRemainingMs,
		RecordedAt:       state.RecordedAt,
		Reasons:          make([]string, 0, 2),
	}
	if state.Worker != nil {
		result.WorkerName = workerFullName(state.Worker.FirstName, state.Worker.LastName)
	}

	totalMiles := trip.TotalMiles
	if !state.CurrentTractorID.IsNil() {
		result.TractorID = state.CurrentTractorID.String()
		if position, ok := positionsByTractor[result.TractorID]; ok {
			applyTractorPosition(result, position, trip.FirstStop, &totalMiles)
		}
	}

	if totalMiles > 0 {
		result.EstimatedDriveMs = int64(totalMiles / averageLinehaulMph * 3_600_000)
	}

	if now-state.RecordedAt > feasibilityStaleSeconds {
		result.Verdict = telematics.FeasibilityVerdictUnknown
		result.Reasons = append(result.Reasons, "HOS data is stale")
		return result
	}

	if state.ShiftDrivingViolationMs > 0 || state.CycleViolationMs > 0 {
		result.Verdict = telematics.FeasibilityVerdictInfeasible
		result.Reasons = append(result.Reasons, "Active HOS violation")
		return result
	}
	if state.DriveRemainingMs <= 0 || state.ShiftRemainingMs <= 0 || state.CycleRemainingMs <= 0 {
		result.Verdict = telematics.FeasibilityVerdictInfeasible
		result.Reasons = append(result.Reasons, "No available hours remaining")
		return result
	}

	required := result.EstimatedDriveMs + feasibilityBufferMs
	if result.EstimatedDriveMs > 0 {
		if state.DriveRemainingMs < required || state.ShiftRemainingMs < required {
			result.Verdict = telematics.FeasibilityVerdictInfeasible
			result.Reasons = append(result.Reasons, fmt.Sprintf(
				"Needs %s of drive time; %s remaining",
				formatMsShort(required),
				formatMsShort(intutils.Min(state.DriveRemainingMs, state.ShiftRemainingMs)),
			))
			return result
		}
		if state.DriveRemainingMs < required+feasibilityTightMarginMs ||
			state.ShiftRemainingMs < required+feasibilityTightMarginMs {
			result.Verdict = telematics.FeasibilityVerdictTight
			result.Reasons = append(result.Reasons, "Clocks cover the trip with little margin")
			return result
		}
	} else if state.DriveRemainingMs < feasibilityTightMarginMs {
		result.Verdict = telematics.FeasibilityVerdictTight
		result.Reasons = append(result.Reasons, "Less than 90 minutes of drive time remaining")
		return result
	}

	result.Verdict = telematics.FeasibilityVerdictFeasible
	return result
}

func applyTractorPosition(
	result *Result,
	position *telematics.VehiclePosition,
	firstStop *Point,
	totalMiles *float64,
) {
	if position.Tractor != nil {
		result.TractorCode = position.Tractor.Code
	}
	if firstStop == nil {
		return
	}
	deadhead := geoutils.HaversineMiles(
		position.Latitude,
		position.Longitude,
		firstStop.Latitude,
		firstStop.Longitude,
	)
	result.DeadheadMiles = &deadhead
	*totalMiles += deadhead
}

func Sort(results []*Result) {
	rank := map[string]int{
		telematics.FeasibilityVerdictFeasible:   0,
		telematics.FeasibilityVerdictTight:      1,
		telematics.FeasibilityVerdictUnknown:    2,
		telematics.FeasibilityVerdictInfeasible: 3,
	}
	sort.SliceStable(results, func(i, j int) bool {
		if rank[results[i].Verdict] != rank[results[j].Verdict] {
			return rank[results[i].Verdict] < rank[results[j].Verdict]
		}
		return results[i].DriveRemainingMs > results[j].DriveRemainingMs
	})
}

func workerFullName(first, last string) string {
	if first == "" {
		return last
	}
	if last == "" {
		return first
	}
	return first + " " + last
}

func formatMsShort(ms int64) string {
	hours := ms / 3_600_000
	minutes := (ms % 3_600_000) / 60_000
	return fmt.Sprintf("%dh %02dm", hours, minutes)
}
