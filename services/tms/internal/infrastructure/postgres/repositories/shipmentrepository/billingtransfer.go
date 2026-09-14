package shipmentrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

func billingTransferCandidatePredicate(q *bun.SelectQuery) *bun.SelectQuery {
	sp := buncolgen.ShipmentColumns

	return q.
		Where(sp.Status.In(), bun.List(shipment.BillingTransferCandidateStatuses())).
		Where(
			sp.BillingTransferStatus.Expr("COALESCE({}, '') IN (?)"),
			bun.List([]shipment.BillingTransferStatus{
				shipment.BillingTransferNone,
				shipment.BillingTransferSentBackToOps,
			}),
		)
}

func (r *repository) ListBillingTransferCandidateIDs(
	ctx context.Context,
	req *repositories.ListBillingTransferCandidateIDsRequest,
) (*repositories.BillingTransferCandidateIDsResult, error) {
	dba := r.db.DBForContext(ctx)
	sp := buncolgen.ShipmentColumns

	scope := func(q *bun.SelectQuery) *bun.SelectQuery {
		q = querybuilder.ApplyFiltersWithoutSort(
			q,
			buncolgen.ShipmentTable.Alias,
			req.Filter,
			(*shipment.Shipment)(nil),
		)
		q = billingTransferCandidatePredicate(q)
		if req.Status != "" {
			q = q.Where(sp.Status.Eq(), req.Status)
		}
		return q
	}

	total, err := dba.NewSelect().
		Model((*shipment.Shipment)(nil)).
		Apply(scope).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	result := &repositories.BillingTransferCandidateIDsResult{
		IDs:        make([]pulid.ID, 0, min(total, req.Limit)),
		TotalCount: total,
	}
	if total == 0 || req.Limit <= 0 {
		return result, nil
	}

	if err = dba.NewSelect().
		Model((*shipment.Shipment)(nil)).
		Column(sp.ID.Bare()).
		Apply(scope).
		Order(sp.CreatedAt.OrderAsc(), sp.ID.OrderAsc()).
		Limit(req.Limit).
		Scan(ctx, &result.IDs); err != nil {
		return nil, err
	}

	return result, nil
}
