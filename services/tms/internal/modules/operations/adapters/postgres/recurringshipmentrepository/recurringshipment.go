package recurringshipmentrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/quotatx"
	"github.com/emoss08/trenova/internal/modules/identity/app/quotaservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/operations/domain/recurringshipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/recurringshipment/recurringshipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/recurringshipmentports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var seriesErrors = dberror.Mapping{
	Entity: "RecurringShipment",
	ForeignKey: []dberror.Constraint{
		{
			Name:    "fk_recurring_shipments_source_shipment",
			Field:   "sourceShipmentId",
			Message: "Source shipment does not exist in your organization",
		},
		{
			Name:    "fk_recurring_shipments_customer",
			Field:   "customerId",
			Message: "Customer does not exist in your organization",
		},
		{
			Name:    "fk_recurring_shipments_origin_location",
			Field:   "originLocationId",
			Message: "Origin location does not exist in your organization",
		},
		{
			Name:    "fk_recurring_shipments_destination_location",
			Field:   "destinationLocationId",
			Message: "Destination location does not exist in your organization",
		},
	},
}

var runErrors = dberror.Mapping{
	Entity: "RecurringShipmentRun",
	ForeignKey: []dberror.Constraint{
		{
			Name:    "fk_recurring_shipment_runs_series",
			Field:   "recurringShipmentId",
			Message: "Recurring shipment does not exist in your organization",
		},
	},
}

type Params struct {
	fx.In

	DB        *postgres.Connection
	Logger    *zap.Logger
	Generator seqgen.Generator
	Quota     platformplanports.QuotaGuard `optional:"true"`
}

type repository struct {
	db        *postgres.Connection
	l         *zap.Logger
	generator seqgen.Generator
	quota     platformplanports.QuotaGuard
}

func New(p Params) recurringshipmentports.RecurringShipmentRepository {
	return &repository{
		db:        p.DB,
		l:         p.Logger.Named("postgres.recurring-shipment-repository"),
		generator: p.Generator,
		quota:     quotaservice.OrUnlimited(p.Quota),
	}
}

type seriesQuery = keel.Query[recurringshipment.RecurringShipment]

func withLane(q *seriesQuery) *seriesQuery {
	rsh := recurringshipmentdb.RecurringShipments
	return q.Load(rsh.Customer(), rsh.OriginLocation(), rsh.DestinationLocation())
}

func (r *repository) List(
	ctx context.Context,
	req *recurringshipmentports.ListRecurringShipmentsRequest,
) (*pagination.ListResult[*recurringshipment.RecurringShipment], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*recurringshipment.RecurringShipment], error) {
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				recurringshipmentdb.RecurringShipments.Query(),
				&listquery.Request[recurringshipment.RecurringShipment]{
					Filter: req.Filter,
					Shape:  withLane,
				},
			)
			if err != nil {
				r.l.Error("failed to list recurring shipments", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *recurringshipmentports.ListRecurringShipmentConnectionRequest,
) (*pagination.CursorListResult[*recurringshipment.RecurringShipment], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*recurringshipment.RecurringShipment], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				recurringshipmentdb.RecurringShipments.Query(),
				&listquery.Request[recurringshipment.RecurringShipment]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.RecurringShipmentColumns,
					Shape:   withLane,
				},
			)
			if err != nil {
				r.l.Error("failed to list recurring shipment connection", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req *recurringshipmentports.GetRecurringShipmentByIDRequest,
) (*recurringshipment.RecurringShipment, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*recurringshipment.RecurringShipment, error) {
			rsh := recurringshipmentdb.RecurringShipments
			q := withLane(rsh.Where(rsh.ID.Eq(req.ID))).Load(rsh.EnteredBy())
			if req.ExpandDetails {
				q = q.Load(rsh.SourceShipment())
			}

			entity, err := q.One(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error(
					"failed to get recurring shipment",
					zap.String("id", req.ID.String()),
					zap.Error(err),
				)
				return nil, seriesErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *recurringshipment.RecurringShipment,
) (*recurringshipment.RecurringShipment, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*recurringshipment.RecurringShipment, error) {
			if err := r.applyDerivedFields(ctx, entity); err != nil {
				return nil, err
			}

			if err := r.insertSeries(ctx, entity); err != nil {
				r.l.Error("failed to create recurring shipment", zap.Error(err))
				return nil, seriesErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) insertSeries(
	ctx context.Context,
	entity *recurringshipment.RecurringShipment,
) error {
	return quotatx.Run(ctx, r.db, r.quota, func(c context.Context) error {
		return recurringshipmentdb.RecurringShipments.Create(c, r.db.Keel(c), entity)
	}, platformplanports.QuotaRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		Meter:    platformcatalog.MeterRecurringShipmentSeries,
		Quantity: 1,
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *recurringshipment.RecurringShipment,
) (*recurringshipment.RecurringShipment, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*recurringshipment.RecurringShipment, error) {
			if err := r.applyDerivedFields(ctx, entity); err != nil {
				return nil, err
			}

			err := recurringshipmentdb.RecurringShipments.Save(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error(
					"failed to update recurring shipment",
					zap.String("id", entity.ID.String()),
					zap.Error(err),
				)
				return nil, seriesErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) UpdateStatus(
	ctx context.Context,
	req *recurringshipmentports.UpdateRecurringShipmentStatusRequest,
) (*recurringshipment.RecurringShipment, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*recurringshipment.RecurringShipment, error) {
			log := r.l.With(
				zap.String("operation", "UpdateStatus"),
				zap.String("id", req.RecurringShipmentID.String()),
				zap.String("status", string(req.Status)),
			)

			entity, err := r.GetByID(ctx, &recurringshipmentports.GetRecurringShipmentByIDRequest{
				ID:         req.RecurringShipmentID,
				TenantInfo: req.TenantInfo,
			})
			if err != nil {
				return nil, err
			}

			entity.Version = req.Version
			if err = entity.ApplyStatus(req.Status, timeutils.NowUnix()); err != nil {
				log.Error("failed to compute next occurrence on resume", zap.Error(err))
				return nil, err
			}

			rsh := recurringshipmentdb.RecurringShipments
			if err = rsh.Save(
				ctx,
				r.db.Keel(ctx),
				entity,
				rsh.Status,
				rsh.NextOccurrenceAt,
				rsh.ConsecutiveFailures,
			); err != nil {
				log.Error("failed to update recurring shipment status", zap.Error(err))
				return nil, seriesErrors.Map(err, req.RecurringShipmentID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *recurringshipmentports.RecurringShipmentSelectOptionsRequest,
) (*pagination.ListResult[*recurringshipment.RecurringShipment], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*recurringshipment.RecurringShipment], error) {
			rsh := recurringshipmentdb.RecurringShipments
			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				rsh.Pick(rsh.ID, rsh.Name, rsh.Status, rsh.CronExpression),
				req.SelectQueryRequest,
				func(s string) keel.Pred { return rsh.Name.ContainsFold(s) },
			)
		},
	)
}

func (r *repository) Match(
	ctx context.Context,
	req *recurringshipmentports.MatchRecurringShipmentsRequest,
) ([]*recurringshipment.RecurringShipment, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*recurringshipment.RecurringShipment, error) {
			rsh := recurringshipmentdb.RecurringShipments
			entities, err := withLane(rsh.Where(
				rsh.Status.Eq(recurringshipment.StatusActive),
				rsh.CustomerID.Eq(req.CustomerID),
				rsh.OriginLocationID.Eq(req.OriginLocationID),
				rsh.DestinationLocationID.Eq(req.DestinationLocationID),
			)).
				Order(rsh.CreatedAt.Asc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to match recurring shipments", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func laneStop(
	sp *shipmentdb.ShipmentsTable,
	alias string,
	locationID pulid.ID,
	types ...shipment.StopType,
) keel.Pred {
	sm := shipmentdb.ShipmentMoves.As(alias + "_sm")
	st := shipmentdb.Stops.As(alias + "_stp")

	return keel.Exists(sm.
		Join(st, st.ShipmentMoveID.EqCol(sm.ID)).
		Pick(keel.Raw("1")).
		Where(
			sm.ShipmentID.EqCol(sp.ID),
			st.Type.In(types...),
			st.LocationID.Eq(locationID),
		))
}

func (r *repository) DetectLanePattern(
	ctx context.Context,
	req *recurringshipmentports.DetectLanePatternRequest,
) (*recurringshipmentports.LanePatternSummary, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*recurringshipmentports.LanePatternSummary, error) {
			lookbackDays := req.LookbackDays
			if lookbackDays <= 0 {
				lookbackDays = 90
			}

			sp := shipmentdb.Shipments
			summary, err := keel.IntoOne[recurringshipmentports.LanePatternSummary](
				ctx,
				r.db.Keel(ctx),
				sp.Pick(
					keel.Raw("count(*)::int"),
					keel.DynOf(sp.CreatedAt).Min().Coalesce(0),
					keel.DynOf(sp.CreatedAt).Max().Coalesce(0),
				).Where(
					sp.CustomerID.Eq(req.CustomerID),
					sp.Status.Ne(shipment.StatusCanceled),
					sp.CreatedAt.Gte(timeutils.DaysAgoUnix(lookbackDays)),
					laneStop(sp, "pick", req.OriginLocationID,
						shipment.StopTypePickup, shipment.StopTypeSplitPickup),
					laneStop(sp, "drop", req.DestinationLocationID,
						shipment.StopTypeDelivery, shipment.StopTypeSplitDelivery),
				),
			)
			if err != nil {
				r.l.Error("failed to detect lane pattern", zap.Error(err))
				return nil, err
			}

			minShipments := req.MinShipments
			if minShipments <= 0 {
				minShipments = 3
			}

			if summary.ShipmentCount < minShipments {
				return nil, nil //nolint:nilnil // no pattern is a valid non-error outcome
			}

			return &summary, nil
		},
	)
}

func (r *repository) ListDue(
	ctx context.Context,
	req *recurringshipmentports.ListDueRecurringShipmentsRequest,
) ([]*recurringshipment.RecurringShipment, error) {
	ctx = dbscope.WithSystem(ctx, "list recurring shipments due across every organization")
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*recurringshipment.RecurringShipment, error) {
			rsh := recurringshipmentdb.RecurringShipments
			q := rsh.AllTenants().
				Where(
					rsh.Status.Eq(recurringshipment.StatusActive),
					rsh.AutoGenerate.IsTrue(),
					rsh.NextOccurrenceAt.IsNotNull(),
					keel.Raw(
						"? - (?::bigint * 86400) <= ?",
						rsh.NextOccurrenceAt,
						rsh.LeadTimeDays,
						req.Now,
					),
				).
				Order(rsh.NextOccurrenceAt.Asc())
			if req.Limit > 0 {
				q = q.Limit(req.Limit)
			}

			entities, err := q.All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list due recurring shipments", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *repository) ListRuns(
	ctx context.Context,
	req *recurringshipmentports.ListRecurringShipmentRunsRequest,
) (*pagination.ListResult[*recurringshipment.RecurringShipmentRun], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*recurringshipment.RecurringShipmentRun], error) {
			rsr := recurringshipmentdb.RecurringShipmentRuns
			info := pagination.Info{}
			if req.Filter != nil {
				info = req.Filter.Pagination
			}

			runs := rsr.Where(rsr.RecurringShipmentID.Eq(req.RecurringShipmentID))
			var items []*recurringshipment.RecurringShipmentRun
			var total int64
			err := keel.Batch(ctx, r.db.Keel(ctx),
				keel.ListInto(runs.Clone().
					Load(rsr.GeneratedShipment(), rsr.TriggeredBy()).
					Order(rsr.OccurrenceAt.Desc()).
					Limit(info.SafeLimit()).
					Offset(info.SafeOffset()), &items),
				keel.CountInto(runs, &total),
			)
			if err != nil {
				r.l.Error(
					"failed to list recurring shipment runs",
					zap.String("recurringShipmentId", req.RecurringShipmentID.String()),
					zap.Error(err),
				)
				return nil, err
			}

			return &pagination.ListResult[*recurringshipment.RecurringShipmentRun]{
				Items: items,
				Total: int(total),
			}, nil
		},
	)
}
