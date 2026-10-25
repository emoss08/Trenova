package ratesimulationrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratesimulation"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratesimulation/ratesimulationdb"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratesimulationports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var simulationErrors = dberror.Mapping{Entity: "RateSimulation"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) ratesimulationports.RateSimulationRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.ratesimulation-repository"),
	}
}

func (r *repository) GetByID(
	ctx context.Context,
	req *ratesimulationports.GetRateSimulationByIDRequest,
) (*ratesimulation.RateSimulation, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*ratesimulation.RateSimulation, error) {
		rs := ratesimulationdb.RateSimulations
		q := rs.Where(rs.ID.Eq(req.RateSimulationID))
		if req.IncludeResults {
			q = q.Load(rs.Results())
		}

		entity, err := q.One(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get rate simulation",
				zap.String("id", req.RateSimulationID.String()),
				zap.Error(err),
			)
			return nil, simulationErrors.Map(err, req.RateSimulationID.String())
		}

		return entity, nil
	})
}

func simulationsQuery(
	req *ratesimulationports.ListRateSimulationsRequest,
) *keel.Query[*ratesimulation.RateSimulation] {
	rs := ratesimulationdb.RateSimulations
	if req.RateAgreementID != nil && !req.RateAgreementID.IsNil() {
		return rs.Where(rs.RateAgreementID.Eq(*req.RateAgreementID))
	}

	return rs.Query()
}

func (r *repository) List(
	ctx context.Context,
	req *ratesimulationports.ListRateSimulationsRequest,
) (*pagination.ListResult[*ratesimulation.RateSimulation], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*ratesimulation.RateSimulation], error) {
			rs := ratesimulationdb.RateSimulations
			page := req.Filter.Pagination
			rows := simulationsQuery(req).
				Order(rs.CreatedAt.Desc()).
				Limit(page.SafeLimit()).
				Offset(page.SafeOffset())

			var items []*ratesimulation.RateSimulation
			var total int64
			if err := keel.Batch(
				ctx,
				r.db.Keel(ctx),
				keel.ListInto(rows, &items),
				keel.CountInto(simulationsQuery(req), &total),
			); err != nil {
				r.l.Error("failed to list rate simulations", zap.Error(err))
				return nil, err
			}

			return &pagination.ListResult[*ratesimulation.RateSimulation]{
				Items: items,
				Total: int(total),
			}, nil
		},
	)
}

func resultsQuery(
	req *ratesimulationports.ListRateSimulationResultsRequest,
) *keel.Query[*ratesimulation.RateSimulationResult] {
	res := ratesimulationdb.RateSimulationResults

	return res.Where(
		res.RateSimulationID.Eq(req.RateSimulationID),
		keel.When(req.ChangedOnly, res.Delta.Ne(decimal.Zero)),
	)
}

func (r *repository) ListResults(
	ctx context.Context,
	req *ratesimulationports.ListRateSimulationResultsRequest,
) (*pagination.ListResult[*ratesimulation.RateSimulationResult], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*ratesimulation.RateSimulationResult], error) {
			res := ratesimulationdb.RateSimulationResults
			rows := resultsQuery(req).Order(res.Delta.Desc())

			var items []*ratesimulation.RateSimulationResult
			var total int64
			read := keel.AllInto(rows, &items)
			if req.Filter != nil {
				page := req.Filter.Pagination
				read = keel.ListInto(rows.Limit(page.SafeLimit()).Offset(page.SafeOffset()), &items)
			}
			if err := keel.Batch(
				ctx,
				r.db.Keel(ctx),
				read,
				keel.CountInto(resultsQuery(req), &total),
			); err != nil {
				r.l.Error("failed to list rate simulation results", zap.Error(err))
				return nil, err
			}

			return &pagination.ListResult[*ratesimulation.RateSimulationResult]{
				Items: items,
				Total: int(total),
			}, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *ratesimulation.RateSimulation,
) (*ratesimulation.RateSimulation, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*ratesimulation.RateSimulation, error) {
		if err := ratesimulationdb.RateSimulations.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create rate simulation", zap.Error(err))
			return nil, simulationErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *ratesimulation.RateSimulation,
) (*ratesimulation.RateSimulation, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*ratesimulation.RateSimulation, error) {
		if err := ratesimulationdb.RateSimulations.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update rate simulation", zap.Error(err))
			return nil, simulationErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

// AppendResults writes one batch of replayed shipments.
//
// A run writes as it goes rather than at the end, so a simulation that fails
// halfway still shows what it managed to price.
func (r *repository) AppendResults(
	ctx context.Context,
	results []*ratesimulation.RateSimulationResult,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(results) == 0 {
			return nil
		}

		err := ratesimulationdb.RateSimulationResults.CreateMany(ctx, r.db.Keel(ctx), results)
		if err != nil {
			r.l.Error("failed to append rate simulation results", zap.Error(err))
			return err
		}

		return nil
	})
}

func simulationShipmentsQuery(
	req *ratesimulationports.ListSimulationShipmentsRequest,
) *keel.Query[*shipment.Shipment] {
	sp := shipmentdb.Shipments
	shipDate := sp.ActualShipDate.Coalesce(sp.CreatedAt)

	return sp.Pick(sp.ID).
		Where(
			shipDate.Gte(req.From),
			shipDate.Lt(req.To),
			keel.When(!req.AfterID.IsNil(), sp.ID.Gt(req.AfterID)),
		).
		Order(sp.ID.Asc()).
		Limit(req.Limit)
}

// ListShipments walks the shipments a simulation replays, a page at a time.
//
// It pages by id rather than by offset. The walk runs for minutes against a
// table that is still being written to, and an offset would silently skip or
// repeat rows as shipments are created underneath it. PULIDs sort by creation,
// so ordering by id is also ordering by age.
func (r *repository) ListShipments(
	ctx context.Context,
	req *ratesimulationports.ListSimulationShipmentsRequest,
) (*ratesimulationports.SimulationShipmentPage, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*ratesimulationports.SimulationShipmentPage, error) {
			ids, err := keel.Into[pulid.ID](ctx, r.db.Keel(ctx), simulationShipmentsQuery(req))
			if err != nil {
				r.l.Error("failed to list shipments for a simulation", zap.Error(err))
				return nil, err
			}

			result := &ratesimulationports.SimulationShipmentPage{ShipmentIDs: ids}

			// A full page means there is probably another. A short one is the end, and
			// saying so saves a round trip that would return nothing.
			if len(ids) == req.Limit && len(ids) > 0 {
				result.NextAfterID = ids[len(ids)-1]
			}

			return result, nil
		},
	)
}
