package ratesimulationrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/ratesimulation"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// resultInsertBatch bounds how many result rows go in one statement. A run
// carries up to a hundred thousand, and one insert that size holds locks far
// longer than the work deserves.
const resultInsertBatch = 500

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.RateSimulationRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.ratesimulation-repository"),
	}
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetRateSimulationByIDRequest,
) (*ratesimulation.RateSimulation, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.RateSimulationID.String()),
	)

	entity := new(ratesimulation.RateSimulation)
	cols := buncolgen.RateSimulationColumns

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RateSimulationScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.RateSimulationID)
		})

	if req.IncludeResults {
		q = q.Relation(buncolgen.Rel(buncolgen.RateSimulationRelations.Results))
	}

	if err := q.Scan(ctx); err != nil {
		log.Error("failed to get rate simulation", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "RateSimulation")
	}

	return entity, nil
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListRateSimulationsRequest,
) (*pagination.ListResult[*ratesimulation.RateSimulation], error) {
	cols := buncolgen.RateSimulationColumns
	entities := make([]*ratesimulation.RateSimulation, 0, req.Filter.Pagination.SafeLimit())

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(buncolgen.RateSimulationApplyTenant(req.Filter.TenantInfo))

	if req.RateAgreementID != nil && !req.RateAgreementID.IsNil() {
		q = q.Where(cols.RateAgreementID.Eq(), *req.RateAgreementID)
	}

	// Newest first: a simulation is looked at while it is fresh, and the older
	// ones are history.
	q = q.Order(cols.CreatedAt.OrderDesc()).
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset())

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		r.l.Error("failed to list rate simulations", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*ratesimulation.RateSimulation]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) ListResults(
	ctx context.Context,
	req *repositories.ListRateSimulationResultsRequest,
) (*pagination.ListResult[*ratesimulation.RateSimulationResult], error) {
	entities := make([]*ratesimulation.RateSimulationResult, 0)
	cols := buncolgen.RateSimulationResultColumns

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RateSimulationResultScopeTenant(sq, req.TenantInfo).
				Where(cols.RateSimulationID.Eq(), req.RateSimulationID)
		})

	// A targeted amendment leaves most shipments untouched, and a grid of
	// unchanged rows buries the handful that moved.
	if req.ChangedOnly {
		q = q.Where(cols.Delta.NotEq(), 0)
	}

	// Largest increases first: the shipment that will produce the phone call is
	// what somebody opened this for.
	q = q.Order(cols.Delta.OrderDesc())

	if req.Filter != nil {
		q = q.Limit(req.Filter.Pagination.SafeLimit()).
			Offset(req.Filter.Pagination.SafeOffset())
	}

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		r.l.Error("failed to list rate simulation results", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*ratesimulation.RateSimulationResult]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *ratesimulation.RateSimulation,
) (*ratesimulation.RateSimulation, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to create rate simulation", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *ratesimulation.RateSimulation,
) (*ratesimulation.RateSimulation, error) {
	ov := entity.Version
	entity.Version++

	cols := buncolgen.RateSimulationColumns

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(cols.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update rate simulation", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "RateSimulation", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

// AppendResults writes one batch of replayed shipments.
//
// A run writes as it goes rather than at the end, so a simulation that fails
// halfway still shows what it managed to price.
func (r *repository) AppendResults(
	ctx context.Context,
	results []*ratesimulation.RateSimulationResult,
) error {
	if len(results) == 0 {
		return nil
	}

	for start := 0; start < len(results); start += resultInsertBatch {
		batch := results[start:min(start+resultInsertBatch, len(results))]

		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(&batch).
			Exec(ctx); err != nil {
			r.l.Error("failed to append rate simulation results", zap.Error(err))

			return err
		}
	}

	return nil
}

// ListShipments walks the shipments a simulation replays, a page at a time.
//
// It pages by id rather than by offset. The walk runs for minutes against a
// table that is still being written to, and an offset would silently skip or
// repeat rows as shipments are created underneath it. PULIDs sort by creation,
// so ordering by id is also ordering by age.
func (r *repository) ListShipments(
	ctx context.Context,
	req *repositories.ListSimulationShipmentsRequest,
) (*repositories.SimulationShipmentPage, error) {
	cols := buncolgen.ShipmentColumns

	entities := make([]*shipment.Shipment, 0, req.Limit)

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Column("id").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.ShipmentScopeTenant(sq, req.TenantInfo)

			// The window is read against the day the freight actually moved,
			// falling back to when the shipment was created for one that never
			// recorded a ship date. Rating already picks its date this way, so
			// a simulation covers the same shipments a re-rate would.
			shipDate := bun.Ident(cols.ActualShipDate.Qualified())
			createdAt := bun.Ident(cols.CreatedAt.Qualified())

			sq = sq.
				Where("COALESCE(?, ?) >= ?", shipDate, createdAt, req.From).
				Where("COALESCE(?, ?) < ?", shipDate, createdAt, req.To)

			if !req.AfterID.IsNil() {
				sq = sq.Where(cols.ID.Gt(), req.AfterID)
			}

			return sq
		}).
		Order(cols.ID.OrderAsc()).
		Limit(req.Limit)

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list shipments for a simulation", zap.Error(err))
		return nil, err
	}

	ids := make([]pulid.ID, 0, len(entities))
	for _, entity := range entities {
		ids = append(ids, entity.ID)
	}

	result := &repositories.SimulationShipmentPage{ShipmentIDs: ids}

	// A full page means there is probably another. A short one is the end, and
	// saying so saves a round trip that would return nothing.
	if len(entities) == req.Limit && len(entities) > 0 {
		result.NextAfterID = entities[len(entities)-1].ID
	}

	return result, nil
}
