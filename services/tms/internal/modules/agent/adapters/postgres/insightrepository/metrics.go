package insightrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/ports/insightports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention/detentiondb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type MetricsParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type metricsRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewMetrics(p MetricsParams) insightports.InsightMetricsRepository {
	return &metricsRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.insight-metrics-repository"),
	}
}

const minOnTimeStops = 12

func customerOnTimeQuery(
	req insightports.InsightWindowRequest,
) *keel.Query[*shipment.Stop] {
	stp := shipmentdb.Stops
	smv := shipmentdb.ShipmentMoves
	sp := shipmentdb.Shipments
	cus := customerdb.Customers

	priorStart := req.WindowStart - (req.WindowEnd - req.WindowStart)
	current := keel.And(
		stp.ActualArrival.Gte(req.WindowStart),
		stp.ActualArrival.Lt(req.WindowEnd),
	)
	prior := keel.And(stp.ActualArrival.Gte(priorStart), stp.ActualArrival.Lt(req.WindowStart))
	onTime := keel.Int64Of(stp.ActualArrival).
		LteExpr(stp.ScheduledWindowEnd.Coalesce(stp.ScheduledWindowStart))
	currentTotal := keel.Count().Filter(current)

	return stp.Join(smv, smv.ID.EqCol(stp.ShipmentMoveID)).
		Join(sp, sp.ID.EqCol(smv.ShipmentID)).
		Join(cus, cus.ID.EqCol(sp.CustomerID)).
		Group(cus.ID, cus.Name).
		Pick(
			cus.ID,
			cus.Name,
			currentTotal,
			keel.Count().Filter(keel.And(current, onTime)),
			keel.Count().Filter(prior),
			keel.Count().Filter(keel.And(prior, onTime)),
		).
		Where(
			stp.Status.Eq(shipment.StopStatusCompleted),
			stp.Type.In(shipment.StopTypeDelivery, shipment.StopTypeSplitDelivery),
			stp.ActualArrival.IsNotNull(),
			stp.ActualArrival.Gt(0),
			stp.ScheduledWindowStart.Gt(0),
			stp.ActualArrival.Gte(priorStart),
			stp.ActualArrival.Lt(req.WindowEnd),
		).
		Having(currentTotal.Gte(minOnTimeStops))
}

func (r *metricsRepository) CustomerOnTimeComparison(
	ctx context.Context,
	req insightports.InsightWindowRequest,
) ([]insightports.CustomerOnTimeRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]insightports.CustomerOnTimeRow, error) {
			rows, err := keel.Into[insightports.CustomerOnTimeRow](
				ctx,
				r.db.Keel(ctx),
				customerOnTimeQuery(req),
			)
			if err != nil {
				r.l.Error("failed to read on-time comparison",
					zap.String("operation", "CustomerOnTimeComparison"),
					zap.Error(err))

				return nil, err
			}

			return nonNil(rows), nil
		},
	)
}

func unbilledShipmentsQuery(
	req insightports.UnbilledShipmentsRequest,
) *keel.Query[*shipment.Shipment] {
	sp := shipmentdb.Shipments
	cus := customerdb.Customers

	return sp.Join(cus, cus.ID.EqCol(sp.CustomerID)).
		Group(cus.ID, cus.Name).
		Pick(
			cus.ID,
			cus.Name,
			keel.Count(),
			keel.CastText(keel.NumericOf(sp.TotalChargeAmount.Sum()).Coalesce(0)),
			sp.ActualDeliveryDate.Min(),
		).
		Where(
			sp.ActualDeliveryDate.IsNotNull(),
			sp.ActualDeliveryDate.Gt(0),
			sp.ActualDeliveryDate.Lt(req.DeliveredBefore),
			sp.BilledAt.IsNull(),
			sp.Status.In(shipment.BillingTransferCandidateStatuses()...),
		)
}

func (r *metricsRepository) UnbilledDeliveredShipments(
	ctx context.Context,
	req insightports.UnbilledShipmentsRequest,
) ([]insightports.UnbilledShipmentRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]insightports.UnbilledShipmentRow, error) {
			rows, err := keel.Into[insightports.UnbilledShipmentRow](
				ctx,
				r.db.Keel(ctx),
				unbilledShipmentsQuery(req),
			)
			if err != nil {
				r.l.Error("failed to read unbilled shipments",
					zap.String("operation", "UnbilledDeliveredShipments"),
					zap.Error(err))

				return nil, err
			}

			return nonNil(rows), nil
		},
	)
}

func unbilledDetentionQuery(
	req insightports.InsightWindowRequest,
) *keel.Query[*detention.DetentionOccurrence] {
	dto := detentiondb.DetentionOccurrences
	loc := locationdb.Locations

	return dto.Join(loc, loc.ID.EqCol(dto.LocationID)).
		Group(loc.ID, loc.Name).
		Pick(
			loc.ID,
			loc.Name,
			keel.Count(),
			keel.CastText(keel.NumericOf(dto.BillableAmount.Sum()).Coalesce(0)),
			keel.CastInt64(keel.NumericOf(dto.RawDwellMinutes.Sum()).Coalesce(0)),
		).
		Where(
			dto.ClockStartAt.Gte(req.WindowStart),
			dto.ClockStartAt.Lt(req.WindowEnd),
			dto.Status.In(
				detention.OccurrenceStatusWaived,
				detention.OccurrenceStatusNotBillable,
				detention.OccurrenceStatusPending,
				detention.OccurrenceStatusApproved,
			),
			dto.IsOpen.Eq(false),
			keel.NumericOf(dto.BillableAmount).Gt(0),
		)
}

func (r *metricsRepository) UnbilledDetention(
	ctx context.Context,
	req insightports.InsightWindowRequest,
) ([]insightports.UnbilledDetentionRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]insightports.UnbilledDetentionRow, error) {
			rows, err := keel.Into[insightports.UnbilledDetentionRow](
				ctx,
				r.db.Keel(ctx),
				unbilledDetentionQuery(req),
			)
			if err != nil {
				r.l.Error("failed to read unbilled detention",
					zap.String("operation", "UnbilledDetention"),
					zap.Error(err))

				return nil, err
			}

			return nonNil(rows), nil
		},
	)
}

func customerEmptyMilesQuery(
	req insightports.InsightWindowRequest,
) *keel.Query[*shipment.ShipmentMove] {
	smv := shipmentdb.ShipmentMoves
	sp := shipmentdb.Shipments
	cus := customerdb.Customers

	emptyMiles := keel.Float64Of(smv.Distance.Sum().Filter(smv.Loaded.Eq(false))).Coalesce(0)
	totalMiles := keel.Float64Of(smv.Distance.Sum()).Coalesce(0)

	return smv.Join(sp, sp.ID.EqCol(smv.ShipmentID)).
		Join(cus, cus.ID.EqCol(sp.CustomerID)).
		Group(cus.ID, cus.Name).
		Pick(
			cus.ID,
			cus.Name,
			keel.CastText(emptyMiles),
			keel.CastText(totalMiles),
			keel.Count(),
			keel.CastText(keel.Sum(emptyMiles).OverWindow(keel.Window{})),
			keel.CastText(keel.Sum(totalMiles).OverWindow(keel.Window{})),
		).
		Where(
			smv.Status.Eq(shipment.MoveStatusCompleted),
			smv.Distance.IsNotNull(),
			smv.Distance.Gt(0),
			sp.ActualDeliveryDate.Gte(req.WindowStart),
			sp.ActualDeliveryDate.Lt(req.WindowEnd),
		)
}

func (r *metricsRepository) CustomerEmptyMiles(
	ctx context.Context,
	req insightports.InsightWindowRequest,
) ([]insightports.CustomerEmptyMilesRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]insightports.CustomerEmptyMilesRow, error) {
			rows, err := keel.Into[insightports.CustomerEmptyMilesRow](
				ctx,
				r.db.Keel(ctx),
				customerEmptyMilesQuery(req),
			)
			if err != nil {
				r.l.Error("failed to read empty miles",
					zap.String("operation", "CustomerEmptyMiles"),
					zap.Error(err))

				return nil, err
			}

			return nonNil(rows), nil
		},
	)
}

func expiringCredentialsQuery(
	req insightports.ExpiringCredentialsRequest,
) *keel.Query[*worker.WorkerCredential] {
	wcred := workerdb.WorkerCredentials
	wct := workerdb.WorkerCredentialTypes
	wrk := workerdb.Workers

	return wcred.Join(wct, wct.ID.EqCol(wcred.CredentialTypeID)).
		Join(wrk, wrk.ID.EqCol(wcred.WorkerID)).
		Group(wct.ID, wct.Name).
		Pick(
			wct.ID,
			wct.Name,
			keel.BoolOr(keel.BoolOf(wct.IsRequired)),
			wcred.WorkerID.CountDistinct(),
			wcred.ExpiresAt.Min(),
			wcred.WorkerID.CountDistinct().Filter(wcred.ExpiresAt.Lt(req.From)),
		).
		Where(
			wcred.ExpiresAt.IsNotNull(),
			wcred.ExpiresAt.Gt(0),
			wcred.ExpiresAt.Lt(req.Through),
			wcred.ArchivedAt.IsNull(),
			wrk.Status.Eq(domaintypes.StatusActive),
		)
}

func (r *metricsRepository) ExpiringCredentials(
	ctx context.Context,
	req insightports.ExpiringCredentialsRequest,
) ([]insightports.ExpiringCredentialRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]insightports.ExpiringCredentialRow, error) {
			rows, err := keel.Into[insightports.ExpiringCredentialRow](
				ctx,
				r.db.Keel(ctx),
				expiringCredentialsQuery(req),
			)
			if err != nil {
				r.l.Error("failed to read expiring credentials",
					zap.String("operation", "ExpiringCredentials"),
					zap.Error(err))

				return nil, err
			}

			return nonNil(rows), nil
		},
	)
}

func nonNil[T any](rows []T) []T {
	if rows == nil {
		return []T{}
	}

	return rows
}
