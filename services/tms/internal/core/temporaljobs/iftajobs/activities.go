package iftajobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	JurisdictionMileRepo repositories.ShipmentMoveJurisdictionMileRepository
	DistanceCalculation  services.DistanceCalculationService
	Logger               *zap.Logger
}

type Activities struct {
	jurisdictionMileRepo repositories.ShipmentMoveJurisdictionMileRepository
	distanceCalculation  services.DistanceCalculationService
	logger               *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		jurisdictionMileRepo: p.JurisdictionMileRepo,
		distanceCalculation:  p.DistanceCalculation,
		logger:               p.Logger.Named("ifta-activities"),
	}
}

func (a *Activities) ListMovesMissingBreakdownActivity(
	ctx context.Context,
	input ListMovesMissingBreakdownInput,
) (*ListMovesMissingBreakdownResult, error) {
	page, err := a.jurisdictionMileRepo.ListUnattributedMoves(
		ctx,
		repositories.UnattributedMovesRequest{
			TenantInfo: input.TenantInfo,
			Start:      input.Start,
			End:        input.End,
			Limit:      input.Limit,
			Offset:     input.Offset,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list moves missing jurisdiction breakdown: %w", err)
	}
	return &ListMovesMissingBreakdownResult{
		MoveIDs:    page.MoveIDs,
		TotalMoves: page.TotalMoves,
		TotalMiles: page.TotalMiles,
	}, nil
}

func (a *Activities) AttributeMovesActivity(
	ctx context.Context,
	input AttributeMovesInput,
) (*AttributeMovesResult, error) {
	result := &AttributeMovesResult{Miles: decimal.Zero}
	for _, moveID := range input.MoveIDs {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		rows, err := a.distanceCalculation.RecalculateMoveJurisdictionMiles(
			ctx,
			services.RecalculateMoveJurisdictionMilesRequest{
				TenantInfo:     input.TenantInfo,
				ShipmentMoveID: moveID,
				UserID:         input.TenantInfo.UserID,
			},
		)
		if err != nil {
			result.Failed++
			a.logger.Warn("jurisdiction backfill failed for move",
				zap.String("moveId", moveID.String()),
				zap.Error(err))
		} else {
			result.Processed++
			attributed := shipment.ShipmentMove{JurisdictionMiles: rows}
			result.Miles = result.Miles.Add(
				decimal.NewFromFloat(attributed.JurisdictionMilesSumMiles()).Round(2),
			)
		}
		activity.RecordHeartbeat(ctx, result.Processed+result.Failed)
	}
	return result, nil
}

func chunkMoveIDs(ids []pulid.ID, size int) [][]pulid.ID {
	if size <= 0 || len(ids) == 0 {
		return [][]pulid.ID{}
	}
	chunks := make([][]pulid.ID, 0, (len(ids)+size-1)/size)
	for start := 0; start < len(ids); start += size {
		end := min(start+size, len(ids))
		chunks = append(chunks, ids[start:end])
	}
	return chunks
}
