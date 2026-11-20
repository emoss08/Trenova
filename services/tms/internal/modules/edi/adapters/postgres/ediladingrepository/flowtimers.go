package ediladingrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ladingproject"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ladingproject/ladingprojectdb"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ladingports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	dueInstancesReason   = "list the open EDI flow instances in every tenant whose timer is due"
	locateInstanceReason = "find the tenant of the EDI flow instance a development clock advance names, before its tenant is known"
)

// ListDueInstances reads the open instances whose timer is due, oldest first,
// by the partial index on wake_at. It is the only timer behind a flow: the
// minute sweep dispatches what it returns.
func (r *flows) ListDueInstances(
	ctx context.Context,
	req *ladingports.ListDueFlowInstancesRequest,
) ([]ladingports.FlowInstanceRef, error) {
	ctx = dbscope.WithSystem(ctx, dueInstancesReason)

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]ladingports.FlowInstanceRef, error) {
		if req.Limit <= 0 {
			return []ladingports.FlowInstanceRef{}, nil
		}

		i := ladingprojectdb.FlowInstances
		return keel.Into[ladingports.FlowInstanceRef](
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

func (r *flows) LocateInstance(
	ctx context.Context,
	id pulid.ID,
) (ladingports.FlowInstanceRef, error) {
	ctx = dbscope.WithSystem(ctx, locateInstanceReason)

	return dbtx.Read(ctx, r.db, func(ctx context.Context) (ladingports.FlowInstanceRef, error) {
		i := ladingprojectdb.FlowInstances
		ref, err := keel.IntoOne[ladingports.FlowInstanceRef](
			ctx,
			r.db.Keel(ctx),
			i.Pick(i.ID, i.OrganizationID, i.BusinessUnitID).
				Where(i.ID.Eq(id)).
				AllTenants(),
		)
		if err != nil {
			return ladingports.FlowInstanceRef{}, instanceErrors.Map(err, id.String())
		}
		return ref, nil
	})
}
