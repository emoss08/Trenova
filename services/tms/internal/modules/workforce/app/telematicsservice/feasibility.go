package telematicsservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/hosfeasibility"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/telematicsports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	FeasibilityVerdictFeasible   = telematics.FeasibilityVerdictFeasible
	FeasibilityVerdictTight      = telematics.FeasibilityVerdictTight
	FeasibilityVerdictInfeasible = telematics.FeasibilityVerdictInfeasible
	FeasibilityVerdictUnknown    = telematics.FeasibilityVerdictUnknown

	feasibilityPositionAgeSec = int64(4 * 3600)
)

type FeasibilityPoint = hosfeasibility.Point

type FeasibilityRequest struct {
	TenantInfo pagination.TenantInfo
	FirstStop  *FeasibilityPoint
	TotalMiles float64
}

type DriverFeasibility = hosfeasibility.Result

func (s *Service) EvaluateDriverFeasibility(
	ctx context.Context,
	req *FeasibilityRequest,
) ([]*DriverFeasibility, error) {
	if err := s.requireSamsara(ctx); err != nil {
		return nil, err
	}
	states, err := s.repo.ListWorkerHOSStates(ctx, &telematicsports.ListWorkerHOSStatesRequest{
		TenantInfo:    req.TenantInfo,
		IncludeWorker: true,
	})
	if err != nil {
		return nil, err
	}

	positions, err := s.repo.ListVehiclePositions(
		ctx,
		&telematicsports.ListVehiclePositionsRequest{
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
		results = append(results, hosfeasibility.Evaluate(
			state,
			positionsByTractor,
			hosfeasibility.Trip{FirstStop: req.FirstStop, TotalMiles: req.TotalMiles},
			now,
		))
	}

	hosfeasibility.Sort(results)
	return results, nil
}
