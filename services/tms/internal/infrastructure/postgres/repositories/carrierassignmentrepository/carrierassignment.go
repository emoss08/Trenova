package carrierassignmentrepository

import (
	"context"

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

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.CarrierAssignmentRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.carrier-assignment-repository"),
	}
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetCarrierAssignmentByIDRequest,
) (*shipment.CarrierAssignment, error) {
	cols := buncolgen.CarrierAssignmentColumns
	entity := new(shipment.CarrierAssignment)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation("Carrier").
		Relation("Accessorials").
		Relation("ShipmentMove").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierAssignmentScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.CarrierAssignmentID)
		}).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to get carrier assignment", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "Carrier assignment")
	}

	return entity, nil
}

func (r *repository) GetActiveByMoveID(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
) (*shipment.CarrierAssignment, error) {
	cols := buncolgen.CarrierAssignmentColumns
	entity := new(shipment.CarrierAssignment)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation("Carrier").
		Relation("Accessorials").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierAssignmentScopeTenant(sq, tenantInfo).
				Where(cols.ShipmentMoveID.Eq(), moveID).
				Where(cols.Status.Ne(), shipment.CarrierAssignmentStatusCanceled)
		}).
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // nil result represents an optional absence in this API
		}
		r.l.Error("failed to get carrier assignment by move", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) FindForMatching(
	ctx context.Context,
	req *repositories.FindCarrierAssignmentForMatchingRequest,
) (*shipment.CarrierAssignment, error) {
	if req.ProNumber == "" && req.ShipmentID.IsNil() {
		return nil, nil //nolint:nilnil // nothing to match without a reference
	}

	cols := buncolgen.CarrierAssignmentColumns
	entity := new(shipment.CarrierAssignment)
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation("Carrier").
		Relation("Accessorials").
		Relation("ShipmentMove").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.CarrierAssignmentScopeTenant(sq, req.TenantInfo).
				Where(cols.Status.Ne(), shipment.CarrierAssignmentStatusCanceled)
			if !req.CarrierID.IsNil() {
				sq = sq.Where(cols.CarrierID.Eq(), req.CarrierID)
			}
			if req.ProNumber != "" {
				sq = sq.Where(cols.ProNumber.Eq(), req.ProNumber)
			} else {
				sq = sq.Where(
					"EXISTS (SELECT 1 FROM shipment_moves move WHERE move.id = casn.shipment_move_id "+
						"AND move.organization_id = casn.organization_id "+
						"AND move.business_unit_id = casn.business_unit_id "+
						"AND move.shipment_id = ?)",
					req.ShipmentID,
				)
			}
			return sq
		}).
		Order(cols.CreatedAt.OrderDesc()).
		Limit(1)
	if err := query.Scan(ctx); err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // nil assignment represents an optional absence
		}
		r.l.Error("failed to find carrier assignment for matching", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) stampAccessorials(entity *shipment.CarrierAssignment) {
	for _, acc := range entity.Accessorials {
		acc.ID = ""
		acc.CarrierAssignmentID = entity.ID
		acc.OrganizationID = entity.OrganizationID
		acc.BusinessUnitID = entity.BusinessUnitID
	}
}

func (r *repository) insertAccessorials(
	ctx context.Context,
	entity *shipment.CarrierAssignment,
) error {
	if len(entity.Accessorials) == 0 {
		return nil
	}

	_, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(&entity.Accessorials).
		Returning("*").
		Exec(ctx)
	return err
}

func (r *repository) deleteAccessorials(
	ctx context.Context,
	entity *shipment.CarrierAssignment,
) error {
	_, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*shipment.CarrierAssignmentAccessorial)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.CarrierAssignmentAccessorialScopeTenantDelete(dq, pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			}).Where(buncolgen.CarrierAssignmentAccessorialColumns.CarrierAssignmentID.Eq(), entity.ID)
		}).
		Exec(ctx)
	return err
}

func (r *repository) Create(
	ctx context.Context,
	entity *shipment.CarrierAssignment,
) (*shipment.CarrierAssignment, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("shipmentMoveId", entity.ShipmentMoveID.String()),
	)

	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		log.Error("failed to create carrier assignment", zap.Error(err))
		return nil, err
	}

	r.stampAccessorials(entity)
	if err := r.insertAccessorials(ctx, entity); err != nil {
		log.Error("failed to insert carrier assignment accessorials", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *shipment.CarrierAssignment,
) (*shipment.CarrierAssignment, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update carrier assignment", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "Carrier assignment", entity.ID.String()); err != nil {
		return nil, err
	}

	if err = r.deleteAccessorials(ctx, entity); err != nil {
		log.Error("failed to delete carrier assignment accessorials", zap.Error(err))
		return nil, err
	}

	r.stampAccessorials(entity)
	if err = r.insertAccessorials(ctx, entity); err != nil {
		log.Error("failed to insert carrier assignment accessorials", zap.Error(err))
		return nil, err
	}

	return entity, nil
}
