package dispatchconsoleservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) attachPlanPrices(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moves []*BoardMove,
) error {
	moveIDs := make([]pulid.ID, 0, len(moves))
	for _, move := range moves {
		if move != nil && move.BoardMove != nil && !move.IsCovered {
			moveIDs = append(moveIDs, move.MoveID)
		}
	}
	if len(moveIDs) == 0 {
		return nil
	}

	prices, err := s.prices.ListByMoves(ctx, &dispatchplanports.ListPricesRequest{
		TenantInfo: tenantInfo,
		MoveIDs:    moveIDs,
	})
	if err != nil {
		return err
	}

	byMove := make(map[pulid.ID]int, len(prices))
	for i, price := range prices {
		byMove[price.MoveID] = i
	}
	for _, move := range moves {
		if move == nil || move.BoardMove == nil || move.IsCovered {
			continue
		}
		if i, ok := byMove[move.MoveID]; ok {
			move.PlanPrice = prices[i]
		}
	}

	return nil
}

func attachPlannedMoves(drivers []*BoardDriver, moves []*BoardMove) {
	planned := make(map[pulid.ID]*PlannedMove, len(moves))
	for _, move := range moves {
		price := move.PlanPrice
		if price == nil || !price.Covered() {
			continue
		}
		current, ok := planned[price.WorkerID]
		if ok && current.GeneratedAt > price.GeneratedAt {
			continue
		}
		planned[price.WorkerID] = &PlannedMove{
			MoveID:      price.MoveID,
			ProNumber:   move.ProNumber,
			CostCents:   price.CostCents,
			ValueCents:  price.DriverValueCents,
			GeneratedAt: price.GeneratedAt,
		}
	}
	if len(planned) == 0 {
		return
	}

	for _, driver := range drivers {
		driver.PlannedMove = planned[driver.WorkerID]
	}
}
