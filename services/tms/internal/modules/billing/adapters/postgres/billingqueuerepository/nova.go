package billingqueuerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingvocab"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const novaBacklogReason = "list the organizations with billing queue items Nova has not reviewed"

// unreviewed is an invoice item the checks have read and Nova has not; it is
// what idx_billing_queue_items_nova_backlog holds, written as that index's
// predicate.
func unreviewed() []keel.Pred {
	bq := billingqueuedb.BillingQueueItems

	return []keel.Pred{
		bq.AgentReviewHash.IsNull(),
		bq.BillType.Eq(billingvocab.BillTypeInvoice),
		bq.Lane.In(billingqueue.LaneNeedsYou, billingqueue.LaneWaiting, billingqueue.LaneReady),
		bq.Status.In(billingqueue.StatusReadyForReview, billingqueue.StatusInReview),
		bq.ShipmentID.IsNotNull(),
	}
}

func (r *repository) ListUnreviewedItems(
	ctx context.Context,
	req *billingqueueports.ListUnreviewedItemsRequest,
) ([]*billingqueueports.UnreviewedItem, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*billingqueueports.UnreviewedItem, error) {
			bq := billingqueuedb.BillingQueueItems
			preds := unreviewed()
			if after := req.After; after != nil {
				preds = append(preds, keel.Or(
					bq.CreatedAt.Gt(after.CreatedAt),
					keel.And(bq.CreatedAt.Eq(after.CreatedAt), bq.ID.Gt(after.ID)),
				))
			}

			rows, err := bq.Pick(bq.ID, bq.CreatedAt).
				Where(preds...).
				Order(bq.CreatedAt.Asc(), bq.ID.Asc()).
				Limit(req.Limit).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, err
			}

			out := make([]*billingqueueports.UnreviewedItem, 0, len(rows))
			for _, row := range rows {
				out = append(out, &billingqueueports.UnreviewedItem{ID: row.ID, CreatedAt: row.CreatedAt})
			}

			return out, nil
		},
	)
}

type backlogTenant struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

func (r *repository) ListNovaBacklogTenants(
	ctx context.Context,
	limit int,
) ([]pagination.TenantInfo, error) {
	ctx = dbscope.WithSystem(ctx, novaBacklogReason)

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pagination.TenantInfo, error) {
		bq := billingqueuedb.BillingQueueItems
		rows, err := keel.Into[backlogTenant](ctx, r.db.Keel(ctx), bq.
			Pick(bq.OrganizationID, bq.BusinessUnitID).
			Where(unreviewed()...).
			Group(bq.OrganizationID, bq.BusinessUnitID).
			Limit(limit).
			AllTenants())
		if err != nil {
			return nil, err
		}

		out := make([]pagination.TenantInfo, 0, len(rows))
		for _, row := range rows {
			out = append(out, pagination.TenantInfo{OrgID: row.OrganizationID, BuID: row.BusinessUnitID})
		}

		return out, nil
	})
}
