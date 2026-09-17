package carrierintelrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const verificationEntityName = "CarrierEquipmentVerification"

type verificationRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewVerificationRepository(p Params) repositories.CarrierEquipmentVerificationRepository {
	return &verificationRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.carrier-equipment-verification-repository"),
	}
}

func (r *verificationRepository) Create(
	ctx context.Context,
	entity *carrierintel.CarrierEquipmentVerification,
) (*carrierintel.CarrierEquipmentVerification, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to create carrier equipment verification", zap.Error(err))
		return nil, fmt.Errorf("create carrier equipment verification: %w", err)
	}

	return entity, nil
}

func (r *verificationRepository) GetByID(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*carrierintel.CarrierEquipmentVerification, error) {
	entity := new(carrierintel.CarrierEquipmentVerification)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierEquipmentVerificationScopeTenant(sq, tenantInfo).
				Where(buncolgen.CarrierEquipmentVerificationColumns.ID.Eq(), id)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, verificationEntityName)
	}

	return entity, nil
}

func (r *verificationRepository) ListByAssignmentIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	assignmentIDs []pulid.ID,
) ([]*carrierintel.CarrierEquipmentVerification, error) {
	if len(assignmentIDs) == 0 {
		return []*carrierintel.CarrierEquipmentVerification{}, nil
	}

	cols := buncolgen.CarrierEquipmentVerificationColumns
	entities := make([]*carrierintel.CarrierEquipmentVerification, 0, len(assignmentIDs))
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierEquipmentVerificationScopeTenant(sq, tenantInfo).
				Where(cols.CarrierAssignmentID.In(), bun.List(assignmentIDs))
		}).
		Order(cols.VerifiedAt.OrderDesc(), cols.ID.OrderDesc()).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list carrier equipment verifications", zap.Error(err))
		return nil, fmt.Errorf("list carrier equipment verifications: %w", err)
	}

	return entities, nil
}

func (r *verificationRepository) Update(
	ctx context.Context,
	entity *carrierintel.CarrierEquipmentVerification,
) (*carrierintel.CarrierEquipmentVerification, error) {
	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update carrier equipment verification", zap.Error(err))
		return nil, fmt.Errorf("update carrier equipment verification: %w", err)
	}
	if err = dberror.CheckRowsAffected(results, verificationEntityName, entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}
