package bankreceiptcandidaterepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt/bankreceiptdb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/entitlement"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/workqueue/workqueuedb"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/timeutils"
)

const sweepReason = "queue every organization's unapplied bank receipts for their candidates again"

func (r *repository) QueueUnapplied(
	ctx context.Context,
	req *bankreceiptports.QueueUnappliedRequest,
) (int64, error) {
	ctx = dbscope.WithSystem(ctx, sweepReason)

	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		br := bankreceiptdb.BankReceipts
		wq := workqueuedb.Items
		entitled, err := r.entitlements.Organizations(
			ctx,
			br.OrganizationID.Col,
			entitlement.For(platformcatalog.FeatureAccounting),
		)
		if err != nil {
			return 0, err
		}
		now := keel.Int64(timeutils.NowUnix())
		queued, err := wq.UpsertFrom(
			ctx,
			r.db.Keel(ctx),
			br.AllTenants().
				Pick(br.OrganizationID, br.BusinessUnitID, keel.Text(req.Kind), br.ID, now, now, now).
				Where(unapplied(br), entitled),
			keel.OnConflict(wq.OrganizationID, wq.BusinessUnitID, wq.Kind, wq.SubjectID).
				DoNothing(),
			wq.OrganizationID,
			wq.BusinessUnitID,
			wq.Kind,
			wq.SubjectID,
			wq.DueAt,
			wq.CreatedAt,
			wq.UpdatedAt,
		)
		if err != nil {
			return 0, fmt.Errorf("queue unapplied bank receipts: %w", err)
		}

		return queued, nil
	})
}
