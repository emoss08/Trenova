package billingqueuerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/sliceutils"
)

const recheckSweepReason = "list billing queue items whose checks are missing or stale across every organization"

func (r *repository) ListRecheckCandidates(
	ctx context.Context,
	req *billingqueueports.ListRecheckCandidatesRequest,
) ([]*billingqueueports.RecheckCandidate, error) {
	ctx = dbscope.WithSystem(ctx, recheckSweepReason)

	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*billingqueueports.RecheckCandidate, error) {
			rows, err := keel.Into[billingqueueports.RecheckCandidate](
				ctx,
				r.db.Keel(ctx),
				recheckCandidates(req),
			)
			if err != nil {
				return nil, err
			}

			return sliceutils.Pointers(rows), nil
		},
	)
}

func recheckCandidates(
	req *billingqueueports.ListRecheckCandidatesRequest,
) keel.Query[*billingqueue.BillingQueueItem] {
	bq := billingqueuedb.BillingQueueItems

	return bq.Pick(bq.ID, bq.OrganizationID, bq.BusinessUnitID).
		Where(
			bq.Status.In(billingqueue.ReviewStatuses()...),
			keel.Or(bq.CheckedAt.IsNull(), bq.CheckedAt.Lt(req.StaleBefore)),
		).
		Order(bq.CheckedAt.Asc().NullsFirst(), bq.ID.Asc()).
		Limit(req.Limit).
		AllTenants()
}
