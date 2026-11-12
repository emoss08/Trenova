package ediladingrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ladingproject"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ladingproject/ladingprojectdb"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ladingports"
	"github.com/emoss08/trenova/pkg/dbscope"
)

const dueInstancesReason = "list the open EDI flow instances in every tenant whose timer is due"

// ListDueInstances reads the open instances whose timer is due, oldest first,
// by the partial index on wake_at. It is the only timer behind a flow: the
// minute sweep dispatches what it returns.
func (r *flows) ListDueInstances(
	ctx context.Context,
	req *ladingports.ListDueFlowInstancesRequest,
) ([]ladingports.DueFlowInstance, error) {
	ctx = dbscope.WithSystem(ctx, dueInstancesReason)

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]ladingports.DueFlowInstance, error) {
		if req.Limit <= 0 {
			return []ladingports.DueFlowInstance{}, nil
		}

		i := ladingprojectdb.FlowInstances
		return keel.Into[ladingports.DueFlowInstance](
			ctx,
			r.db.Keel(ctx),
			i.Pick(i.ID, i.OrganizationID, i.BusinessUnitID).
				Where(
					i.WakeAt.IsNotNull(),
					i.WakeAt.Lte(req.Now),
					i.Status.Ne(ladingproject.FlowInstanceStatusClosed),
					keel.When(len(req.Exclude) > 0, i.ID.NotIn(req.Exclude...)),
				).
				Order(i.WakeAt.Asc(), i.ID.Asc()).
				Limit(req.Limit).
				AllTenants(),
		)
	})
}
