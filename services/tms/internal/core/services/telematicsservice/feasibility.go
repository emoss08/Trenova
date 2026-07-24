package telematicsservice

import (
	"context"
	"fmt"
	"sort"

	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/geoutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	FeasibilityVerdictFeasible   = "feasible"
	FeasibilityVerdictTight      = "tight"
	FeasibilityVerdictInfeasible = "infeasible"
	FeasibilityVerdictUnknown    = "unknown"

	averageLinehaulMph        = float64(50)
	feasibilityBufferMs       = int64(30 * 60 * 1000)
	feasibilityTightMarginMs  = int64(90 * 60 * 1000)
	feasibilityStaleSeconds   = int64(12 * 3600)
	feasibilityPositionAgeSec = int64(4 * 3600)
)

type FeasibilityPoint struct {
	Latitude  float64
	Longitude float64
}

type FeasibilityRequest struct {
	TenantInfo pagination.TenantInfo
	FirstStop  *FeasibilityPoint
	TotalMiles float64
}

type DriverFeasibility struct {
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

func (s *Service) EvaluateDriverFeasibility(
	ctx context.Context,
	req *FeasibilityRequest,
) ([]*DriverFeasibility, error) {
	states, err := s.repo.ListWorkerHOSStates(ctx, &repositories.ListWorkerHOSStatesRequest{
		TenantInfo:    req.TenantInfo,
		IncludeWorker: true,
	})
	if err != nil {
		return nil, err
	}

	positions, err := s.repo.ListVehiclePositions(
		ctx,
		&repositories.ListVehiclePositionsRequest{
			TenantInfo:     req.TenantInfo,
			MaxAgeSeconds:  feasibilityPositionAgeSec,
			IncludeTractor: true,
		},
	)
	if err != nil {
		return nil, err
	}
	positionsByTractor := make(map[string]*telematics.VehiclePosition, len(positions))
	for _, position := range positions {
		positionsByTractor[position.TractorID.String()] = position
	}

	now := timeutils.NowUnix()
	results := make([]*DriverFeasibility, 0, len(states))
	for _, state := range states {
		results = append(results, evaluateDriver(state, positionsByTractor, req, now))
	}

	sortFeasibilityResults(results)
	return results, nil
}

func evaluateDriver(
	state *telematics.WorkerHOSState,
	positionsByTractor map[string]*telematics.VehiclePosition,
	req *FeasibilityRequest,
	now int64,
) *DriverFeasibility {
	result := &DriverFeasibility{
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

	totalMiles := req.TotalMiles
	if !state.CurrentTractorID.IsNil() {
		result.TractorID = state.CurrentTractorID.String()
		if position, ok := positionsByTractor[result.TractorID]; ok {
			applyTractorPosition(result, position, req.FirstStop, &totalMiles)
		}
	}

	if totalMiles > 0 {
		result.EstimatedDriveMs = int64(totalMiles / averageLinehaulMph * 3_600_000)
	}

	if now-state.RecordedAt > feasibilityStaleSeconds {
		result.Verdict = FeasibilityVerdictUnknown
		result.Reasons = append(result.Reasons, "HOS data is stale")
		return result
	}

	if state.ShiftDrivingViolationMs > 0 || state.CycleViolationMs > 0 {
		result.Verdict = FeasibilityVerdictInfeasible
		result.Reasons = append(result.Reasons, "Active HOS violation")
		return result
	}
	if state.DriveRemainingMs <= 0 || state.ShiftRemainingMs <= 0 || state.CycleRemainingMs <= 0 {
		result.Verdict = FeasibilityVerdictInfeasible
		result.Reasons = append(result.Reasons, "No available hours remaining")
		return result
	}

	required := result.EstimatedDriveMs + feasibilityBufferMs
	if result.EstimatedDriveMs > 0 {
		if state.DriveRemainingMs < required || state.ShiftRemainingMs < required {
			result.Verdict = FeasibilityVerdictInfeasible
			result.Reasons = append(result.Reasons, fmt.Sprintf(
				"Needs %s of drive time; %s remaining",
				formatMsShort(required),
				formatMsShort(minInt64(state.DriveRemainingMs, state.ShiftRemainingMs)),
			))
			return result
		}
		if state.DriveRemainingMs < required+feasibilityTightMarginMs ||
			state.ShiftRemainingMs < required+feasibilityTightMarginMs {
			result.Verdict = FeasibilityVerdictTight
			result.Reasons = append(result.Reasons, "Clocks cover the trip with little margin")
			return result
		}
	} else if state.DriveRemainingMs < feasibilityTightMarginMs {
		result.Verdict = FeasibilityVerdictTight
		result.Reasons = append(result.Reasons, "Less than 90 minutes of drive time remaining")
		return result
	}

	result.Verdict = FeasibilityVerdictFeasible
	return result
}

func applyTractorPosition(
	result *DriverFeasibility,
	position *telematics.VehiclePosition,
	firstStop *FeasibilityPoint,
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

func sortFeasibilityResults(results []*DriverFeasibility) {
	rank := map[string]int{
		FeasibilityVerdictFeasible:   0,
		FeasibilityVerdictTight:      1,
		FeasibilityVerdictUnknown:    2,
		FeasibilityVerdictInfeasible: 3,
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

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
