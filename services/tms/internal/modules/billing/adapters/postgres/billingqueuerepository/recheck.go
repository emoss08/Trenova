package billingqueuerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
)

const (
	recheckStaleReason   = "mark billing queue items whose checks are missing or stale as due across every organization"
	recheckTenantsReason = "list the organizations with billing queue items whose checks are due"
)

// MarkRecheckDue makes the items due. Every request moves the marker
// forward, never back and never to the value it held, so the drain's settle,
// which clears the marker only while it still holds the value it claimed,
// keeps an item requested again while its checks ran.
func (r *repository) MarkRecheckDue(
	ctx context.Context,
	req *billingqueueports.MarkRecheckDueRequest,
) error {
	ids := sliceutils.Dedupe(req.ItemIDs)
	if len(ids) == 0 {
		return nil
	}

	ctx = dbscope.WithTenant(ctx, req.TenantInfo.DBTenant())
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		bq := billingqueuedb.BillingQueueItems
		_, err := bq.Where(bq.ID.In(ids...)).Update(
			ctx,
			r.db.Keel(ctx),
			bq.RecheckDueAt.ToExpr(
				bq.RecheckDueAt.Coalesce(int64(0)).Add(keel.Int64(1)).Greatest(keel.Int64(req.At)),
			),
			bq.Version.ToExpr(bq.Version),
			bq.UpdatedAt.ToExpr(bq.UpdatedAt),
		)

		return err
	})
}

func (r *repository) ListRecheckDue(
	ctx context.Context,
	req *billingqueueports.ListRecheckDueRequest,
) ([]*billingqueueports.RecheckDueItem, error) {
	ctx = dbscope.WithTenant(ctx, req.TenantInfo.DBTenant())
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*billingqueueports.RecheckDueItem, error) {
			bq := billingqueuedb.BillingQueueItems
			rows, err := keel.Into[billingqueueports.RecheckDueItem](
				ctx,
				r.db.Keel(ctx),
				bq.Pick(bq.ID, bq.RecheckDueAt).
					Where(bq.RecheckDueAt.Lte(req.Now)).
					Order(bq.RecheckDueAt.Asc(), bq.ID.Asc()).
					Limit(req.Limit),
			)
			if err != nil {
				return nil, err
			}

			return sliceutils.Pointers(rows), nil
		},
	)
}

func (r *repository) SettleRecheck(
	ctx context.Context,
	req *billingqueueports.SettleRecheckRequest,
) error {
	ctx = dbscope.WithTenant(ctx, req.TenantInfo.DBTenant())
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		bq := billingqueuedb.BillingQueueItems
		next := bq.RecheckDueAt.ToNull()
		if req.RetryAt != nil {
			next = bq.RecheckDueAt.To(*req.RetryAt)
		}
		_, err := bq.Where(
			bq.ID.Eq(req.ItemID),
			bq.RecheckDueAt.Eq(req.ClaimedDueAt),
		).Update(
			ctx,
			r.db.Keel(ctx),
			next,
			bq.Version.ToExpr(bq.Version),
			bq.UpdatedAt.ToExpr(bq.UpdatedAt),
		)

		return err
	})
}

func (r *repository) MarkStaleRecheckDue(
	ctx context.Context,
	req *billingqueueports.MarkStaleRecheckDueRequest,
) (int64, error) {
	ctx = dbscope.WithSystem(ctx, recheckStaleReason)
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		bq := billingqueuedb.BillingQueueItems
		stale := bq.Pick(bq.ID).
			Where(
				bq.Lane.In(billingqueue.ReviewLanes()...),
				bq.Status.In(billingqueue.ReviewStatuses()...),
				keel.Or(bq.CheckedAt.IsNull(), bq.CheckedAt.Lt(req.StaleBefore)),
				bq.RecheckDueAt.IsNull(),
			).
			Order(bq.CheckedAt.Asc().NullsFirst(), bq.ID.Asc()).
			Limit(req.Limit).
			AllTenants()

		return bq.AllTenants().
			Where(bq.ID.InQuery(stale), bq.RecheckDueAt.IsNull()).
			Update(
				ctx,
				r.db.Keel(ctx),
				bq.RecheckDueAt.To(req.Now),
				bq.Version.ToExpr(bq.Version),
				bq.UpdatedAt.ToExpr(bq.UpdatedAt),
			)
	})
}

func (r *repository) ListRecheckDueTenants(
	ctx context.Context,
	req *billingqueueports.ListRecheckDueTenantsRequest,
) ([]pagination.TenantInfo, error) {
	ctx = dbscope.WithSystem(ctx, recheckTenantsReason)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pagination.TenantInfo, error) {
		bq := billingqueuedb.BillingQueueItems
		rows, err := keel.Into[recheckDueTenant](
			ctx,
			r.db.Keel(ctx),
			bq.Pick(bq.OrganizationID, bq.BusinessUnitID).
				Where(bq.RecheckDueAt.Lte(req.Now)).
				Distinct().
				Limit(req.Limit).
				AllTenants(),
		)
		if err != nil {
			return nil, err
		}

		tenants := make([]pagination.TenantInfo, 0, len(rows))
		for _, row := range rows {
			tenants = append(tenants, pagination.TenantInfo{
				OrgID: row.OrganizationID,
				BuID:  row.BusinessUnitID,
			})
		}

		return tenants, nil
	})
}

type recheckDueTenant struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}
