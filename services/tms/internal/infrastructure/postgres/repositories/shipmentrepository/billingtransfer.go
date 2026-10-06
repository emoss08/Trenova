package shipmentrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

const billingTransferCandidateAlias = "candidate"

func billingTransferCandidatePredicate(q *bun.SelectQuery) *bun.SelectQuery {
	return q.Where("?", BillingTransferCandidateCondition())
}

func (r *repository) ListBillingTransferCandidateIDs(
	ctx context.Context,
	req *repositories.ListBillingTransferCandidateIDsRequest,
) (*repositories.BillingTransferCandidateIDsResult, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*repositories.BillingTransferCandidateIDsResult, error) {
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

			matched := dba.NewSelect().
				Model((*shipment.Shipment)(nil)).
				Apply(scope)
			candidate := sp.ID.WithAlias(billingTransferCandidateAlias)
			candidateCreatedAt := sp.CreatedAt.WithAlias(billingTransferCandidateAlias)

			if err = dba.NewSelect().
				TableExpr("(?) AS ?", matched, bun.Ident(billingTransferCandidateAlias)).
				ColumnExpr(candidate.Qualified()).
				Order(candidateCreatedAt.OrderAsc(), candidate.OrderAsc()).
				Limit(req.Limit).
				Scan(ctx, &result.IDs); err != nil {
				return nil, err
			}

			return result, nil
		},
	)
}
