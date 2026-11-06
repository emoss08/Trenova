package dispatchplanservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchplanprice"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) recordPrices(
	ctx context.Context,
	tenant pagination.TenantInfo,
	moves []*dispatchboardports.BoardMove,
	plan *dispatchplanports.Plan,
) error {
	moveIDs := make([]pulid.ID, 0, len(moves))
	for _, move := range moves {
		moveIDs = append(moveIDs, move.MoveID)
	}

	return s.prices.Replace(ctx, &dispatchplanports.ReplacePricesRequest{
		TenantInfo:  tenant,
		GeneratedAt: plan.GeneratedAt,
		MoveIDs:     moveIDs,
		Prices:      pricesOf(tenant, plan),
	})
}

func pricesOf(
	tenant pagination.TenantInfo,
	plan *dispatchplanports.Plan,
) []*dispatchplanprice.DispatchPlanPrice {
	prices := make(
		[]*dispatchplanprice.DispatchPlanPrice,
		0,
		len(plan.Pairings)+len(plan.Uncovered),
	)
	for _, pairing := range plan.Pairings {
		prices = append(prices, &dispatchplanprice.DispatchPlanPrice{
			OrganizationID:       tenant.OrgID,
			BusinessUnitID:       tenant.BuID,
			MoveID:               pairing.MoveID,
			WorkerID:             pairing.Candidate.WorkerID,
			WorkerName:           pairing.Candidate.WorkerName,
			CostCents:            pairing.PlanCostCents,
			PreferenceFloorCents: pairing.PreferenceFloorCents,
			CostLines:            pairing.Candidate.Cost.Lines,
			CoveragePriceCents:   pairing.CoveragePriceCents,
			DriverValueCents:     pairing.DriverValueCents,
			Alternatives:         pairing.Alternatives,
			GeneratedAt:          plan.GeneratedAt,
		})
	}
	for _, uncovered := range plan.Uncovered {
		price := &dispatchplanprice.DispatchPlanPrice{
			OrganizationID:       tenant.OrgID,
			BusinessUnitID:       tenant.BuID,
			MoveID:               uncovered.MoveID,
			CoveragePriceCents:   uncovered.CoveragePriceCents,
			UncoveredReason:      uncovered.Reason,
			PreferenceFloorCents: uncovered.PreferenceFloorCents,
			Alternatives:         []dispatchplanprice.Alternative{},
			GeneratedAt:          plan.GeneratedAt,
		}
		if uncovered.BestCandidate != nil {
			price.WorkerName = uncovered.BestCandidate.WorkerName
			price.CostCents = uncovered.PlanCostCents
			price.CostLines = uncovered.BestCandidate.Cost.Lines
		}
		prices = append(prices, price)
	}

	return prices
}
