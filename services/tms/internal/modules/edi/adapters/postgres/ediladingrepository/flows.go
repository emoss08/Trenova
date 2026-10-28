//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package ediladingrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage/edimessagedb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ladingproject"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ladingproject/ladingprojectdb"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ladingports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

const (
	maxFlowEventsListed    = 200
	maxOpenInstancesListed = 50
)

var (
	instanceErrors = dberror.Mapping{Entity: "FlowInstance"}
	messageErrors  = dberror.Mapping{Entity: "EDIMessage"}
)

type flows struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewFlows(p Params) ladingports.FlowRepository {
	return &flows{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-flow-repository"),
	}
}

func (r *flows) GetInstance(
	ctx context.Context,
	req ladingports.GetFlowInstanceRequest,
) (*ladingproject.FlowInstance, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*ladingproject.FlowInstance, error) {
		i := ladingprojectdb.FlowInstances
		instance, err := i.Where(i.ID.Eq(req.ID)).Load(i.Partner()).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, instanceErrors.Map(err, req.ID.String())
		}
		return instance, nil
	})
}

func (r *flows) GetInstanceByKey(
	ctx context.Context,
	req ladingports.GetFlowInstanceByKeyRequest,
) (*ladingproject.FlowInstance, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*ladingproject.FlowInstance, error) {
		i := ladingprojectdb.FlowInstances
		instance, err := i.Where(
			i.EDIPartnerID.Eq(req.PartnerID),
			i.Flow.Eq(req.Flow),
			i.Key.Eq(req.Key),
		).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, instanceErrors.Map(err, req.Key)
		}
		return instance, nil
	})
}

// OpenInstance returns the instance for the partner, flow and key, creating it when it does not
// exist: two messages for the same key that arrive together open one instance.
func (r *flows) OpenInstance(
	ctx context.Context,
	instance *ladingproject.FlowInstance,
) (*ladingproject.FlowInstance, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*ladingproject.FlowInstance, error) {
		db := r.db.Keel(ctx)
		i := ladingprojectdb.FlowInstances
		if len(instance.State) == 0 {
			instance.State = []byte("{}")
		}
		if _, err := i.CreateOrSkip(
			ctx,
			db,
			[]*ladingproject.FlowInstance{instance},
			i.OrganizationID,
			i.BusinessUnitID,
			i.EDIPartnerID,
			i.Flow,
			i.Key,
		); err != nil {
			return nil, instanceErrors.Map(err, instance.Key)
		}
		opened, err := i.Where(
			i.EDIPartnerID.Eq(instance.EDIPartnerID),
			i.Flow.Eq(instance.Flow),
			i.Key.Eq(instance.Key),
		).One(ctx, db)
		if err != nil {
			return nil, instanceErrors.Map(err, instance.Key)
		}
		return opened, nil
	})
}

func (r *flows) SaveInstance(
	ctx context.Context,
	instance *ladingproject.FlowInstance,
) (*ladingproject.FlowInstance, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*ladingproject.FlowInstance, error) {
		i := ladingprojectdb.FlowInstances
		err := i.Save(
			ctx,
			r.db.Keel(ctx),
			instance,
			i.ProjectID,
			i.VersionID,
			i.State,
			i.Status,
			i.Reason,
			i.WakeAt,
			i.ShipmentID,
			i.TransferID,
			i.LastEventAt,
			i.HeldAt,
			i.ResolvedByID,
			i.ResolvedAt,
		)
		if err != nil {
			return nil, instanceErrors.Map(err, instance.ID.String())
		}
		return instance, nil
	})
}

func (r *flows) AppendEvent(ctx context.Context, event *ladingproject.FlowEvent) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(event.Detail) == 0 {
			event.Detail = []byte("{}")
		}
		if event.Actions == nil {
			event.Actions = []lading.FlowAction{}
		}
		return ladingprojectdb.FlowEvents.Create(ctx, r.db.Keel(ctx), event)
	})
}

func instanceFilters(req *ladingports.ListFlowInstancesRequest) []keel.Pred {
	i := ladingprojectdb.FlowInstances
	search := ""
	if req.Filter != nil {
		search = req.Filter.Query
	}
	return []keel.Pred{
		keel.When(req.PartnerID.IsNotNil(), i.EDIPartnerID.Eq(req.PartnerID)),
		keel.When(req.ShipmentID.IsNotNil(), i.ShipmentID.Eq(req.ShipmentID)),
		keel.When(req.TransferID.IsNotNil(), i.TransferID.Eq(req.TransferID)),
		keel.When(len(req.Statuses) > 0, i.Status.In(req.Statuses...)),
		keel.When(search != "", keel.Or(
			i.Key.ContainsFold(search),
			i.Flow.ContainsFold(search),
		)),
	}
}

func (r *flows) ListInstances(
	ctx context.Context,
	req *ladingports.ListFlowInstancesRequest,
) (*pagination.ListResult[*ladingproject.FlowInstance], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*ladingproject.FlowInstance], error) {
			i := ladingprojectdb.FlowInstances
			q := i.Omit(i.State).
				Where(instanceFilters(req)...).
				Load(i.Partner()).
				Order(i.UpdatedAt.Desc())
			return listquery.Offset(ctx, r.db.Keel(ctx), q, req.Filter.Pagination)
		},
	)
}

// ListOpenInstances is every flow instance not closed that follows a shipment or a transfer,
// at most a page of them.
func (r *flows) ListOpenInstances(
	ctx context.Context,
	req *ladingports.ListShipmentFlowInstancesRequest,
) ([]*ladingproject.FlowInstance, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*ladingproject.FlowInstance, error) {
		if req.ShipmentID.IsNil() && req.TransferID.IsNil() {
			return []*ladingproject.FlowInstance{}, nil
		}
		i := ladingprojectdb.FlowInstances
		return i.Where(
			keel.When(req.ShipmentID.IsNotNil(), i.ShipmentID.Eq(req.ShipmentID)),
			keel.When(req.TransferID.IsNotNil(), i.TransferID.Eq(req.TransferID)),
			i.Status.Ne(ladingproject.FlowInstanceStatusClosed),
		).
			Order(i.CreatedAt.Asc()).
			Limit(maxOpenInstancesListed).
			List(ctx, r.db.Keel(ctx))
	})
}

func (r *flows) ListInstancesCursor(
	ctx context.Context,
	req *ladingports.ListFlowInstancesRequest,
) (*pagination.CursorListResult[*ladingproject.FlowInstance], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*ladingproject.FlowInstance], error) {
			i := ladingprojectdb.FlowInstances
			return listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				i.Omit(i.State).Where(instanceFilters(req)...),
				&listquery.Request[ladingproject.FlowInstance]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape: func(q *keel.Query[*ladingproject.FlowInstance]) *keel.Query[*ladingproject.FlowInstance] {
						return q.Load(i.Partner())
					},
				},
			)
		},
	)
}

func (r *flows) ListEvents(
	ctx context.Context,
	req ladingports.ListFlowEventsRequest,
) ([]*ladingproject.FlowEvent, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*ladingproject.FlowEvent, error) {
		e := ladingprojectdb.FlowEvents
		limit := req.Limit
		if limit <= 0 || limit > maxFlowEventsListed {
			limit = maxFlowEventsListed
		}
		return e.Where(e.InstanceID.Eq(req.InstanceID)).
			Order(e.CreatedAt.Desc(), e.ID.Desc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
	})
}

func (r *flows) GetSendMessage(
	ctx context.Context,
	req ladingports.GetFlowSendMessageRequest,
) (*edimessage.EDIMessage, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*edimessage.EDIMessage, error) {
		m := edimessagedb.EDIMessages
		message, err := m.Omit(m.Lineage).Where(
			m.FlowInstanceID.Eq(req.InstanceID),
			m.FlowSendID.Eq(req.SendID),
		).Order(m.GeneratedAt.Desc()).First(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, messageErrors.Map(err, req.SendID)
		}
		return message, nil
	})
}
