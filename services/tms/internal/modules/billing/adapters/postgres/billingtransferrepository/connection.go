package billingtransferrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingtransfer"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingtransfer/billingtransferdb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingtransferports"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

func runItemScope(req *billingtransferports.ListBillingTransferRunItemsRequest) keel.Pred {
	item := billingtransferdb.BillingTransferRunItems
	return keel.And(
		item.RunID.Eq(req.RunID),
		keel.When(len(req.Statuses) > 0, item.Status.In(req.Statuses...)),
	)
}

func (r *repository) ListItems(
	ctx context.Context,
	req *billingtransferports.ListBillingTransferRunItemsRequest,
) (*pagination.CursorListResult[*billingtransfer.BillingTransferRunItem], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*billingtransfer.BillingTransferRunItem], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				billingtransferdb.BillingTransferRunItems.Where(runItemScope(req)),
				&listquery.Request[billingtransfer.BillingTransferRunItem]{
					Filter: req.Filter,
					Cursor: req.Cursor,
				},
			)
			if err != nil {
				r.l.Error("failed to list billing transfer run items", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}
