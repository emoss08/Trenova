package shipmentrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/shared/pulid"
)

func (r *repository) ListBillingTransferCandidateIDs(
	ctx context.Context,
	req *shipmentports.ListBillingTransferCandidateIDsQuery,
) (*shipmentports.BillingTransferCandidateIDsResult, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*shipmentports.BillingTransferCandidateIDsResult, error) {
			sp := shipmentdb.Shipments
			q := sp.Where(BillingTransferCandidateCondition())
			if req.Status != "" {
				q = q.Where(sp.Status.Eq(req.Status))
			}
			if req.Filter != nil {
				q = listquery.Filter(q, req.Filter)
			}

			total, err := q.Clone().Count(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, err
			}

			result := &shipmentports.BillingTransferCandidateIDsResult{
				IDs:        make([]pulid.ID, 0, min(int(total), req.Limit)),
				TotalCount: int(total),
			}
			if total == 0 || req.Limit <= 0 {
				return result, nil
			}

			order := []keel.Order{sp.CreatedAt.Asc(), sp.ID.Asc()}
			if req.RecentFirst {
				order = []keel.Order{sp.UpdatedAt.Desc(), sp.ID.Desc()}
			}

			ids, err := keel.Into[pulid.ID](
				ctx,
				r.db.Keel(ctx),
				q.Pick(sp.ID).Order(order...).Limit(req.Limit),
			)
			if err != nil {
				return nil, err
			}
			result.IDs = append(result.IDs, ids...)

			return result, nil
		},
	)
}
