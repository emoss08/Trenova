package driversettlementservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

type MovePayEstimate struct {
	GrossMinor   int64  `json:"grossMinor"`
	CurrencyCode string `json:"currencyCode"`
}

// EstimateWorkerMovePay computes what a move would pay a worker under their
// effective pay assignment without persisting anything. Returns a not-found
// error when the worker has no pay assignment, so callers can treat "no
// estimate" as a normal condition.
func (s *Service) EstimateWorkerMovePay(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	shipmentID pulid.ID,
	moveID pulid.ID,
) (*MovePayEstimate, error) {
	sp, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:         shipmentID,
		TenantInfo: tenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, err
	}

	totalDistance := decimal.Zero
	for _, move := range sp.Moves {
		totalDistance = totalDistance.Add(moveDistance(move))
	}

	targetIndex := -1
	for i, move := range sp.Moves {
		if move != nil && move.ID == moveID {
			targetIndex = i
			break
		}
	}
	if targetIndex == -1 {
		return nil, errortypes.NewNotFoundError("Move not found on this shipment")
	}

	assignment, err := s.assignmentRepo.GetEffectiveForWorker(
		ctx,
		repositories.GetWorkerPayAssignmentRequest{
			TenantInfo: tenantInfo,
			WorkerID:   workerID,
			AsOf:       timeutils.NowUnix(),
		},
	)
	if err != nil {
		return nil, errortypes.NewNotFoundError("No pay assignment in effect for this driver")
	}
	if assignment.PayProfile == nil {
		return nil, errortypes.NewNotFoundError("Pay assignment is missing its pay profile")
	}

	_, gross := computeMovePay(&moveCalcInput{
		Profile:           assignment.PayProfile,
		SplitPercent:      assignment.SplitPercent,
		RateOverrides:     assignment.RateOverrides,
		Shipment:          sp,
		Move:              sp.Moves[targetIndex],
		TotalTripDistance: totalDistance,
		MoveCount:         len(sp.Moves),
		HasHazmat:         shipmentHasHazmat(sp),
		FuelSurcharge:     shipmentFuelSurcharge(sp),
	})

	return &MovePayEstimate{
		GrossMinor:   gross,
		CurrencyCode: assignment.PayProfile.CurrencyCode,
	}, nil
}
